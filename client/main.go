package main

import (
	"context"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())

	serverAddress := os.Getenv("SERVER_ADDRESS")
	if serverAddress == "" {
		log.Fatal("SERVER_ADDRESS environment variable is required")
	}

	dialer := &net.Dialer{}

	serverConn, err := dialer.DialContext(ctx, "tcp", serverAddress)
	if err != nil {
		log.Println("Error connecting to server:", err)
		return
	}
	go func() {
		<-ctx.Done()
		conerr := serverConn.Close()
		if conerr != nil {
			log.Println("Error closing server connection:", conerr)
		}
	}()
	log.Printf("Connected to remote server: %s\n", serverConn.RemoteAddr())

	sshConn, err := dialer.DialContext(ctx, "tcp", "localhost:22")
	if err != nil {
		log.Println("Error connecting to openssh server:", err)
		return
	}
	go func() {
		<-ctx.Done()
		conerr := sshConn.Close()
		if conerr != nil {
			log.Println("Error closing server connection:", conerr)
		}
	}()
	log.Printf("Connected to openssh server: %s\n", sshConn.RemoteAddr())

	log.Printf("Piping data between %s and %s...\n", serverConn.RemoteAddr(), sshConn.RemoteAddr())
	wg := sync.WaitGroup{}

	wg.Go(func() {
		_, ioerr := io.Copy(serverConn, sshConn)
		if ioerr != nil {
			log.Println("Error piping data:", err)
		}
	})

	wg.Go(func() {
		_, ioerr := io.Copy(sshConn, serverConn)
		if ioerr != nil {
			log.Println("Error piping data:", err)
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
