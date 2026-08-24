package rpc

import (
	"fmt"
	"ntx/internal/shared/transport"
	"ntx/internal/shared/transport/unixsock"
	"time"
)

const (
	minBackoff = 1 * time.Second
	maxBackoff = 30 * time.Second
)

// Client is a JSON-RPC client over a persistent Unix domain socket
// connection, reconnecting with exponential backoff on failure.
type Client struct {
	socketPath string
	conn       transport.Conn
	nextID     uint64
	backoff    time.Duration
}

// NewClient creates a Client that connects lazily on the first Call.
func NewClient(socketPath string) *Client {
	return &Client{socketPath: socketPath, backoff: minBackoff}
}

func (c *Client) ensureConnected() error {
	if c.conn != nil {
		return nil
	}

	conn, err := unixsock.Dial(c.socketPath)
	if err != nil {
		time.Sleep(c.backoff)

		c.backoff *= 2
		if c.backoff > maxBackoff {
			c.backoff = maxBackoff
		}

		return fmt.Errorf("dial %s: %w", c.socketPath, err)
	}

	c.conn = conn
	c.backoff = minBackoff

	return nil
}

// Call sends a request for method with params and returns the decoded
// response. On any transport failure the connection is dropped so the next
// call reconnects. params is any rather than a type parameter because Go
// methods cannot declare their own type parameters.
func (c *Client) Call(method Method, params any) (Response, error) {
	if err := c.ensureConnected(); err != nil {
		return Response{}, err
	}

	c.nextID++

	req, err := NewRequest(c.nextID, method, params)
	if err != nil {
		return Response{}, fmt.Errorf("build request: %w", err)
	}

	if err := WriteRequest(c.conn, req); err != nil {
		c.closeBroken()

		return Response{}, fmt.Errorf("write request: %w", err)
	}

	resp, err := ReadResponse(c.conn)
	if err != nil {
		c.closeBroken()

		return Response{}, fmt.Errorf("read response: %w", err)
	}

	return resp, nil
}

func (c *Client) closeBroken() {
	_ = c.conn.Close()
	c.conn = nil
}
