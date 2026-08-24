// Package transport defines a pluggable Listen/Dial abstraction so the RPC
// layer is not tied to a specific network mechanism. Concrete transports
// live in subpackages (e.g. unixsock).
package transport

import (
	"io"
	"net"
)

// Conn is a bidirectional stream connection between two peers.
type Conn interface {
	io.ReadWriteCloser
}

// Listener accepts incoming Conns.
type Listener interface {
	Accept() (Conn, error)
	Close() error
}

var _ Listener = (*netListener)(nil)

// netListener adapts a net.Listener to Listener.
type netListener struct {
	ln net.Listener
}

// NewListener adapts a net.Listener to Listener.
func NewListener(ln net.Listener) Listener {
	return &netListener{ln: ln}
}

// Accept waits for and returns the next connection.
func (l *netListener) Accept() (Conn, error) {
	c, err := l.ln.Accept()
	if err != nil {
		return nil, err //nolint:wrapcheck // caller wraps with transport context
	}

	return c, nil
}

// Close closes the listener.
func (l *netListener) Close() error {
	return l.ln.Close() //nolint:wrapcheck // caller wraps with transport context
}
