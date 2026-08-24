// Package notification implements the createNotification domain logic:
// validating an incoming notification and persisting it to the
// notifications table.
package notification

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"ntx/internal/shared/dto"
	"ntx/internal/shared/rpc"
	"time"

	"github.com/google/uuid"
)

// maxIconBytes is the largest decoded icon size accepted.
const maxIconBytes = 8 * 1024 * 1024

var _ rpc.ValidationError = (*ValidationError)(nil)

// ValidationError describes a single invalid field in
// dto.CreateNotificationParams.
type ValidationError struct {
	Field   string
	Message string
}

// Error implements the error interface.
func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// Validation implements rpc.ValidationError.
func (e *ValidationError) Validation() {}

// ValidateCreate checks p for required fields and constraints, returning a
// *ValidationError describing the first problem found, if any.
func ValidateCreate(p dto.CreateNotificationParams) error {
	if p.AppName == "" {
		return &ValidationError{Field: "app_name", Message: "must not be empty"}
	}

	if p.Summary == "" {
		return &ValidationError{Field: "summary", Message: "must not be empty"}
	}

	if p.Timestamp == "" {
		return &ValidationError{Field: "timestamp", Message: "must not be empty"}
	}

	if _, err := time.Parse(time.RFC3339, p.Timestamp); err != nil {
		return &ValidationError{Field: "timestamp", Message: "must be RFC3339"}
	}

	if (p.IconBase64 == "") != (p.IconMime == "") {
		return &ValidationError{
			Field:   "icon_base64/icon_mime",
			Message: "must both be present or both be absent",
		}
	}

	if p.IconBase64 != "" {
		decoded, err := base64.StdEncoding.DecodeString(p.IconBase64)
		if err != nil {
			return &ValidationError{Field: "icon_base64", Message: "must be valid base64"}
		}

		if len(decoded) > maxIconBytes {
			return &ValidationError{Field: "icon_base64", Message: "exceeds max icon size"}
		}
	}

	return nil
}

func nullableString(s string) any {
	if s == "" {
		return nil
	}

	return s
}

// CreateNotificationHandler implements rpc.RouteHandler for the
// createNotification method, bound to one *sql.DB.
type CreateNotificationHandler struct {
	sqlDB *sql.DB
}

// NewCreateNotificationHandler builds a createNotification RPC handler
// bound to sqlDB.
func NewCreateNotificationHandler(sqlDB *sql.DB) *CreateNotificationHandler {
	return &CreateNotificationHandler{sqlDB}
}

// Handle validates p, inserts a new notification row, and returns the
// createNotification RPC result carrying its generated UUIDv7 id.
func (cnh *CreateNotificationHandler) Handle(
	_ context.Context,
	p dto.CreateNotificationParams,
) (dto.CreateNotificationResult, error) {
	if err := ValidateCreate(p); err != nil {
		return dto.CreateNotificationResult{}, err
	}

	id, err := uuid.NewV7()
	if err != nil {
		return dto.CreateNotificationResult{}, fmt.Errorf("notification: generate id: %w", err)
	}

	actions := p.Actions
	if actions == nil {
		actions = []string{}
	}

	hints := p.Hints
	if hints == nil {
		hints = map[string]any{}
	}

	actionsJSON, err := json.Marshal(actions)
	if err != nil {
		return dto.CreateNotificationResult{}, fmt.Errorf("notification: marshal actions: %w", err)
	}

	hintsJSON, err := json.Marshal(hints)
	if err != nil {
		return dto.CreateNotificationResult{}, fmt.Errorf("notification: marshal hints: %w", err)
	}

	const stmt = `
		INSERT INTO notifications
			(id, timestamp, app_name, replaces_id, summary, body, actions, hints, expire_timeout, icon_base64, icon_mime)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = cnh.sqlDB.Exec(stmt,
		id.String(), p.Timestamp, p.AppName, p.ReplacesID, p.Summary, p.Body,
		string(actionsJSON), string(hintsJSON), p.ExpireTimeout,
		nullableString(p.IconBase64), nullableString(p.IconMime),
	)
	if err != nil {
		return dto.CreateNotificationResult{}, fmt.Errorf("notification: insert: %w", err)
	}

	return dto.CreateNotificationResult{ID: id.String()}, nil
}
