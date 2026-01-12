package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/trunners/runners/logger"
	"github.com/trunners/runners/server/config"
	"github.com/trunners/runners/server/github"
	"github.com/trunners/runners/server/pool"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())

	log := logger.New()
	ctx = logger.WithLogger(ctx, log)

	config, err := config.Load()
	if err != nil {
		log.ErrorContext(ctx, "Failed to load config", "error", err)
		os.Exit(1)
	}

	gh, err := github.New(config.GithubToken)
	if err != nil {
		log.ErrorContext(ctx, "Failed to create GitHub client", "error", err)
		os.Exit(1)
	}

	wg := sync.WaitGroup{}
	for _, wf := range config.Workflows {
		wg.Go(func() {
			workflowCtx := logger.Append(ctx, slog.String("workflow", wf.ID))
			workflow(workflowCtx, wf, gh, config.Port)
		})
	}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		cancel()
	}()

	wg.Wait()
}

func workflow(ctx context.Context, w config.Workflow, gh github.Github, port int) {
	log := logger.FromContext(ctx)

	p, err := pool.Start(ctx, port)
	if err != nil {
		log.ErrorContext(ctx, "Failed to create connection pool", "error", err)
		return
	}

	log.InfoContext(ctx, "Listening for ssh connections", "port", port)
	err = p.Wait(ctx)
	if err != nil {
		return
	}

	log.InfoContext(ctx, "Starting workflow")
	err = gh.Workflow(ctx, w.ID, w.Owner, w.Repository, w.Ref, w.RunsOn, fmt.Sprintf("%s:%d", w.Hostname, port))
	if err != nil {
		log.ErrorContext(ctx, "Failed to start workflow", "error", err)
		return
	}

	log.InfoContext(ctx, "Connection established, bridging")
	err = p.Bridge(ctx)
	if err != nil {
		log.ErrorContext(ctx, "Failed to bridge connections", "error", err)
		return
	}

	log.InfoContext(ctx, "Connection terminated")
}
