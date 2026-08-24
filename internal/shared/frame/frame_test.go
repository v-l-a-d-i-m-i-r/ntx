package frame_test

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"ntx/internal/shared/frame"
	"strings"
	"testing"
)

func TestWriteReadRoundTrip(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	want := []byte("hello, world")
	if err := frame.Write(&buf, want); err != nil {
		t.Fatalf("Write: unexpected error: %v", err)
	}

	got, err := frame.Read(&buf)
	if err != nil {
		t.Fatalf("Read: unexpected error: %v", err)
	}

	if !bytes.Equal(got, want) {
		t.Errorf("Read = %q, want %q", got, want)
	}
}

func TestWriteReadEmptyBody(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	if err := frame.Write(&buf, []byte{}); err != nil {
		t.Fatalf("Write: unexpected error: %v", err)
	}

	got, err := frame.Read(&buf)
	if err != nil {
		t.Fatalf("Read: unexpected error: %v", err)
	}

	if len(got) != 0 {
		t.Errorf("Read = %q, want empty", got)
	}
}

func TestWriteRejectsOversizedBody(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	err := frame.Write(&buf, make([]byte, frame.MaxSize+1))
	if err == nil {
		t.Fatal("Write: expected error for oversized body, got nil")
	}
}

func TestReadRejectsOversizedPrefix(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	var prefix [4]byte
	binary.BigEndian.PutUint32(prefix[:], frame.MaxSize+1)
	buf.Write(prefix[:])

	_, err := frame.Read(&buf)
	if err == nil {
		t.Fatal("Read: expected error for oversized declared length, got nil")
	}
}

func TestReadTruncatedPrefix(t *testing.T) {
	t.Parallel()

	_, err := frame.Read(strings.NewReader("ab"))
	if err == nil {
		t.Fatal("Read: expected error for truncated prefix, got nil")
	}
}

func TestReadTruncatedBody(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	var prefix [4]byte
	binary.BigEndian.PutUint32(prefix[:], 10)
	buf.Write(prefix[:])
	buf.WriteString("short")

	_, err := frame.Read(&buf)
	if err == nil {
		t.Fatal("Read: expected error for truncated body, got nil")
	}

	if err != nil && !bytes.Contains([]byte(err.Error()), []byte("read body")) {
		t.Errorf("Read error = %v, want body read error", err)
	}
}

func TestReadNoData(t *testing.T) {
	t.Parallel()

	_, err := frame.Read(bytes.NewReader(nil))
	if err == nil {
		t.Fatal("Read: expected error for no data, got nil")
	}

	if !errors.Is(err, io.EOF) {
		t.Errorf("Read error = %v, want EOF-related error", err)
	}
}
