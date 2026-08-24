// Package unixsock implements the transport.Listener/Conn abstraction over
// Unix domain sockets.
package unixsock

import (
	"fmt"
	"net"
	"ntx/internal/shared/transport"
	"os"
)

const socketPerm = 0o600

// Listen removes any stale socket file at path, binds a new Unix domain
// socket listener there, and sets its permissions to 0600.
func Listen(path string) (transport.Listener, error) {
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("unixsock: remove stale socket: %w", err)
	}

	ln, err := net.Listen("unix", path)
	if err != nil {
		return nil, fmt.Errorf("unixsock: listen on %s: %w", path, err)
	}

	if err := os.Chmod(path, socketPerm); err != nil {
		_ = ln.Close()

		return nil, fmt.Errorf("unixsock: chmod socket: %w", err)
	}

	return transport.NewListener(ln), nil
}

// Dial connects to a Unix domain socket at path.
func Dial(path string) (transport.Conn, error) {
	conn, err := net.Dial("unix", path)
	if err != nil {
		return nil, fmt.Errorf("unixsock: dial %s: %w", path, err)
	}

	return conn, nil
}
