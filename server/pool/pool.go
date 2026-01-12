package pool

import (
	"context"
	"errors"
	"fmt"
	"net"

	"github.com/trunners/runners/logger"
)

type Pool struct {
	port     int
	listener net.Listener

	tcps chan net.Conn
	sshs chan net.Conn
}

func Start(ctx context.Context, port int) (*Pool, error) {
	log := logger.FromContext(ctx)

	cfg := net.ListenConfig{}
	listener, err := cfg.Listen(ctx, "tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return nil, err
	}

	p := &Pool{
		port:     port,
		listener: listener,
		tcps:     make(chan net.Conn, 10), //nolint:mnd // buffer size 10
		sshs:     make(chan net.Conn, 10), //nolint:mnd // buffer size 10
	}

	// start listening for connections
	go p.listen(ctx)

	// close listener on context done
	go func() {
		<-ctx.Done()
		err = p.listener.Close()
		if err != nil {
			log.ErrorContext(ctx, "Could not close listener", "error", err)
		}
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

	connection := newConnection(conn)

	test, err := connection.Peek(3) //nolint:mnd // peek at first 3 bytes to determine connection type
	switch {
	case err != nil:
		log.WarnContext(ctx, "Could not peek connection", "error", err)
	case string(test) == "SSH":
		connection.Protocol = TypeSSH
	case string(test) == "TCP":
		connection.Protocol = TypeTCP
		_, err = connection.ReadBytes(3) //nolint:mnd // read the first 3 bytes to pop them
		if err != nil {
			log.WarnContext(ctx, "Could not read END bytes", "error", err)
		}
	}

	log.DebugContext(
		ctx,
		"New connection",
		"type",
		connection.Type(),
		"remote",
		connection.RemoteAddr(),
		"local",
		connection.LocalAddr(),
	)

	switch connection.Protocol {
	case TypeSSH:
		select {
		case p.sshs <- connection:
		default:
			log.WarnContext(ctx, "SSH connection pool full, closing connection", "remote", connection.RemoteAddr())
			_ = connection.Close()
		}

	case TypeTCP:
		fallthrough
	default:
		select {
		case p.tcps <- connection:
		default:
			log.WarnContext(ctx, "TCP connection pool full, closing connection", "remote", connection.RemoteAddr())
			_ = connection.Close()
		}
	}
}

// Next returns the next connection from the pool matching the given protocol.
func (p *Pool) Next(ctx context.Context, protocol ConnectionProtocol) (net.Conn, error) {
	for {
		switch protocol {
		case TypeSSH:
			select {
			case <-ctx.Done():
				return nil, ctx.Err()

			case connection := <-p.sshs:
				return connection, nil
			}

		case TypeTCP:
			fallthrough
		default:
			select {
			case <-ctx.Done():
				return nil, ctx.Err()

			case connection := <-p.tcps:
				return connection, nil
			}
		}
	}
}
