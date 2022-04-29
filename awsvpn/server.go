package awsvpn

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
)

func (a *App) httpServer(ctx context.Context) {
	mux := http.NewServeMux()

	mux.HandleFunc("/shutdown", a.httpMakeCommandHandler(CmdShutdown))
	mux.HandleFunc("/connect", a.httpMakeCommandHandler(CmdConnect))
	mux.HandleFunc("/disconnect", a.httpMakeCommandHandler(CmdDisconnect))
	mux.HandleFunc("/status", a.httpStatusHandler)
	mux.HandleFunc("/", a.httpSAMLHandler)

	srv := http.Server{
		Addr:    "127.0.0.1:35001",
		Handler: mux,
	}

	go func() {
		err := srv.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			a.busCh <- FatalErr(err)
		}
	}()

	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), terminationTimeout)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		a.busCh <- FatalErr(err)
	}

	log.Println("[AWS VPN] HTTP server terminated")
}

func serveCloser(w http.ResponseWriter) {
	fmt.Fprintf(w, `
		<!DOCTYPE html>
		<html>
		<head>
			<style>
				html, body {
					box-sizing: border-box;
					height: 100%%;
					margin: 0;
				}

				body {
					display: grid;
					justify-content: center;
					align-content: center;

					font-family: sans-serif;
				}
			</style>
		</head>
		<body>
			<p>It's now safe to turn off your computer.</p>
			<script language="javascript">
				open('', '_self').close()
			</script>
		</body>
		</html>
	`)
}

func (a *App) httpMakeCommandHandler(cmd int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		a.busCh <- Command(cmd)
		w.Header().Add("Access-Control-Allow-Origin", r.Header.Get("Origin"))
		w.WriteHeader(204)
	}
}

func (a *App) httpStatusHandler(w http.ResponseWriter, r *http.Request) {
	switch a.st.Get() {
	case StatusDisconnected:
		w.Write([]byte("{\"status\": \"disconnected\"}\n"))
	case StatusConnecting:
		w.Write([]byte("{\"status\": \"connecting\"}\n"))
	case StatusConnected:
		w.Write([]byte("{\"status\": \"connected\"}\n"))
	case StatusDisconnecting:
		w.Write([]byte("{\"status\": \"disconnecting\"}\n"))
	default:
		w.WriteHeader(500)
	}
}

func (a *App) httpSAMLHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		if err := r.ParseForm(); err != nil {
			a.busCh <- NonFatalErr(fmt.Errorf("handler: %w", err))
			return
		}

		SAMLResponse := r.FormValue("SAMLResponse")
		if len(SAMLResponse) == 0 {
			a.busCh <- NonFatalErr(fmt.Errorf("handler: SAMLResponse field is empty or not exists"))
			return
		}

		a.dataCh <- url.QueryEscape(SAMLResponse)
		serveCloser(w)
	default:
		serveCloser(w)
	}
}
