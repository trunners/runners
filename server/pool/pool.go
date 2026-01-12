package pool

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"

	"github.com/trunners/runners/logger"
)

type Pool struct {
	port        int
	listener    net.Listener
	notify      chan struct{}
	connections []net.Conn
	mu          sync.Mutex
}

func Start(ctx context.Context, port int) (*Pool, error) {
	cfg := net.ListenConfig{}
	listener, err := cfg.Listen(ctx, "tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return nil, err
	}

	p := &Pool{
		port:        port,
		listener:    listener,
		notify:      make(chan struct{}),
		connections: []net.Conn{},
		mu:          sync.Mutex{},
	}

	// start listening for connections
	go p.listen(ctx)

	// close listener on context done
	go func() {
		<-ctx.Done()
		p.close(ctx)
	}()

	return p, nil
}

func (p *Pool) listen(ctx context.Context) {
	log := logger.FromContext(ctx)

	for {
		conn, err := p.listener.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				break
			}

			log.ErrorContext(ctx, "Could not accept connection", "error", err)
			continue
		}

		p.add(ctx, conn)
	}
}

func (p *Pool) add(ctx context.Context, conn net.Conn) {
	log := logger.FromContext(ctx)

	p.mu.Lock()
	defer p.mu.Unlock()

	if len(p.connections) >= 2 { //nolint:mnd // max 2
		log.WarnContext(
			ctx,
			"Rejecting connection, pool is full",
			"port",
			p.port,
			"remote",
			conn.RemoteAddr(),
			"local",
			conn.LocalAddr(),
		)

		err := conn.Close()
		if err != nil {
			log.ErrorContext(
				ctx,
				"Could not close connection",
				"port",
				p.port,
				"remote",
				conn.RemoteAddr(),
				"local",
				conn.LocalAddr(),
				"error",
				err,
			)
		}

		return
	}

	log.InfoContext(ctx, "Accepted connection", "port", p.port, "remote", conn.RemoteAddr(), "local", conn.LocalAddr())
	p.connections = append(p.connections, conn)

	select {
	case p.notify <- struct{}{}:
	default:
	}
}

func (p *Pool) get() []net.Conn {
	p.mu.Lock()
	defer p.mu.Unlock()

	return p.connections
}

// Close closes all connections in the pool and the listener.
func (p *Pool) close(ctx context.Context) {
	log := logger.FromContext(ctx)

	p.mu.Lock()
	defer p.mu.Unlock()

	// close all the connections in the pool
	for _, conn := range p.connections {
		_ = conn.Close()
	}
	p.connections = []net.Conn{}

	// close the listener
	err := p.listener.Close()
	if err != nil {
		log.ErrorContext(ctx, "Could not close listener", "error", err)
	}
}

// Reset closes all connections in the pool and resets it to empty.
func (p *Pool) Reset(ctx context.Context) {
	log := logger.FromContext(ctx)

	p.mu.Lock()
	defer p.mu.Unlock()

	// close all the connections in the pool
	for _, conn := range p.connections {
		err := conn.Close()
		if err != nil {
			log.ErrorContext(ctx, "Could not close connection", "error", err)
		}
	}
	p.connections = []net.Conn{}
}

// Wait blocks until there are size connections in the pool or the context is cancelled.
func (p *Pool) Wait(ctx context.Context, size int) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case <-p.notify:
			connections := p.get()
			if len(connections) < size {
				continue
			}

			return nil
		}
	}
}

// Bridge pipes data between the two connections in the pool.
func (p *Pool) Bridge(ctx context.Context) error {
	log := logger.FromContext(ctx)

	p.mu.Lock()
	defer p.mu.Unlock()

	if len(p.connections) < 2 { //nolint:mnd // min 2
		return errors.New("not enough connections to bridge")
	}

	wg := sync.WaitGroup{}

	wg.Go(func() {
		_, ioerr := io.Copy(p.connections[0], p.connections[1])
		if ioerr != nil {
			log.ErrorContext(ctx, "Could not pipe data", "error", ioerr, "port", p.port)
		}
	})

	wg.Go(func() {
		_, ioerr := io.Copy(p.connections[1], p.connections[0])
		if ioerr != nil {
			log.ErrorContext(ctx, "Could not pipe data", "error", ioerr, "port", p.port)
		}
	})

	wg.Wait()

	return nil
}
