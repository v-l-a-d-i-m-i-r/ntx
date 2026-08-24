package notification

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"ntx/internal/shared/dto"
	"ntx/internal/shared/rpc"
)

var _ rpc.NotFoundError = (*NotFoundError)(nil)

// NotFoundError reports that no notification matches a requested id.
type NotFoundError struct {
	ID string
}

// Error implements the error interface.
func (e *NotFoundError) Error() string {
	return fmt.Sprintf("notification not found: %s", e.ID)
}

// NotFound implements rpc.NotFoundError.
func (e *NotFoundError) NotFound() {}

// ValidateGet checks p for required fields, returning a *ValidationError
// describing the problem, if any.
func ValidateGet(p dto.GetNotificationParams) error {
	if p.ID == "" {
		return &ValidationError{Field: "id", Message: "must not be empty"}
	}

	return nil
}

// GetNotificationHandler implements rpc.RouteHandler for the
// getNotification method, bound to one *sql.DB.
type GetNotificationHandler struct {
	sqlDB *sql.DB
}

// NewGetNotificationHandler builds a getNotification RPC handler bound to
// sqlDB.
func NewGetNotificationHandler(sqlDB *sql.DB) *GetNotificationHandler {
	return &GetNotificationHandler{sqlDB}
}

// Handle validates p and returns the notification row matching p.ID as the
// getNotification RPC result, or a *NotFoundError if no row matches.
func (gnh *GetNotificationHandler) Handle(
	_ context.Context,
	p dto.GetNotificationParams,
) (dto.GetNotificationResult, error) {
	if err := ValidateGet(p); err != nil {
		return dto.GetNotificationResult{}, err
	}

	const stmt = `
		SELECT id, timestamp, app_name, replaces_id, summary, body, actions, hints,
			expire_timeout, icon_base64, icon_mime
		FROM notifications
		WHERE id = ?
	`

	var res dto.GetNotificationResult
	var actionsJSON string
	var hintsJSON string
	var iconBase64 sql.NullString
	var iconMime sql.NullString

	row := gnh.sqlDB.QueryRow(stmt, p.ID)

	err := row.Scan(
		&res.ID, &res.Timestamp, &res.AppName, &res.ReplacesID, &res.Summary, &res.Body,
		&actionsJSON, &hintsJSON, &res.ExpireTimeout, &iconBase64, &iconMime,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return dto.GetNotificationResult{}, &NotFoundError{ID: p.ID}
	}

	if err != nil {
		return dto.GetNotificationResult{}, fmt.Errorf("notification: query: %w", err)
	}

	if err := json.Unmarshal([]byte(actionsJSON), &res.Actions); err != nil {
		return dto.GetNotificationResult{}, fmt.Errorf("notification: unmarshal actions: %w", err)
	}

	if err := json.Unmarshal([]byte(hintsJSON), &res.Hints); err != nil {
		return dto.GetNotificationResult{}, fmt.Errorf("notification: unmarshal hints: %w", err)
	}

	res.IconBase64 = iconBase64.String
	res.IconMime = iconMime.String

	return res, nil
}
