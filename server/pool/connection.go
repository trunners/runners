package pool

import (
	"bufio"
	"io"
	"net"
)

type ConnectionProtocol int

const (
	TypeTCP ConnectionProtocol = iota
	TypeSSH
)

type Connection struct {
	net.Conn

	Protocol ConnectionProtocol
	r        *bufio.Reader
}

func newConnection(c net.Conn) Connection {
	return Connection{
		c,
		TypeTCP,
		bufio.NewReader(c),
	}
}

func (b Connection) Peek(n int) ([]byte, error) {
	return b.r.Peek(n)
}

func (b Connection) Read(p []byte) (int, error) {
	return b.r.Read(p)
}

func (b Connection) ReadBytes(n int) ([]byte, error) {
	buf := make([]byte, n)
	_, err := io.ReadFull(b, buf)
	if err != nil {
		return nil, err
	}

	return buf, nil
}

func (b Connection) Type() string {
	switch b.Protocol {
	case TypeSSH:
		return "SSH"
	case TypeTCP:
		fallthrough
	default:
		return "TCP"
	}
}
