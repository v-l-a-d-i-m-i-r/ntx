// Package db implements a minimal, reversible SQLite migration engine.
// Migrations are supplied by the caller as an ordered slice; applied
// migrations are tracked by name in a "migrations" table.
package db

import (
	"database/sql"
	"fmt"
)

// Migration is a single reversible schema change.
type Migration interface {
	// Name uniquely identifies the migration, e.g. "20260824143022917_create_notifications".
	Name() string
	// Up applies the migration within tx.
	Up(tx *sql.Tx) error
	// Down reverts the migration within tx.
	Down(tx *sql.Tx) error
}

// DB wraps a *sql.DB with an ordered list of migrations.
type DB struct {
	sqlDB      *sql.DB
	migrations []Migration
}

// New wraps sqlDB with the given ordered migrations and ensures the
// migrations bookkeeping table exists.
func New(sqlDB *sql.DB, migrations []Migration) (*DB, error) {
	d := &DB{sqlDB: sqlDB, migrations: migrations}

	if err := d.ensureMigrationsTable(); err != nil {
		return nil, err
	}

	return d, nil
}

// Conn returns the underlying *sql.DB for use by callers outside the
// migration engine.
func (d *DB) Conn() *sql.DB {
	return d.sqlDB
}

func (d *DB) ensureMigrationsTable() error {
	const stmt = `CREATE TABLE IF NOT EXISTS migrations (
		name TEXT PRIMARY KEY,
		applied_at TEXT NOT NULL
	)`

	if _, err := d.sqlDB.Exec(stmt); err != nil {
		return fmt.Errorf("db: create migrations table: %w", err)
	}

	return nil
}

func (d *DB) appliedNames() (map[string]bool, error) {
	rows, err := d.sqlDB.Query(`SELECT name FROM migrations`)
	if err != nil {
		return nil, fmt.Errorf("db: query applied migrations: %w", err)
	}
	defer rows.Close() //nolint:errcheck // read-only cleanup, nothing actionable on failure

	applied := make(map[string]bool)

	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("db: scan applied migration: %w", err)
		}

		applied[name] = true
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("db: iterate applied migrations: %w", err)
	}

	return applied, nil
}

// UpAllMigrations applies every pending migration, in order.
func (d *DB) UpAllMigrations() error {
	applied, err := d.appliedNames()
	if err != nil {
		return err
	}

	for _, m := range d.migrations {
		if applied[m.Name()] {
			continue
		}

		if err := d.applyUp(m); err != nil {
			return err
		}
	}

	return nil
}

// DownAllMigrations reverts every applied migration, in reverse order.
func (d *DB) DownAllMigrations() error {
	applied, err := d.appliedNames()
	if err != nil {
		return err
	}

	for i := len(d.migrations) - 1; i >= 0; i-- {
		m := d.migrations[i]
		if !applied[m.Name()] {
			continue
		}

		if err := d.applyDown(m); err != nil {
			return err
		}
	}

	return nil
}

// UpOneMigration applies the next pending migration, if any. It reports
// whether a migration was applied.
func (d *DB) UpOneMigration() (bool, error) {
	applied, err := d.appliedNames()
	if err != nil {
		return false, err
	}

	for _, m := range d.migrations {
		if applied[m.Name()] {
			continue
		}

		if err := d.applyUp(m); err != nil {
			return false, err
		}

		return true, nil
	}

	return false, nil
}

// DownOneMigration reverts the most recently applied migration, if any. It
// reports whether a migration was reverted.
func (d *DB) DownOneMigration() (bool, error) {
	applied, err := d.appliedNames()
	if err != nil {
		return false, err
	}

	for i := len(d.migrations) - 1; i >= 0; i-- {
		m := d.migrations[i]
		if !applied[m.Name()] {
			continue
		}

		if err := d.applyDown(m); err != nil {
			return false, err
		}

		return true, nil
	}

	return false, nil
}

func (d *DB) applyUp(m Migration) error {
	tx, err := d.sqlDB.Begin()
	if err != nil {
		return fmt.Errorf("db: begin tx for migration %s: %w", m.Name(), err)
	}
	defer tx.Rollback() //nolint:errcheck // no-op if already committed

	if err := m.Up(tx); err != nil {
		return fmt.Errorf("db: apply migration %s: %w", m.Name(), err)
	}

	if _, err := tx.Exec(
		`INSERT INTO migrations (name, applied_at) VALUES (?, datetime('now'))`,
		m.Name(),
	); err != nil {
		return fmt.Errorf("db: record migration %s: %w", m.Name(), err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("db: commit migration %s: %w", m.Name(), err)
	}

	return nil
}

func (d *DB) applyDown(m Migration) error {
	tx, err := d.sqlDB.Begin()
	if err != nil {
		return fmt.Errorf("db: begin tx for migration %s: %w", m.Name(), err)
	}
	defer tx.Rollback() //nolint:errcheck // no-op if already committed

	if err := m.Down(tx); err != nil {
		return fmt.Errorf("db: revert migration %s: %w", m.Name(), err)
	}

	if _, err := tx.Exec(`DELETE FROM migrations WHERE name = ?`, m.Name()); err != nil {
		return fmt.Errorf("db: unrecord migration %s: %w", m.Name(), err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("db: commit revert of migration %s: %w", m.Name(), err)
	}

	return nil
}
