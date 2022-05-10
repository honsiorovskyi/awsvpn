package main

import (
	"awsvpn/awsvpn"
	"awsvpn/openvpn"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"regexp"
	"runtime"
	"strconv"
)

const (
	defaultPort  = 443
	defaultProto = "udp"
)

var (
	confRemoteRegexp = regexp.MustCompile(`(?m)^remote\s+(\S+)`)
	confPortRegexp   = regexp.MustCompile(`(?m)(^remote\s+\S+\s+|port\s+)([0-9]+)`)
	confProtoRegexp  = regexp.MustCompile(`(?m)(^remote\s+\S+\s+[0-9]+\s+|proto\s+)(tcp[46]?|udp[46]?)`)
	caSectionRegexp  = regexp.MustCompile(`(?s)<ca>(.*)</ca>`)
)

func defaultOpenVPNConfig() string {
	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatalln(err)
	}

	return path.Join(home, ".awsvpn.ovpn")
}

func defaultBrowser() string {
	switch runtime.GOOS {
	case "windows": // no idea what else to use here :)
		return "explorer.exe"
	case "darwin": // better override it with "open -g -a YOUR_BROWSER_NAME"
		return "open"
	default: // let's try good old xdg
		return "xdg-open"
	}
}

func defaultOpenVPNClient() string {
	switch runtime.GOOS {
	case "windows": // no idea how this thing works :)
		return "C:\\Program Files\\OpenVPN\\OpenVPN.exe"
	case "darwin": // using native AWS client
		return "/Applications/AWS VPN Client/AWS VPN Client.app/Contents/Resources/openvpn/acvc-openvpn"
	default: // assuming it's a *nix OS and we have a customly build OpenVPN client
		return "/opt/aws-openvpn/bin/openvpn"
	}
}

var (
	configPath     = flag.String("config", defaultOpenVPNConfig(), "Absolute path to the config file (OpenVPN-compatible)")
	browserCmd     = flag.String("browser", defaultBrowser(), "Command used to open the authorization URL")
	openVPNCommand = flag.String("openvpn", defaultOpenVPNClient(), "Absolute path to the OpenVPN binary")
)

func parseOpenVPNConfig(cmd string, cfgPath string) (*openvpn.Config, error) {
	f, err := os.Open(cfgPath)
	if err != nil {
		return nil, fmt.Errorf("config: %w", err)
	}
	defer f.Close()

	b, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("config: %w", err)
	}

	// ca
	ca := caSectionRegexp.FindSubmatch(b)
	if ca == nil {
		return nil, fmt.Errorf("config: <ca> section not found in the config")
	}

	// remote
	remote := confRemoteRegexp.FindSubmatch(b)
	if remote == nil {
		return nil, fmt.Errorf("config: remote not found in the config")
	}

	cfg := &openvpn.Config{
		Command:       cmd,
		CACertificate: string(ca[1]),
		Remote:        string(remote[1]),
		Port:          defaultPort,
		Proto:         defaultProto,
	}

	// port
	if p := confPortRegexp.FindSubmatch(b); p != nil {
		port, err := strconv.Atoi(string(p[2]))
		if err != nil {
			return nil, fmt.Errorf("config: invalid port: %s", string(p[2]))
		}
		cfg.Port = port
	}

	// proto
	if p := confProtoRegexp.FindSubmatch(b); p != nil {
		cfg.Proto = string(p[2])
	}

	return cfg, nil
}

func main() {
	flag.Parse()

	cfg, err := parseOpenVPNConfig(*openVPNCommand, *configPath)
	if err != nil {
		log.Fatal(err)
	}

	app, err := awsvpn.NewApp(*cfg, *browserCmd)
	if err != nil {
		log.Fatalln(err)
	}

	app.Start()
}
