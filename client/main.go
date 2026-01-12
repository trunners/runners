package main

import (
	"context"
	"io"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/trunners/runners/logger"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())

	log := logger.New()
	ctx = logger.WithLogger(ctx, log)

	serverAddress := os.Getenv("SERVER_ADDRESS")
	if serverAddress == "" {
		log.ErrorContext(ctx, "SERVER_ADDRESS environment variable is required")
		os.Exit(1)
	}

	openssh, err := dial(ctx, "localhost:22")
	if err != nil {
		log.ErrorContext(ctx, "Could not connect to openssh", "error", err)
		os.Exit(1)
	}
	log.InfoContext(ctx, "Connected to openssh", "address", openssh.RemoteAddr())

	server, err := dial(ctx, serverAddress)
	if err != nil {
		log.ErrorContext(ctx, "Could not connect to server", "error", err)
		return
	}
	log.InfoContext(ctx, "Connected to remote server", "address", server.RemoteAddr())

	log.InfoContext(ctx, "Piping data", "server", server.RemoteAddr(), "openssh", openssh.RemoteAddr())
	wg := sync.WaitGroup{}

	wg.Go(func() {
		_, err = io.Copy(server, openssh)
		if err != nil {
			log.ErrorContext(ctx, "Failed piping data from server to openssh", "error", err)
		}
	})

	wg.Go(func() {
		_, ioerr := io.Copy(openssh, server)
		if ioerr != nil {
			log.ErrorContext(ctx, "Failed piping data from openssh to server", "error", err)
		}
	})

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		cancel()
	}()

	wg.Wait()
}

func dial(ctx context.Context, address string) (net.Conn, error) {
	log := logger.FromContext(ctx)

	dialer := &net.Dialer{}
	server, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return nil, err
	}

	go func() {
		<-ctx.Done()
		err = server.Close()
		if err != nil {
			log.ErrorContext(ctx, "Error closing connection", "address", address, "error", err)
		}
	}()

	return server, nil
}
