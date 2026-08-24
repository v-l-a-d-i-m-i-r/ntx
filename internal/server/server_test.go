package server

import (
	"context"
	"net"
	"ntx/internal/shared/log"
	"ntx/internal/shared/rpc"
	"testing"
)

type testParams struct {
	Value string `json:"value"`
}

type testResult struct {
	Echo string `json:"echo"`
}

func newTestRouter(
	handler func(ctx context.Context, p testParams) (testResult, error),
) (*rpc.Router, rpc.Method) {
	const method rpc.Method = "testMethod"

	rt := rpc.NewRouter()
	rt.Register(method, handler)

	return rt, method
}

// startHandleConn runs handleConn against one end of an in-memory pipe and
// returns the other end for the test to drive as an RPC client.
func startHandleConn(t *testing.T, rt *rpc.Router) net.Conn {
	t.Helper()

	client, server := net.Pipe()
	t.Cleanup(func() { _ = client.Close() })

	logger := log.NewStdout(log.StdoutParams{})

	go handleConn(context.Background(), server, rt, logger)

	return client
}

func sendRequest(t *testing.T, conn net.Conn, method rpc.Method, params []byte) rpc.Response {
	t.Helper()

	req := rpc.Request{JSONRPC: "2.0", ID: 1, Method: method, Params: params}

	if err := rpc.WriteRequest(conn, req); err != nil {
		t.Fatalf("write request: %v", err)
	}

	resp, err := rpc.ReadResponse(conn)
	if err != nil {
		t.Fatalf("read response: %v", err)
	}

	return resp
}

func TestHandleConnSuccess(t *testing.T) {
	t.Parallel()

	rt, method := newTestRouter(func(_ context.Context, p testParams) (testResult, error) {
		return testResult{Echo: p.Value}, nil
	})
	conn := startHandleConn(t, rt)

	resp := sendRequest(t, conn, method, []byte(`{"value":"hi"}`))

	if resp.Error != nil {
		t.Fatalf("unexpected error response: %+v", resp.Error)
	}

	if string(resp.Result) != `{"echo":"hi"}` {
		t.Fatalf("unexpected result: %s", resp.Result)
	}
}

func TestHandleConnInvalidParams(t *testing.T) {
	t.Parallel()

	rt, method := newTestRouter(func(_ context.Context, _ testParams) (testResult, error) {
		t.Fatal("handler should not be called for invalid params")

		return testResult{}, nil
	})
	conn := startHandleConn(t, rt)

	resp := sendRequest(t, conn, method, []byte("123"))

	if resp.Error == nil || resp.Error.Code != rpc.ErrCodeInvalidParams {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestHandleConnMethodNotFound(t *testing.T) {
	t.Parallel()

	rt := rpc.NewRouter()
	conn := startHandleConn(t, rt)

	resp := sendRequest(t, conn, "unknown", nil)

	if resp.Error == nil || resp.Error.Code != rpc.ErrCodeMethodNotFound {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestHandleConnMultipleRequestsOnOneConn(t *testing.T) {
	t.Parallel()

	rt, method := newTestRouter(func(_ context.Context, p testParams) (testResult, error) {
		return testResult{Echo: p.Value}, nil
	})
	conn := startHandleConn(t, rt)

	first := sendRequest(t, conn, method, []byte(`{"value":"one"}`))
	if string(first.Result) != `{"echo":"one"}` {
		t.Fatalf("unexpected first result: %s", first.Result)
	}

	second := sendRequest(t, conn, method, []byte(`{"value":"two"}`))
	if string(second.Result) != `{"echo":"two"}` {
		t.Fatalf("unexpected second result: %s", second.Result)
	}
}
