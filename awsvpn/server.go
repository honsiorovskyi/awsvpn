package awsvpn

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os/exec"
	"time"
)

func (a *App) open(url string) error {
	return exec.Command("sh", "-c", fmt.Sprintf("%s %s", a.browserCmd, url)).Start()
}

func (a *App) httpServer(ctx context.Context) {
	mux := http.NewServeMux()

	mux.HandleFunc("/connect", a.httpConnectHandler)
	mux.HandleFunc("/disconnect", a.httpMakeCommandHandler(CmdDisconnect))
	mux.HandleFunc("/shutdown", a.httpMakeCommandHandler(CmdShutdown))
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

func (a *App) httpMakeCommandHandler(cmd int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		a.busCh <- Command(cmd)
		w.Header().Add("Access-Control-Allow-Origin", r.Header.Get("Origin"))
		w.WriteHeader(204)
	}
}

func msg(w http.ResponseWriter, m string, args ...interface{}) {
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

				.msg {
					max-width: 800px;
				}

				li {
					line-height: 2em;
					text-indent: -2em;
					padding-left: 2em;
					list-style-type: none;
				}

				code {
					font-weight: bold;
				}
			</style>
		</head>
		<body>
			<div class="msg">
				<p>%s</p>
			</div>
		</body>
		</html>
	`, fmt.Sprintf(m, args...))
}

const (
	redirectURLParam   = "redirect_url"
	redirectDelayParam = "redirect_delay"
	relayStateParam    = "RelayState"
)

const (
	msgInProgress             = `Connect/disconnect in progress. You can close this tab or <a href="/disconnect">disconnect</a>.`
	msgInProgressWithRedirect = `Connect/disconnect in progress. You can <a href="%s">continue to the requested page manually</a> if you want.`
	msgAlreadyConnected       = `Already connected. You can close the tab.`
	msgCantParseAuthLink      = `Couldn't parse authorization link: %q`
	msgDone                   = `Done! You can now close this tab or <a href="/disconnect">disconnect</a>`
	msgHelp                   = `
		Here's how to use it:
		<ul>
			<li><code>/connect</code> — normal connect using browser command configured
			<li><code>/connect?method=redirect&redirect_url=REDIRECT_URL&redirect_delay=DELAY</code> — redirect to the authorization link and then to <code>REDIRECT_URL</code>, optionally wait <code>DELAY</code> before the second redirect (to let the connection "make its way through")
			<li><code>/disconnect</code> — disconnect or abort connection attempt
			<li><code>/status</code> — connection status (JSON)
			<li><code>/shutdown</code> — shutdown the daemon
		</ul>
	`
)

func getRedirect(q url.Values) (string, time.Duration, bool) {
	if !q.Has(redirectURLParam) {
		return "", 0, false
	}

	d, err := time.ParseDuration(q.Get(redirectDelayParam))
	if err != nil {
		d = 0
	}

	return q.Get(redirectURLParam), d, true
}

func (a *App) httpConnectHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Access-Control-Allow-Origin", r.Header.Get("Origin"))

	switch r.URL.Query().Get("method") {
	case "redirect":
		switch a.st.Get() {
		case StatusConnected:
			if redir, _, ok := getRedirect(r.URL.Query()); ok {
				w.Header().Add("Location", redir)
				w.WriteHeader(302)
				return
			}

			msg(w, msgAlreadyConnected)
			return

		case StatusDisconnected:
			a.busCh <- Command(CmdConnect)

			// get auth link and parse it
			authLink := <-a.authLinkCh
			u, err := url.Parse(authLink)
			if err != nil {
				msg(w, msgCantParseAuthLink, authLink)
				w.WriteHeader(500)
				return
			}

			// if there's redirect requested, add it as RelayState
			if redir, d, ok := getRedirect(r.URL.Query()); ok {
				data := url.Values{}
				data.Add(redirectURLParam, redir)
				data.Add(redirectDelayParam, d.String())

				q := u.Query()
				q.Set(relayStateParam, data.Encode())
				u.RawQuery = q.Encode()
			}

			// redirect to the auth link
			w.Header().Add("Location", u.String())
			w.WriteHeader(302)
		default:
			if r, _, ok := getRedirect(r.URL.Query()); ok {
				msg(w, msgInProgressWithRedirect, r)
				return
			}

			msg(w, msgInProgress)
			return
		}

	default:
		a.busCh <- Command(CmdConnect)

		log.Println("[AWS VPN] Interactive connect: opening auth link in browser...")
		a.open(<-a.authLinkCh)

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

		if v, err := url.ParseQuery(r.FormValue(relayStateParam)); err == nil {
			if redir, d, ok := getRedirect(v); ok {
				time.Sleep(d)

				w.Header().Set("Location", redir)
				w.WriteHeader(302)
				return
			}
		}

		msg(w, msgDone)
		return
	default:
		msg(w, msgHelp)
	}
}
