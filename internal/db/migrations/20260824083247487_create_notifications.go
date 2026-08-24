package migrations

import (
	"database/sql"
	"ntx/internal/db"
)

var _ db.Migration = m20260824083247487CreateNotifications{}

type m20260824083247487CreateNotifications struct{}

func (m20260824083247487CreateNotifications) Name() string {
	return "20260824083247487_create_notifications"
}

func (m20260824083247487CreateNotifications) Up(tx *sql.Tx) error {
	_, err := tx.Exec(`
		CREATE TABLE notifications (
			id             TEXT PRIMARY KEY,
			timestamp      TEXT NOT NULL,
			app_name       TEXT NOT NULL,
			replaces_id    INTEGER NOT NULL,
			summary        TEXT NOT NULL,
			body           TEXT NOT NULL,
			actions        TEXT NOT NULL,
			hints          TEXT NOT NULL,
			expire_timeout INTEGER NOT NULL,
			icon_base64    TEXT,
			icon_mime      TEXT
		)
	`)

	return err
}

func (m20260824083247487CreateNotifications) Down(tx *sql.Tx) error {
	_, err := tx.Exec(`DROP TABLE notifications`)

	return err
}
