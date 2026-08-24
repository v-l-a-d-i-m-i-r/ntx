package rpc_test

import (
	"context"
	"encoding/json"
	"errors"
	"ntx/internal/shared/rpc"
	"testing"
)

type routerTestParams struct {
	Value string `json:"value"`
}

type routerTestResult struct {
	Echo string `json:"echo"`
}

type routerTestValidationErr struct{}

func (routerTestValidationErr) Error() string { return "invalid value" }
func (routerTestValidationErr) Validation()   {}

type routerTestNotFoundErr struct{}

func (routerTestNotFoundErr) Error() string { return "missing" }
func (routerTestNotFoundErr) NotFound()     {}

var errRouterTestBoom = errors.New("boom")

func newRouterRequest(t *testing.T, method rpc.Method, params []byte) []byte {
	t.Helper()

	raw, err := json.Marshal(rpc.Request{JSONRPC: "2.0", ID: 1, Method: method, Params: params})
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}

	return raw
}

func decodeRouterResponse(t *testing.T, raw []byte) rpc.Response {
	t.Helper()

	var resp rpc.Response
	if err := json.Unmarshal(raw, &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	return resp
}

func TestRouterHandleSuccess(t *testing.T) {
	t.Parallel()

	rt := rpc.NewRouter()
	rt.Register("echo", func(_ context.Context, p routerTestParams) (routerTestResult, error) {
		return routerTestResult{Echo: p.Value}, nil
	})

	raw, err := rt.Handle(
		context.Background(),
		newRouterRequest(t, "echo", []byte(`{"value":"hi"}`)),
	)
	if err != nil {
		t.Fatalf("Handle: unexpected error: %v", err)
	}

	resp := decodeRouterResponse(t, raw)
	if resp.Error != nil {
		t.Fatalf("unexpected error response: %+v", resp.Error)
	}

	if string(resp.Result) != `{"echo":"hi"}` {
		t.Fatalf("unexpected result: %s", resp.Result)
	}
}

func TestRouterHandleMethodNotFound(t *testing.T) {
	t.Parallel()

	rt := rpc.NewRouter()

	raw, err := rt.Handle(context.Background(), newRouterRequest(t, "unknown", nil))
	if err != nil {
		t.Fatalf("Handle: unexpected error: %v", err)
	}

	resp := decodeRouterResponse(t, raw)
	if resp.Error == nil || resp.Error.Code != rpc.ErrCodeMethodNotFound {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestRouterHandleInvalidParams(t *testing.T) {
	t.Parallel()

	rt := rpc.NewRouter()
	rt.Register("echo", func(_ context.Context, _ routerTestParams) (routerTestResult, error) {
		t.Fatal("handler should not be called for invalid params")

		return routerTestResult{}, nil
	})

	raw, err := rt.Handle(context.Background(), newRouterRequest(t, "echo", []byte("123")))
	if err != nil {
		t.Fatalf("Handle: unexpected error: %v", err)
	}

	resp := decodeRouterResponse(t, raw)
	if resp.Error == nil || resp.Error.Code != rpc.ErrCodeInvalidParams {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestRouterHandleValidationError(t *testing.T) {
	t.Parallel()

	rt := rpc.NewRouter()
	rt.Register("echo", func(_ context.Context, _ routerTestParams) (routerTestResult, error) {
		return routerTestResult{}, routerTestValidationErr{}
	})

	raw, err := rt.Handle(context.Background(), newRouterRequest(t, "echo", []byte("{}")))
	if err != nil {
		t.Fatalf("Handle: unexpected error: %v", err)
	}

	resp := decodeRouterResponse(t, raw)
	if resp.Error == nil || resp.Error.Code != rpc.ErrCodeValidation {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestRouterHandleNotFoundError(t *testing.T) {
	t.Parallel()

	rt := rpc.NewRouter()
	rt.Register("echo", func(_ context.Context, _ routerTestParams) (routerTestResult, error) {
		return routerTestResult{}, routerTestNotFoundErr{}
	})

	raw, err := rt.Handle(context.Background(), newRouterRequest(t, "echo", []byte("{}")))
	if err != nil {
		t.Fatalf("Handle: unexpected error: %v", err)
	}

	resp := decodeRouterResponse(t, raw)
	if resp.Error == nil || resp.Error.Code != rpc.ErrCodeNotFound {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestRouterHandleInternalErrorHidesDetail(t *testing.T) {
	t.Parallel()

	rt := rpc.NewRouter()
	rt.Register("echo", func(_ context.Context, _ routerTestParams) (routerTestResult, error) {
		return routerTestResult{}, errRouterTestBoom
	})

	raw, err := rt.Handle(context.Background(), newRouterRequest(t, "echo", []byte("{}")))
	if err != nil {
		t.Fatalf("Handle: unexpected error: %v", err)
	}

	resp := decodeRouterResponse(t, raw)
	if resp.Error == nil || resp.Error.Code != rpc.ErrCodeInternal {
		t.Fatalf("unexpected response: %+v", resp)
	}

	if resp.Error.Message == errRouterTestBoom.Error() {
		t.Fatalf("internal error message leaked handler detail: %q", resp.Error.Message)
	}
}
