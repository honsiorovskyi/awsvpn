package openvpn

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"syscall"
	"time"
)

var (
	samlHandshakeRespRegex = regexp.MustCompile("(AUTH_FAILED,CRV1.+)\n")
	sidRegexp              = regexp.MustCompile("AUTH_FAILED,CRV1:.+?:(.+?):")
	authLinkRegexp         = regexp.MustCompile("(https{0,1}://.+)[:\n]{0,1}.*$")
)

func parseHandshakeResponse(b []byte) (string, string, error) {
	resp := samlHandshakeRespRegex.FindStringSubmatch(string(b))
	if len(resp) < 2 {
		return "", "", fmt.Errorf("recieved empty response")
	}

	sid := sidRegexp.FindStringSubmatch(resp[1])
	if len(sid) < 2 {
		return "", "", fmt.Errorf("unable to parse SID")
	}

	authLink := authLinkRegexp.FindStringSubmatch(resp[1])
	if len(authLink) < 2 {
		return "", "", fmt.Errorf("unable to parse auth link")
	}

	return sid[1], authLink[1], nil
}

func Handshake(ctx context.Context, c Config) (string, string, error) {
	conf, err := c.pipe()
	if err != nil {
		return "", "", fmt.Errorf("handshake: %w", err)
	}
	defer conf.Close()

	auth, err := newTextSourcePipe("N/A\nACS::35001\n")
	if err != nil {
		return "", "", fmt.Errorf("handshake: %w", err)
	}
	defer auth.Close()

	cmdCtx, killProcess := context.WithCancel(context.Background())
	defer killProcess()

	cmd := exec.CommandContext(cmdCtx, c.Command,
		"--config", "/dev/fd/3",
		"--verb", "3",
		"--proto", c.Proto,
		"--remote", c.Remote, strconv.Itoa(c.Port),
		"--auth-user-pass", "/dev/fd/4",
		"--connect-retry-max", "1",
	)

	// cmd.Stderr = os.Stderr
	// cmd.Stdout = os.Stdout
	cmd.ExtraFiles = []*os.File{conf.source, auth.source}

	go func() {
		<-ctx.Done()
		cmd.Process.Signal(syscall.SIGTERM)

		time.Sleep(c.TerminationTimeout)
		killProcess()
	}()

	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", "", fmt.Errorf("handshake error: %w", err)
	}

	sid, authLink, err := parseHandshakeResponse(out)
	if err != nil {
		return "", "", fmt.Errorf("handshake error: %w", err)
	}

	return sid, authLink, nil
}
