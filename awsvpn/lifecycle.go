package awsvpn

import (
	"awsvpn/openvpn"
	"context"
	"log"
	"net"
	"time"
)

const terminationTimeout = 5 * time.Second

type App struct {
	busCh  chan Message
	dataCh chan string

	s          scheduler
	st         status
	cfg        openvpn.Config
	browserCmd string

	shutdown   func()
	disconnect func()
}

func (a *App) handleShutdown() {
	log.Println("[AWS VPN] Shutdown requested. Terminating...")
	a.shutdown()
}

func (a *App) handleDisconnect() {
	switch a.st.Get() {
	case StatusConnected, StatusConnecting, StatusDisconnecting:
		log.Println("[AWS VPN] Disconnecting...")
		a.disconnect()
	default:
		log.Println("[AWS VPN] Disconnect requested, but no connection established!")
	}
}

func (a *App) handleConnect(srvCtx context.Context) {
	switch a.st.Get() {
	case StatusDisconnected:
		log.Println("[AWS VPN] Connecting...")
		a.st.Update(StatusConnecting)

		// launch connection in background
		var ctx context.Context
		ctx, a.disconnect = context.WithCancel(srvCtx)

		a.s.Run(func() {
			if err := a.connect(ctx); err != nil {
				a.busCh <- NonFatalErr(err)
			}

			a.busCh <- Command(StatusDisconnected)
		})
	default:
		log.Println("[AWS VPN] Connect requested, but already connected or in progress!")
	}
}

func (a *App) lifecycle(srvCtx context.Context) {
	for {
		msg := <-a.busCh
		switch msg.Type() {
		case CmdShutdown:
			a.handleShutdown()
		case CmdDisconnect:
			a.handleDisconnect()
		case CmdConnect:
			a.handleConnect(srvCtx)
		case StatusDisconnected, StatusConnected, StatusConnecting, StatusDisconnecting:
			a.st.Update(msg.Type())
		default:
			log.Println("[AWS VPN] ", msg.Error())
		}
	}
}

func NewApp(cfg openvpn.Config, browserCmd string) (*App, error) {
	if net.ParseIP(cfg.Remote) == nil {
		var err error
		cfg.Remote, err = resolveRemoteIP(cfg.Remote)
		if err != nil {
			return nil, err
		}
	}

	if cfg.TerminationTimeout == 0 {
		cfg.TerminationTimeout = terminationTimeout
	}

	return &App{
		busCh:  make(chan Message),
		dataCh: make(chan string),

		cfg:        cfg,
		st:         status{status: StatusDisconnected},
		browserCmd: browserCmd,
	}, nil
}

func (a *App) Start() {
	var srvCtx context.Context
	srvCtx, a.shutdown = context.WithCancel(context.Background())

	// start lifecycle management
	go a.lifecycle(srvCtx)

	// start http server
	log.Println("[AWS VPN] Starting HTTP server...")
	a.s.Run(func() {
		a.httpServer(srvCtx)
	})

	// block until all tasks are finished
	a.s.Wait()
	log.Println("[AWS VPN] All tasks finished")
}
