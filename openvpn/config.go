package openvpn

import (
	"fmt"
	"time"
)

type Config struct {
	Command       string
	CACertificate string
	Proto         string
	Remote        string
	Port          int

	TerminationTimeout time.Duration
}

func (c Config) pipe() (*pipe, error) {
	return newTextSourcePipe(fmt.Sprintf(`
client
dev tun
proto tcp
nobind
persist-key
persist-tun
remote-cert-tls server
cipher AES-256-GCM
<ca>
%s
</ca>

auth-nocache
reneg-sec 0`, c.CACertificate))
}
