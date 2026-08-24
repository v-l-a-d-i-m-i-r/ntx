package rpc

import (
	"context"
	"encoding/json"
	"errors"
	"log"
)

// RouteHandler is a typed handler for one RPC method.
type RouteHandler[P any, R any] func(ctx context.Context, params P) (R, error)

type rawHandler func(ctx context.Context, rawParams json.RawMessage) (json.RawMessage, error)

// Router dispatches raw JSON-RPC requests to registered RouteHandlers by
// method name.
type Router struct {
	handlers map[Method]rawHandler
}

// NewRouter builds an empty Router.
func NewRouter() *Router {
	return &Router{handlers: make(map[Method]rawHandler)}
}

// Register binds fn to method, decoding params into P and encoding the
// result as R for every request to method.
func (r *Router) Register[P any, R any](method Method, fn RouteHandler[P, R]) {
	r.handlers[method] = func(ctx context.Context, rawParams json.RawMessage) (json.RawMessage, error) {
		var params P
		if len(rawParams) > 0 {
			if err := json.Unmarshal(rawParams, &params); err != nil {
				return nil, &Error{Code: ErrCodeInvalidParams, Message: "invalid params"}
			}
		}

		result, err := fn(ctx, params)
		if err != nil {
			return nil, err
		}

		data, err := json.Marshal(result)
		if err != nil {
			return nil, &Error{Code: ErrCodeInternal, Message: "internal error"}
		}

		return data, nil
	}
}

// Handle decodes rawReq as a Request, dispatches it to the registered
// handler for its method, and returns the marshaled Response.
func (r *Router) Handle(ctx context.Context, rawReq []byte) ([]byte, error) {
	var req Request
	if err := json.Unmarshal(rawReq, &req); err != nil {
		return json.Marshal(NewErrorResponse(req.ID, ErrCodeInvalidRequest, "invalid request"))
	}

	handler, ok := r.handlers[req.Method]
	if !ok {
		return json.Marshal(NewErrorResponse(req.ID, ErrCodeMethodNotFound, "method not found"))
	}

	result, err := handler(ctx, req.Params)
	if err != nil {
		return json.Marshal(NewErrorResponse(req.ID, errCode(err), errMessage(err)))
	}

	return json.Marshal(Response{JSONRPC: version, ID: req.ID, Result: result})
}

// errCode maps a handler error to a JSON-RPC error code: *Error carries its
// own code, ValidationError/NotFoundError map to their dedicated codes, and
// everything else is an internal error.
func errCode(err error) int {
	if rpcErr, ok := errors.AsType[*Error](err); ok {
		return rpcErr.Code
	}

	if _, ok := errors.AsType[ValidationError](err); ok {
		return ErrCodeValidation
	}

	if _, ok := errors.AsType[NotFoundError](err); ok {
		return ErrCodeNotFound
	}

	return ErrCodeInternal
}

// errMessage returns the client-facing message for err. Internal errors are
// logged with full detail but reported to the client as a generic message,
// so unexpected failures don't leak implementation details.
func errMessage(err error) string {
	if rpcErr, ok := errors.AsType[*Error](err); ok {
		return rpcErr.Message
	}

	if _, ok := errors.AsType[ValidationError](err); ok {
		return err.Error()
	}

	if _, ok := errors.AsType[NotFoundError](err); ok {
		return err.Error()
	}

	log.Println("rpc: internal error:", err)

	return "internal error"
}
