package awsvpn

import (
	"crypto/rand"
	"encoding/base32"
	"fmt"
	"net"
)

func randString(l int) (string, error) {
	b := make([]byte, l)
	_, err := rand.Read(b)
	if err != nil {
		return "", fmt.Errorf("randString: %w", err)
	}

	return base32.StdEncoding.EncodeToString(b), nil
}

func resolveRemoteIP(srv string) (string, error) {
	rnd, err := randString(8)
	if err != nil {
		return "", fmt.Errorf("resolveRemoteIP: %w", err)
	}

	ips, err := net.LookupIP(fmt.Sprintf("%s.%s", rnd, srv))
	if err != nil {
		return "", fmt.Errorf("resolveRemoteIP: %w", err)
	}

	if len(ips) < 1 {
		return "", fmt.Errorf("resolveRemoteIP: A records not found")
	}

	return ips[0].String(), nil
}
