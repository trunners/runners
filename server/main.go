package main

import (
	"context"
	"errors"
	"fmt"
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

	workflowPort := os.Getenv("WORKFLOW_PORT")
	if workflowPort == "" {
		log.Println("WORKFLOW_PORT not set, defaulting to 8081")
		workflowPort = "8081"
	}

	devicePort := os.Getenv("DEVICE_PORT")
	if devicePort == "" {
		log.Println("DEVICE_PORT not set, defaulting to 8080")
		devicePort = "8080"
	}

	workflowID := os.Getenv("WORKFLOW_ID")
	if workflowID == "" {
		log.Fatal("WORKFLOW_ID not set")
	}

	workflowOwner := os.Getenv("WORKFLOW_OWNER")
	if workflowOwner == "" {
		log.Fatal("WORKFLOW_OWNER not set")
	}

	workflowRepository := os.Getenv("WORKFLOW_REPOSITORY")
	if workflowRepository == "" {
		log.Fatal("WORKFLOW_REPOSITORY not set")
	}

	githubToken := os.Getenv("GITHUB_TOKEN")
	if githubToken == "" {
		log.Fatal("GITHUB_TOKEN not set")
	}

	workflow, err := NewWorkflow(workflowID, workflowOwner, workflowRepository, githubToken)
	if err != nil {
		log.Fatalf("Failed to create workflow: %v", err)
	}
	log.Printf("Using workflow: %s\n", workflow.ID)

	conns := make(chan net.Conn)
	cfg := net.ListenConfig{}
	wg := sync.WaitGroup{}

	wg.Go(func() {
		workflowListen(ctx, cfg, workflowPort, conns)
	})

	wg.Go(func() {
		deviceListen(ctx, cfg, devicePort, conns, *workflow)
	})

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		cancel()
	}()

	log.Printf("Connect: ssh -p %s %s\n", devicePort, getOutboundIP(ctx).String())
	wg.Wait()
}

func workflowListen(ctx context.Context, cfg net.ListenConfig, port string, conns chan net.Conn) {
	listener, err := cfg.Listen(ctx, "tcp", fmt.Sprintf(":%s", port))
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

func deviceListen(ctx context.Context, cfg net.ListenConfig, port string, conns chan net.Conn, w Workflow) {
	listener, err := cfg.Listen(ctx, "tcp", fmt.Sprintf(":%s", port))
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

// get preferred outbound ip of this machine.
func getOutboundIP(ctx context.Context) net.IP {
	dialer := &net.Dialer{}
	conn, err := dialer.DialContext(ctx, "udp", "8.8.8.8:80")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	localAddr := conn.LocalAddr()
	udpAddr, ok := localAddr.(*net.UDPAddr)
	if !ok {
		log.Printf("Failed to get local UDP address: %v\n", localAddr)
		return net.IPv4zero
	}

	return udpAddr.IP
}
