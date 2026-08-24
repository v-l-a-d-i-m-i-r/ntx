// Package frame implements a length-prefixed message framing protocol over
// an io.Reader/io.Writer stream: a 4-byte big-endian length prefix followed
// by that many bytes of body.
package frame

import (
	"encoding/binary"
	"fmt"
	"io"
)

// MaxSize is the largest body a frame may carry. Reads that declare a
// larger length are rejected before any body bytes are read.
const MaxSize = 16 * 1024 * 1024

const prefixSize = 4

// Write encodes body as a single frame and writes it to w.
func Write(w io.Writer, body []byte) error {
	if len(body) > MaxSize {
		return fmt.Errorf("frame: body of %d bytes exceeds max size %d", len(body), MaxSize)
	}

	var prefix [prefixSize]byte
	binary.BigEndian.PutUint32(prefix[:], uint32(len(body))) //nolint:gosec // bounds checked above

	if _, err := w.Write(prefix[:]); err != nil {
		return fmt.Errorf("frame: write length prefix: %w", err)
	}

	if _, err := w.Write(body); err != nil {
		return fmt.Errorf("frame: write body: %w", err)
	}

	return nil
}

// Read reads a single frame from r and returns its body.
func Read(r io.Reader) ([]byte, error) {
	var prefix [prefixSize]byte

	if _, err := io.ReadFull(r, prefix[:]); err != nil {
		return nil, fmt.Errorf("frame: read length prefix: %w", err)
	}

	size := binary.BigEndian.Uint32(prefix[:])
	if size > MaxSize {
		return nil, fmt.Errorf("frame: declared body size %d exceeds max size %d", size, MaxSize)
	}

	body := make([]byte, size)
	if _, err := io.ReadFull(r, body); err != nil {
		return nil, fmt.Errorf("frame: read body: %w", err)
	}

	return body, nil
}
