package rpc_test

import (
	"bytes"
	"encoding/json"
	"ntx/internal/shared/rpc"
	"testing"
)

func TestRequestRoundTrip(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	req, err := rpc.NewRequest(1, "createNotification", map[string]string{"summary": "hi"})
	if err != nil {
		t.Fatalf("NewRequest: unexpected error: %v", err)
	}

	if err := rpc.WriteRequest(&buf, req); err != nil {
		t.Fatalf("WriteRequest: unexpected error: %v", err)
	}

	got, err := rpc.ReadRequest(&buf)
	if err != nil {
		t.Fatalf("ReadRequest: unexpected error: %v", err)
	}

	if got.Method != "createNotification" || got.ID != 1 {
		t.Errorf("ReadRequest = %+v, want method createNotification id 1", got)
	}

	var params map[string]string
	if err := json.Unmarshal(got.Params, &params); err != nil {
		t.Fatalf("unmarshal params: %v", err)
	}

	if params["summary"] != "hi" {
		t.Errorf("params[summary] = %q, want %q", params["summary"], "hi")
	}
}

func TestResultResponseRoundTrip(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	resp, err := rpc.NewResultResponse(7, map[string]string{"id": "abc"})
	if err != nil {
		t.Fatalf("NewResultResponse: unexpected error: %v", err)
	}

	if err := rpc.WriteResponse(&buf, resp); err != nil {
		t.Fatalf("WriteResponse: unexpected error: %v", err)
	}

	got, err := rpc.ReadResponse(&buf)
	if err != nil {
		t.Fatalf("ReadResponse: unexpected error: %v", err)
	}

	if got.ID != 7 || got.Error != nil {
		t.Fatalf("ReadResponse = %+v, want id 7, no error", got)
	}

	var result map[string]string
	if err := json.Unmarshal(got.Result, &result); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}

	if result["id"] != "abc" {
		t.Errorf("result[id] = %q, want %q", result["id"], "abc")
	}
}

func TestErrorResponseRoundTrip(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	resp := rpc.NewErrorResponse(3, rpc.ErrCodeValidation, "summary is required")

	if err := rpc.WriteResponse(&buf, resp); err != nil {
		t.Fatalf("WriteResponse: unexpected error: %v", err)
	}

	got, err := rpc.ReadResponse(&buf)
	if err != nil {
		t.Fatalf("ReadResponse: unexpected error: %v", err)
	}

	if got.Error == nil {
		t.Fatal("ReadResponse: Error = nil, want set")
	}

	if got.Error.Code != rpc.ErrCodeValidation || got.Error.Message != "summary is required" {
		t.Errorf(
			"Error = %+v, want code %d message %q",
			got.Error,
			rpc.ErrCodeValidation,
			"summary is required",
		)
	}
}

func TestReadRequestMalformedJSON(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	// A validly framed body that is not valid JSON.
	if err := writeRawFrame(&buf, []byte("not json")); err != nil {
		t.Fatalf("writeRawFrame: %v", err)
	}

	_, err := rpc.ReadRequest(&buf)
	if err == nil {
		t.Fatal("ReadRequest: expected error for malformed JSON, got nil")
	}
}

func writeRawFrame(buf *bytes.Buffer, body []byte) error {
	prefix := make([]byte, 4)
	prefix[0] = byte(len(body) >> 24)
	prefix[1] = byte(len(body) >> 16)
	prefix[2] = byte(len(body) >> 8)
	prefix[3] = byte(len(body))
	buf.Write(prefix)
	buf.Write(body)

	return nil
}
