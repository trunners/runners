package main

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"

	"github.com/creack/pty"
	"golang.org/x/crypto/ssh"

	"github.com/trunners/runners/client/config"
	"github.com/trunners/runners/logger"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())

	log := logger.New()
	ctx = logger.WithLogger(ctx, log)
	cfg := config.Load()

	server, err := dial(ctx, cfg)
	if err != nil {
		log.ErrorContext(ctx, "Could not connect to server", "error", err)
		os.Exit(1)
	}
	log.InfoContext(ctx, "Connected to remote server", "address", server.RemoteAddr())

	_, err = server.Write([]byte("TCP"))
	if err != nil {
		log.ErrorContext(ctx, "Could not notify server of new connection", "error", err)
		os.Exit(1)
	}

	sshServer, chans, reqs, err := ssh.NewServerConn(server, cfg.Server)
	if err != nil {
		log.ErrorContext(ctx, "Could not establish SSH connection", "error", err)
		os.Exit(1)
	}

	log.InfoContext(ctx, "New SSH connection", "client", sshServer.RemoteAddr())

	wg := sync.WaitGroup{}
	wg.Go(func() {
		ssh.DiscardRequests(reqs)
	})

	wg.Go(func() {
		channel(ctx, chans, cfg.Shell)
	})

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		cancel()
	}()

	wg.Wait()
}

func channel(ctx context.Context, chans <-chan ssh.NewChannel, shell string) {
	for channel := range chans {
		go pipe(ctx, channel, shell)
	}
}

func pipe(ctx context.Context, channel ssh.NewChannel, shell string) {
	log := logger.FromContext(ctx)

	if t := channel.ChannelType(); t != "session" {
		log.WarnContext(ctx, "Unknown channel type", "type", t)

		err := channel.Reject(ssh.UnknownChannelType, fmt.Sprintf("unknown channel type: %s", t))
		if err != nil {
			log.WarnContext(ctx, "Could not reject channel", "error", err)
		}

		return
	}

	connection, requests, err := channel.Accept()
	if err != nil {
		log.ErrorContext(ctx, "Could not accept channel", "error", err)
		return
	}

	shellcmd := exec.CommandContext(ctx, shell)

	cleanup := func() {
		err = connection.Close()
		if err != nil {
			log.ErrorContext(ctx, "Could not close connection", "error", err)
		}

		_, err = shellcmd.Process.Wait()
		if err != nil {
			log.ErrorContext(ctx, "Could not wait for bash process", "error", err)
		}
	}

	shellpty, err := pty.Start(shellcmd)
	if err != nil {
		log.ErrorContext(ctx, "Could not start pty", "error", err)
		cleanup()
		return
	}

	var once sync.Once
	var wg sync.WaitGroup

	wg.Go(func() {
		_, err = io.Copy(connection, shellpty)
		if err != nil {
			log.WarnContext(ctx, "Error copying from shell to connection", "error", err)
		}

		once.Do(cleanup)
	})

	wg.Go(func() {
		_, err = io.Copy(shellpty, connection)
		if err != nil {
			log.WarnContext(ctx, "Error copying from connection to shell", "error", err)
		}

		once.Do(cleanup)
	})

	wg.Go(func() {
		request(ctx, shellpty, requests)
	})

	wg.Wait()
	log.InfoContext(ctx, "Session closed")
}

func request(ctx context.Context, shell *os.File, requests <-chan *ssh.Request) {
	log := logger.FromContext(ctx)
	var err error

	for req := range requests {
		switch req.Type {
		case "shell":
			// We only accept the default shell
			// (i.e. no command in the Payload)
			if len(req.Payload) == 0 {
				err = req.Reply(true, nil)
				if err != nil {
					log.ErrorContext(ctx, "Could not reply to shell request", "error", err)
				}
			}

		case "pty-req":
			termLen := req.Payload[3]
			w, h := parseDims(req.Payload[termLen+4:])
			SetWinsize(shell.Fd(), w, h)
			// Responding true (OK) here will let the client
			// know we have a pty ready for input
			err = req.Reply(true, nil)
			if err != nil {
				log.ErrorContext(ctx, "Could not reply to pty request", "error", err)
			}

		case "window-change":
			w, h := parseDims(req.Payload)
			SetWinsize(shell.Fd(), w, h)
		}
	}
}

func dial(ctx context.Context, cfg *config.Config) (net.Conn, error) {
	log := logger.FromContext(ctx)

	server, err := cfg.Dialer.DialContext(ctx, "tcp", cfg.Address)
	if err != nil {
		return nil, err
	}

	go func() {
		<-ctx.Done()
		err = server.Close()
		if err != nil {
			log.ErrorContext(ctx, "Error closing connection", "address", cfg.Address, "error", err)
		}
	}()

	return server, nil
}
