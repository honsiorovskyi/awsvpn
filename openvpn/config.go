package openvpn

import "time"

type Config struct {
	Command    string
	ConfigFile string
	Proto      string
	Remote     string
	Port       string

	TerminationTimeout time.Duration
}
