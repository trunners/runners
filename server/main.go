package main

import (
	"context"
	"errors"
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

	workflowID := os.Getenv("WORKFLOW_ID")
	workflowOwner := os.Getenv("WORKFLOW_OWNER")
	workflowRepository := os.Getenv("WORKFLOW_REPOSITORY")
	workflowToken := os.Getenv("WORKFLOW_TOKEN")

	workflow, err := NewWorkflow(workflowID, workflowOwner, workflowRepository, workflowToken)
	if err != nil {
		log.Fatalf("Failed to create workflow: %v", err)
	}
	log.Printf("Initialized workflow: %s\n", workflow.ID)

	conns := make(chan net.Conn)
	cfg := net.ListenConfig{}
	wg := sync.WaitGroup{}

	wg.Go(func() {
		workflowListen(ctx, cfg, conns)
	})

	wg.Go(func() {
		deviceListen(ctx, cfg, conns, *workflow)
	})

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		cancel()
	}()

	log.Println("Connect: ssh -p 8080 localhost")
	wg.Wait()
}

func workflowListen(ctx context.Context, cfg net.ListenConfig, conns chan net.Conn) {
	listener, err := cfg.Listen(ctx, "tcp", ":8081")
	if err != nil {
		log.Fatal(err)
	}
	go func() {
		<-ctx.Done()
		lerr := listener.Close()
		if lerr != nil {
			log.Println("Error closing workflow listener:", lerr)
		}
	}()

	log.Println("Listening for workflow connections on port 8081")

	for {
		var conn net.Conn
		conn, err = listener.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				break
			}

			log.Println("Error accepting workflow connection:", err)
			continue
		}

		log.Println("New workflow connected:", conn.RemoteAddr())
		conns <- conn
	}
}

func deviceListen(ctx context.Context, cfg net.ListenConfig, conns chan net.Conn, w Workflow) {
	listener, err := cfg.Listen(ctx, "tcp", ":8080")
	if err != nil {
		log.Fatal(err)
	}
	go func() {
		<-ctx.Done()
		lerr := listener.Close()
		if lerr != nil {
			log.Println("Error closing device listener:", lerr)
		}
	}()

	log.Println("Listening for device connections on port 8080")

	for {
		var conn net.Conn
		conn, err = listener.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				break
			}

			log.Println("Error accepting device connection:", err)
			continue
		}

		go deviceHandle(ctx, conn, conns, w)
	}
}

func deviceHandle(ctx context.Context, deviceConn net.Conn, workflowConns chan net.Conn, w Workflow) {
	log.Println("New device connected:", deviceConn.RemoteAddr())
	defer deviceConn.Close()

	log.Println("Starting workflow...")
	err := w.start(ctx, "ubuntu-24.04", "main")
	if err != nil {
		log.Println("Failed to start workflow:", err)
		return
	}

	log.Println("Waiting for workflow connection...")
	workflowConn := <-workflowConns

	log.Printf("Piping data between %s and %s...\n", deviceConn.RemoteAddr(), workflowConn.RemoteAddr())
	wg := sync.WaitGroup{}

	wg.Go(func() {
		_, ioerr := io.Copy(deviceConn, workflowConn)
		if ioerr != nil {
			log.Println("Error piping data:", err)
		}
	})

	wg.Go(func() {
		_, ioerr := io.Copy(workflowConn, deviceConn)
		if ioerr != nil {
			log.Println("Error piping data:", err)
		}
	})

	wg.Wait()

	err = workflowConn.Close()
	if err != nil {
		log.Println("Error closing workflow connection:", err)
	}

	log.Println("Connection terminated")
}
