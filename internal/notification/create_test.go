package notification_test

import (
	"context"
	"database/sql"
	"encoding/base64"
	"ntx/internal/db"
	"ntx/internal/db/migrations"
	"ntx/internal/notification"
	"ntx/internal/shared/dto"
	"path/filepath"
	"strings"
	"testing"
)

func handleCreate(
	sqlDB *sql.DB,
	p dto.CreateNotificationParams,
) (dto.CreateNotificationResult, error) {
	return notification.NewCreateNotificationHandler(sqlDB).Handle(context.Background(), p)
}

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()

	path := filepath.Join(t.TempDir(), "test.db")

	sqlDB, err := db.Open(path)
	if err != nil {
		t.Fatalf("db.Open: unexpected error: %v", err)
	}

	t.Cleanup(func() { _ = sqlDB.Close() })

	d, err := db.New(sqlDB, migrations.All)
	if err != nil {
		t.Fatalf("db.New: unexpected error: %v", err)
	}

	if err := d.UpAllMigrations(); err != nil {
		t.Fatalf("UpAllMigrations: unexpected error: %v", err)
	}

	return sqlDB
}

func validParams() dto.CreateNotificationParams {
	return dto.CreateNotificationParams{
		Timestamp: "2026-08-24T10:00:00Z",
		AppName:   "Firefox",
		Summary:   "New message",
		Body:      "You have mail",
		Actions:   []string{"default"},
		Hints:     map[string]any{"urgency": float64(1)},
	}
}

func TestCreateInsertsRowWithUUIDv7ID(t *testing.T) {
	t.Parallel()

	sqlDB := openTestDB(t)

	res, err := handleCreate(sqlDB, validParams())
	if err != nil {
		t.Fatalf("HandleCreate: unexpected error: %v", err)
	}

	if len(res.ID) != 36 {
		t.Errorf("id = %q, want a 36-char UUID string", res.ID)
	}

	var appName, summary string
	err = sqlDB.QueryRow(`SELECT app_name, summary FROM notifications WHERE id = ?`, res.ID).
		Scan(&appName, &summary)
	if err != nil {
		t.Fatalf("query inserted row: %v", err)
	}

	if appName != "Firefox" || summary != "New message" {
		t.Errorf("row = (%q, %q), want (Firefox, New message)", appName, summary)
	}
}

func TestCreateWithIcon(t *testing.T) {
	t.Parallel()

	sqlDB := openTestDB(t)

	p := validParams()
	p.IconBase64 = base64.StdEncoding.EncodeToString([]byte("fake-png-bytes"))
	p.IconMime = "image/png"

	res, err := handleCreate(sqlDB, p)
	if err != nil {
		t.Fatalf("HandleCreate: unexpected error: %v", err)
	}

	var iconBase64, iconMime string
	err = sqlDB.QueryRow(`SELECT icon_base64, icon_mime FROM notifications WHERE id = ?`, res.ID).
		Scan(&iconBase64, &iconMime)
	if err != nil {
		t.Fatalf("query inserted row: %v", err)
	}

	if iconBase64 != p.IconBase64 || iconMime != "image/png" {
		t.Errorf("icon = (%q, %q), want (%q, image/png)", iconBase64, iconMime, p.IconBase64)
	}
}

func TestValidateRejectsEmptyAppName(t *testing.T) {
	t.Parallel()

	p := validParams()
	p.AppName = ""

	assertValidationField(t, p, "app_name")
}

func TestValidateRejectsEmptySummary(t *testing.T) {
	t.Parallel()

	p := validParams()
	p.Summary = ""

	assertValidationField(t, p, "summary")
}

func TestValidateRejectsBadTimestamp(t *testing.T) {
	t.Parallel()

	p := validParams()
	p.Timestamp = "not-a-timestamp"

	assertValidationField(t, p, "timestamp")
}

func TestValidateRejectsMismatchedIconFields(t *testing.T) {
	t.Parallel()

	p := validParams()
	p.IconBase64 = base64.StdEncoding.EncodeToString([]byte("data"))
	p.IconMime = ""

	assertValidationField(t, p, "icon_base64/icon_mime")
}

func TestValidateRejectsInvalidBase64Icon(t *testing.T) {
	t.Parallel()

	p := validParams()
	p.IconBase64 = "not-valid-base64!!"
	p.IconMime = "image/png"

	assertValidationField(t, p, "icon_base64")
}

func TestValidateRejectsOversizedIcon(t *testing.T) {
	t.Parallel()

	p := validParams()
	p.IconBase64 = base64.StdEncoding.EncodeToString(make([]byte, 8*1024*1024+1))
	p.IconMime = "image/png"

	assertValidationField(t, p, "icon_base64")
}

func TestCreateRejectsInvalidParamsWithoutInserting(t *testing.T) {
	t.Parallel()

	sqlDB := openTestDB(t)

	p := validParams()
	p.Summary = ""

	if _, err := handleCreate(sqlDB, p); err == nil {
		t.Fatal("HandleCreate: expected validation error, got nil")
	}

	var count int
	if err := sqlDB.QueryRow(`SELECT COUNT(*) FROM notifications`).Scan(&count); err != nil {
		t.Fatalf("count rows: %v", err)
	}

	if count != 0 {
		t.Errorf("row count = %d, want 0 after rejected insert", count)
	}
}

func assertValidationField(t *testing.T, p dto.CreateNotificationParams, wantField string) {
	t.Helper()

	err := notification.ValidateCreate(p)
	if err == nil {
		t.Fatal("Validate: expected error, got nil")
	}

	if !strings.Contains(err.Error(), wantField) {
		t.Errorf("Validate error = %q, want it to mention field %q", err.Error(), wantField)
	}
}
