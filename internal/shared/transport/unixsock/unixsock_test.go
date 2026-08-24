package unixsock_test

import (
	"io"
	"ntx/internal/shared/transport/unixsock"
	"os"
	"path/filepath"
	"testing"
)

func TestListenDialRoundTrip(t *testing.T) {
	t.Parallel()

	sockPath := filepath.Join(t.TempDir(), "test.sock")

	ln, err := unixsock.Listen(sockPath)
	if err != nil {
		t.Fatalf("Listen: unexpected error: %v", err)
	}
	defer ln.Close() //nolint:errcheck // test cleanup

	done := make(chan struct{})

	go func() {
		defer close(done)

		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close() //nolint:errcheck // test cleanup

		buf := make([]byte, 5)
		if _, err := io.ReadFull(conn, buf); err != nil {
			t.Errorf("server read: %v", err)

			return
		}

		if string(buf) != "hello" {
			t.Errorf("server read = %q, want hello", buf)
		}

		if _, err := conn.Write([]byte("world")); err != nil {
			t.Errorf("server write: %v", err)
		}
	}()

	client, err := unixsock.Dial(sockPath)
	if err != nil {
		t.Fatalf("Dial: unexpected error: %v", err)
	}
	defer client.Close() //nolint:errcheck // test cleanup

	if _, err := client.Write([]byte("hello")); err != nil {
		t.Fatalf("client write: %v", err)
	}

	buf := make([]byte, 5)
	if _, err := io.ReadFull(client, buf); err != nil {
		t.Fatalf("client read: %v", err)
	}

	if string(buf) != "world" {
		t.Errorf("client read = %q, want world", buf)
	}

	<-done
}

func TestListenRemovesStaleSocket(t *testing.T) {
	t.Parallel()

	sockPath := filepath.Join(t.TempDir(), "stale.sock")

	if err := os.WriteFile(sockPath, []byte("stale"), 0o600); err != nil {
		t.Fatalf("write stale file: %v", err)
	}

	ln, err := unixsock.Listen(sockPath)
	if err != nil {
		t.Fatalf("Listen: unexpected error binding over stale socket file: %v", err)
	}
	defer ln.Close() //nolint:errcheck // test cleanup
}

func TestListenSetsSocketPermissions(t *testing.T) {
	t.Parallel()

	sockPath := filepath.Join(t.TempDir(), "perm.sock")

	ln, err := unixsock.Listen(sockPath)
	if err != nil {
		t.Fatalf("Listen: unexpected error: %v", err)
	}
	defer ln.Close() //nolint:errcheck // test cleanup

	info, err := os.Stat(sockPath)
	if err != nil {
		t.Fatalf("stat socket: %v", err)
	}

	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("socket perm = %o, want 600", perm)
	}
}

func TestDialNoServerFails(t *testing.T) {
	t.Parallel()

	_, err := unixsock.Dial(filepath.Join(t.TempDir(), "nonexistent.sock"))
	if err == nil {
		t.Fatal("Dial: expected error connecting to nonexistent socket, got nil")
	}
}
