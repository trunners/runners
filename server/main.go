package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	"golang.org/x/crypto/ssh"

	"github.com/trunners/runners/logger"
	"github.com/trunners/runners/server/config"
	"github.com/trunners/runners/server/github"
	"github.com/trunners/runners/server/pool"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())

	log := logger.New()
	ctx = logger.WithLogger(ctx, log)

	config, err := config.Load(ctx)
	if err != nil {
		log.ErrorContext(ctx, "Failed to load config", "error", err)
		os.Exit(1)
	}

	gh, err := github.New(config.GithubToken)
	if err != nil {
		log.ErrorContext(ctx, "Failed to create GitHub client", "error", err)
		os.Exit(1)
	}

	p, err := pool.Start(ctx, config.Port)
	if err != nil {
		log.ErrorContext(ctx, "Failed to create connection pool", "error", err)
		os.Exit(1)
	}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		cancel()
	}()

	for {
		select {
		case <-ctx.Done():
			return
		default:
			serve(ctx, config, gh, p)
		}
	}
}

func serve(ctx context.Context, cfg *config.Config, gh github.Github, p *pool.Pool) {
	log := logger.FromContext(ctx)

	log.InfoContext(ctx, "Waiting for SSH connection")
	serverTCP, err := p.Next(ctx, pool.TypeSSH)
	if err != nil {
		return
	}
	defer serverTCP.Close()

	log.InfoContext(ctx, "Creating SSH server")
	serverSSH, serverChans, serverReqs, err := ssh.NewServerConn(serverTCP, cfg.Server)
	if err != nil {
		log.ErrorContext(ctx, "Failed to create SSH server", "error", err)
		return
	}
	go ssh.DiscardRequests(serverReqs)

	log.InfoContext(ctx, "SSH connection established", "user", serverSSH.User())

	w, ok := cfg.Workflows[serverSSH.User()]
	if !ok {
		log.ErrorContext(ctx, "No workflow found for user", "user", serverSSH.User())
		return
	}

	log.InfoContext(ctx, "Starting workflow")
	err = gh.Workflow(ctx, w.ID, w.Owner, w.Repository, w.Ref, w.RunsOn, fmt.Sprintf("%s:%d", cfg.Host, cfg.Port))
	if err != nil {
		log.ErrorContext(ctx, "Failed to start workflow", "error", err)
		return
	}

	log.InfoContext(ctx, "Waiting for TCP connection")
	var clientTCP net.Conn
	clientTCP, err = p.Next(ctx, pool.TypeTCP)
	if err != nil {
		return
	}
	defer clientTCP.Close()

	log.InfoContext(ctx, "Creating SSH client")
	clientSSH, clientChans, clientReqs, err := ssh.NewClientConn(clientTCP, "localhost:22", cfg.Client)
	if err != nil {
		log.ErrorContext(ctx, "Failed to create SSH client", "error", err)
		return
	}
	client := ssh.NewClient(clientSSH, clientChans, clientReqs)

	log.InfoContext(ctx, "Connecting server to client")
	channel(ctx, serverChans, client)

	log.InfoContext(ctx, "Connection terminated")
}

func channel(ctx context.Context, channels <-chan ssh.NewChannel, client *ssh.Client) {
	log := logger.FromContext(ctx)

	for channel := range channels {
		go func() {
			err := pipe(ctx, channel, client)
			if err != nil {
				log.ErrorContext(ctx, "Failed to pipe channel", "error", err)
			}
		}()
	}
}
