package notification_test

import (
	"context"
	"database/sql"
	"encoding/base64"
	"errors"
	"ntx/internal/notification"
	"ntx/internal/shared/dto"
	"testing"
)

func handleGet(sqlDB *sql.DB, p dto.GetNotificationParams) (dto.GetNotificationResult, error) {
	return notification.NewGetNotificationHandler(sqlDB).Handle(context.Background(), p)
}

func TestGetReturnsInsertedNotification(t *testing.T) {
	t.Parallel()

	sqlDB := openTestDB(t)

	created, err := handleCreate(sqlDB, validParams())
	if err != nil {
		t.Fatalf("HandleCreate: unexpected error: %v", err)
	}

	res, err := handleGet(sqlDB, dto.GetNotificationParams(created))
	if err != nil {
		t.Fatalf("HandleGet: unexpected error: %v", err)
	}

	if res.ID != created.ID || res.AppName != "Firefox" || res.Summary != "New message" {
		t.Errorf("unexpected result: %+v", res)
	}

	if res.IconBase64 != "" || res.IconMime != "" {
		t.Errorf("icon fields = (%q, %q), want empty", res.IconBase64, res.IconMime)
	}
}

func TestGetReturnsIcon(t *testing.T) {
	t.Parallel()

	sqlDB := openTestDB(t)

	p := validParams()
	p.IconBase64 = base64.StdEncoding.EncodeToString([]byte("fake-png-bytes"))
	p.IconMime = "image/png"

	created, err := handleCreate(sqlDB, p)
	if err != nil {
		t.Fatalf("HandleCreate: unexpected error: %v", err)
	}

	res, err := handleGet(sqlDB, dto.GetNotificationParams(created))
	if err != nil {
		t.Fatalf("HandleGet: unexpected error: %v", err)
	}

	if res.IconBase64 != p.IconBase64 || res.IconMime != "image/png" {
		t.Errorf(
			"icon = (%q, %q), want (%q, image/png)",
			res.IconBase64,
			res.IconMime,
			p.IconBase64,
		)
	}
}

func TestGetReturnsNotFoundError(t *testing.T) {
	t.Parallel()

	sqlDB := openTestDB(t)

	_, err := handleGet(sqlDB, dto.GetNotificationParams{ID: "missing-id"})

	var nferr *notification.NotFoundError
	if !errors.As(err, &nferr) {
		t.Fatalf("HandleGet: error = %v, want *NotFoundError", err)
	}
}

func TestValidateGetRejectsEmptyID(t *testing.T) {
	t.Parallel()

	err := notification.ValidateGet(dto.GetNotificationParams{})
	if err == nil {
		t.Fatal("ValidateGet: expected error, got nil")
	}
}
