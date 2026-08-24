package rpc_test

import (
	"ntx/internal/shared/rpc"
	"ntx/internal/shared/transport"
	"ntx/internal/shared/transport/unixsock"
	"path/filepath"
	"testing"
)

// serve accepts a single connection on ln, reads one request, and replies
// with resp.
func serve(t *testing.T, ln transport.Listener, resp rpc.Response) {
	t.Helper()

	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close() //nolint:errcheck // test cleanup

		if _, err := rpc.ReadRequest(conn); err != nil {
			t.Errorf("server ReadRequest: %v", err)

			return
		}

		if err := rpc.WriteResponse(conn, resp); err != nil {
			t.Errorf("server WriteResponse: %v", err)
		}
	}()
}

func TestClientCallSuccess(t *testing.T) {
	t.Parallel()

	sockPath := filepath.Join(t.TempDir(), "test.sock")

	ln, err := unixsock.Listen(sockPath)
	if err != nil {
		t.Fatalf("Listen: unexpected error: %v", err)
	}
	defer ln.Close() //nolint:errcheck // test cleanup

	want, err := rpc.NewResultResponse(1, map[string]string{"id": "abc"})
	if err != nil {
		t.Fatalf("NewResultResponse: unexpected error: %v", err)
	}

	serve(t, ln, want)

	client := rpc.NewClient(sockPath)

	resp, err := client.Call("createNotification", map[string]string{"summary": "hi"})
	if err != nil {
		t.Fatalf("Call: unexpected error: %v", err)
	}

	if resp.Error != nil {
		t.Fatalf("Call: Error = %+v, want nil", resp.Error)
	}
}

func TestClientCallServerError(t *testing.T) {
	t.Parallel()

	sockPath := filepath.Join(t.TempDir(), "test.sock")

	ln, err := unixsock.Listen(sockPath)
	if err != nil {
		t.Fatalf("Listen: unexpected error: %v", err)
	}
	defer ln.Close() //nolint:errcheck // test cleanup

	serve(t, ln, rpc.NewErrorResponse(1, rpc.ErrCodeValidation, "summary is required"))

	client := rpc.NewClient(sockPath)

	resp, err := client.Call("createNotification", map[string]string{})
	if err != nil {
		t.Fatalf("Call: unexpected error: %v", err)
	}

	if resp.Error == nil || resp.Error.Message != "summary is required" {
		t.Fatalf("Call: Error = %+v, want message %q", resp.Error, "summary is required")
	}
}

func TestClientCallDialFailureReturnsError(t *testing.T) {
	t.Parallel()

	client := rpc.NewClient(filepath.Join(t.TempDir(), "nonexistent.sock"))

	if _, err := client.Call("createNotification", map[string]string{}); err == nil {
		t.Fatal("Call: expected error dialing nonexistent socket, got nil")
	}
}

func TestClientCallReconnectsAfterServerRestart(t *testing.T) {
	t.Parallel()

	sockPath := filepath.Join(t.TempDir(), "test.sock")

	ln, err := unixsock.Listen(sockPath)
	if err != nil {
		t.Fatalf("Listen: unexpected error: %v", err)
	}

	want, err := rpc.NewResultResponse(1, map[string]string{"id": "abc"})
	if err != nil {
		t.Fatalf("NewResultResponse: unexpected error: %v", err)
	}

	serve(t, ln, want)

	client := rpc.NewClient(sockPath)

	if _, err := client.Call("createNotification", map[string]string{"summary": "hi"}); err != nil {
		t.Fatalf("first Call: unexpected error: %v", err)
	}

	ln.Close() //nolint:errcheck // test cleanup

	if _, err := client.Call("createNotification", map[string]string{"summary": "hi"}); err == nil {
		t.Fatal("second Call: expected error after server closed, got nil")
	}

	ln, err = unixsock.Listen(sockPath)
	if err != nil {
		t.Fatalf("re-Listen: unexpected error: %v", err)
	}
	defer ln.Close() //nolint:errcheck // test cleanup

	serve(t, ln, want)

	if _, err := client.Call("createNotification", map[string]string{"summary": "hi"}); err != nil {
		t.Fatalf("third Call: unexpected error reconnecting: %v", err)
	}
}
