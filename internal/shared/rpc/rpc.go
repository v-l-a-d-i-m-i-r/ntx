// Package rpc implements JSON-RPC 2.0 request/response encoding, framed
// with the internal/frame length-prefix protocol. It is transport-agnostic:
// callers supply an io.Reader/io.Writer.
package rpc

import (
	"encoding/json"
	"fmt"
	"io"
	"ntx/internal/shared/frame"
)

const version = "2.0"

// Method identifies the JSON-RPC method being called.
type Method string

// Standard JSON-RPC 2.0 error codes, plus a custom application-level
// validation error code.
const (
	ErrCodeInvalidRequest = -32600
	ErrCodeMethodNotFound = -32601
	ErrCodeInvalidParams  = -32602
	ErrCodeInternal       = -32603
	ErrCodeValidation     = -32000
	ErrCodeNotFound       = -32001
)

// Request is a JSON-RPC 2.0 request object.
type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      uint64          `json:"id"`
	Method  Method          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// Error is a JSON-RPC 2.0 error object.
type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// Error implements the error interface.
func (e *Error) Error() string {
	return fmt.Sprintf("rpc error %d: %s", e.Code, e.Message)
}

// ValidationError is implemented by domain errors that represent invalid
// client input. A router reports them to the client with
// ErrCodeValidation instead of ErrCodeInternal.
type ValidationError interface {
	error
	Validation()
}

// NotFoundError is implemented by domain errors that represent a missing
// resource. A router reports them to the client with ErrCodeNotFound
// instead of ErrCodeInternal.
type NotFoundError interface {
	error
	NotFound()
}

// Response is a JSON-RPC 2.0 response object. Exactly one of Result or
// Error is set.
type Response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      uint64          `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *Error          `json:"error,omitempty"`
}

// NewRequest builds a Request for method with params marshaled to JSON.
func NewRequest[T any](id uint64, method Method, params T) (Request, error) {
	raw, err := json.Marshal(params)
	if err != nil {
		return Request{}, fmt.Errorf("rpc: marshal params: %w", err)
	}

	return Request{JSONRPC: version, ID: id, Method: method, Params: raw}, nil
}

// NewResultResponse builds a success Response carrying result.
func NewResultResponse[T any](id uint64, result T) (Response, error) {
	raw, err := json.Marshal(result)
	if err != nil {
		return Response{}, fmt.Errorf("rpc: marshal result: %w", err)
	}

	return Response{JSONRPC: version, ID: id, Result: raw}, nil
}

// NewErrorResponse builds an error Response.
func NewErrorResponse(id uint64, code int, message string) Response {
	return Response{JSONRPC: version, ID: id, Error: &Error{Code: code, Message: message}}
}

// WriteRequest encodes req as JSON and writes it as a single frame to w.
func WriteRequest(w io.Writer, req Request) error {
	return writeJSON(w, req)
}

// ReadRequest reads a single frame from r and decodes it as a Request.
func ReadRequest(r io.Reader) (Request, error) {
	body, err := frame.Read(r)
	if err != nil {
		return Request{}, fmt.Errorf("rpc: read request frame: %w", err)
	}

	var req Request
	if err := json.Unmarshal(body, &req); err != nil {
		return Request{}, fmt.Errorf("rpc: decode request: %w", err)
	}

	return req, nil
}

// WriteResponse encodes resp as JSON and writes it as a single frame to w.
func WriteResponse(w io.Writer, resp Response) error {
	return writeJSON(w, resp)
}

// ReadResponse reads a single frame from r and decodes it as a Response.
func ReadResponse(r io.Reader) (Response, error) {
	body, err := frame.Read(r)
	if err != nil {
		return Response{}, fmt.Errorf("rpc: read response frame: %w", err)
	}

	var resp Response
	if err := json.Unmarshal(body, &resp); err != nil {
		return Response{}, fmt.Errorf("rpc: decode response: %w", err)
	}

	return resp, nil
}

func writeJSON[T any](w io.Writer, v T) error {
	body, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("rpc: marshal: %w", err)
	}

	if err := frame.Write(w, body); err != nil {
		return fmt.Errorf("rpc: write frame: %w", err)
	}

	return nil
}
