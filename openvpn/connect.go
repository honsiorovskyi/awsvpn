package openvpn

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const (
	ConnEstablished = iota
	ConnClosed
)

func Connect(ctx context.Context, c Config, sid string, samlResponse string, notifyCh chan int) error {
	conf, err := c.pipe()
	if err != nil {
		return fmt.Errorf("openvpn: %w", err)
	}
	defer conf.Close()

	auth, err := newTextSourcePipe(fmt.Sprintf("N/A\nCRV1::%s::%s\n", sid, samlResponse))
	if err != nil {
		return fmt.Errorf("openvpn: %w", err)
	}
	defer auth.Close()

	cmdCtx, killProcess := context.WithCancel(context.Background())
	defer killProcess()

	cmd := exec.CommandContext(cmdCtx, c.Command,
		"--config", "/dev/fd/3",
		"--verb", "3",
		"--auth-nocache",
		"--inactive", "3600",
		"--proto", c.Proto,
		"--remote", c.Remote, strconv.Itoa(c.Port),
		"--auth-user-pass", "/dev/fd/4",
	)

	cmd.ExtraFiles = []*os.File{conf.source, auth.source}
	cmd.Stderr = os.Stderr

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("openvpn: %w", err)
	}
	defer stdout.Close()

	go func() {
		sc := bufio.NewScanner(stdout)
		for sc.Scan() {
			fmt.Println(sc.Text())

			switch {
			case strings.Contains(sc.Text(), "Initialization Sequence Completed"):
				notifyCh <- ConnEstablished
			case strings.Contains(sc.Text(), "Closing TUN/TAP interface"):
				notifyCh <- ConnClosed
			}
		}

		if err := sc.Err(); err != nil {
			log.Printf("openvpn: %v", err)
		}
	}()

	go func() {
		<-ctx.Done()
		cmd.Process.Signal(syscall.SIGTERM)

		time.Sleep(c.TerminationTimeout)
		killProcess()
	}()

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("openvpn: %w", err)
	}

	return nil
}
