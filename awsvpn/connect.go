package awsvpn

import (
	"context"
	"fmt"
	"log"

	"awsvpn/openvpn"
)

func (a *App) connect(ctx context.Context) error {
	log.Println("[AWS VPN] Performing SAML handshake...")

	authParams, err := openvpn.Handshake(ctx, a.cfg)
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}

	log.Println("[AWS VPN] Response with the auth link received")
	a.authLinkCh <- authParams.HandshakeResponse.AuthLink

	log.Println("[AWS VPN] Waiting for SAML Response...")
	var samlReponse string
	select {
	case <-ctx.Done():
		return fmt.Errorf("connect: %w", ctx.Err())
	case samlReponse = <-a.dataCh:
		log.Println("[AWS VPN] SAML Response received")
	}

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

	log.Println("[AWS VPN] Connecting...")
	if err := openvpn.Connect(ctx, a.cfg, authParams, samlReponse, notifyCh); err != nil {
		return fmt.Errorf("connect: %w", err)
	}

	log.Println("[AWS VPN] VPN connection terminated")

	return nil
}
