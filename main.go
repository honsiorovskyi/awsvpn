package main

import (
	"awsvpn/awsvpn"
	"awsvpn/openvpn"
	"flag"
	"log"
	"os"
	"path"
)

func defaultOpenVPNConfig() string {
	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatalln(err)
	}

	return path.Join(home, ".awsvpn.ovpn")
}

var (
	openVPNCommand = flag.String("openvpn", "/opt/openvpn/sbin/openvpn", "Absolute path to the OpenVPN binary")
	openVPNConfig  = flag.String("openvpn-config", defaultOpenVPNConfig(), "[TEMP] Absolute path to the OpenVPN configuration file")
	openVPNRemote  = flag.String("openvpn-remote", "cvpn-endpoint-YOUR-ID.prod.clientvpn.eu-west-1.amazonaws.com", "[TEMP] AWS OpenVPN server endpoint")
	browserCmd     = flag.String("browser", "open", "Command used to open the authorization URL")
)

func main() {
	flag.Parse()

	cfg := openvpn.Config{
		Command:    *openVPNCommand,
		ConfigFile: *openVPNConfig,
		Proto:      "udp",
		Remote:     *openVPNRemote,
		Port:       "443",
	}

	app, err := awsvpn.NewApp(cfg, *browserCmd)
	if err != nil {
		log.Fatalln(err)
	}

	app.Start()
}
