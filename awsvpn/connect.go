package awsvpn

import (
	"awsvpn/openvpn"
	"context"
	"fmt"
	"log"
	"os/exec"
)

func (a *App) open(url string) error {
	return exec.Command("sh", "-c", fmt.Sprintf("%s %s", a.browserCmd, url)).Start()
}

func (a *App) connect(ctx context.Context) error {
	log.Println("[AWS VPN] Performing SAML handshake...")
	sid, authLink, err := openvpn.Handshake(ctx, a.cfg)
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}

	log.Println("[AWS VPN] Reponse recieved, opening authentication link...")
	a.open(authLink)

	log.Println("[AWS VPN] Waiting for SAML Response...")
	samlReponse := <-a.dataCh

	notifyCh := make(chan int)
	go func() {
		for {
			switch <-notifyCh {
			case openvpn.ConnEstablished:
				a.busCh <- Status(StatusConnected)
			case openvpn.ConnClosed:
				a.busCh <- Status(StatusDisconnecting)
			}
		}
	}()

	log.Println("[AWS VPN] SAML Response received, connecting...")
	if err := openvpn.Connect(ctx, a.cfg, sid, samlReponse, notifyCh); err != nil {
		return fmt.Errorf("connect: %w", err)
	}

	log.Println("[AWS VPN] VPN connection terminated")
	return nil
}
