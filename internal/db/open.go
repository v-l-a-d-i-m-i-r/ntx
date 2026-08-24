package db

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite" // cgo-free sqlite driver
)

// Open opens a SQLite database at path with WAL journaling and a busy
// timeout, so concurrent connections queue on writes instead of failing
// immediately with SQLITE_BUSY.
func Open(path string) (*sql.DB, error) {
	dsn := fmt.Sprintf("file:%s?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)", path)

	sqlDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("db: open %s: %w", path, err)
	}

	if err := sqlDB.Ping(); err != nil {
		_ = sqlDB.Close()

		return nil, fmt.Errorf("db: ping %s: %w", path, err)
	}

	return sqlDB, nil
}
