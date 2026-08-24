package db_test

import (
	"database/sql"
	"ntx/internal/db"
	"path/filepath"
	"testing"
)

type fakeMigration struct {
	name     string
	upStmt   string
	downStmt string
	failUp   bool
	failDown bool
}

func (m fakeMigration) Name() string { return m.name }

func (m fakeMigration) Up(tx *sql.Tx) error {
	if m.failUp {
		return errFail
	}

	_, err := tx.Exec(m.upStmt)

	return err
}

func (m fakeMigration) Down(tx *sql.Tx) error {
	if m.failDown {
		return errFail
	}

	_, err := tx.Exec(m.downStmt)

	return err
}

var errFail = &testError{"forced failure"}

type testError struct{ msg string }

func (e *testError) Error() string { return e.msg }

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()

	path := filepath.Join(t.TempDir(), "test.db")

	sqlDB, err := db.Open(path)
	if err != nil {
		t.Fatalf("db.Open: unexpected error: %v", err)
	}

	t.Cleanup(func() { _ = sqlDB.Close() })

	return sqlDB
}

func TestUpAllMigrationsAppliesInOrder(t *testing.T) {
	t.Parallel()

	sqlDB := openTestDB(t)

	migrations := []db.Migration{
		fakeMigration{name: "0001", upStmt: `CREATE TABLE a (id TEXT)`, downStmt: `DROP TABLE a`},
		fakeMigration{name: "0002", upStmt: `CREATE TABLE b (id TEXT)`, downStmt: `DROP TABLE b`},
	}

	d, err := db.New(sqlDB, migrations)
	if err != nil {
		t.Fatalf("db.New: unexpected error: %v", err)
	}

	if err := d.UpAllMigrations(); err != nil {
		t.Fatalf("UpAllMigrations: unexpected error: %v", err)
	}

	for _, table := range []string{"a", "b"} {
		if _, err := sqlDB.Exec("SELECT * FROM " + table); err != nil {
			t.Errorf("table %s: expected to exist, query failed: %v", table, err)
		}
	}
}

func TestUpAllMigrationsIsIdempotent(t *testing.T) {
	t.Parallel()

	sqlDB := openTestDB(t)

	migrations := []db.Migration{
		fakeMigration{name: "0001", upStmt: `CREATE TABLE a (id TEXT)`, downStmt: `DROP TABLE a`},
	}

	d, err := db.New(sqlDB, migrations)
	if err != nil {
		t.Fatalf("db.New: unexpected error: %v", err)
	}

	if err := d.UpAllMigrations(); err != nil {
		t.Fatalf("UpAllMigrations (1st): unexpected error: %v", err)
	}

	if err := d.UpAllMigrations(); err != nil {
		t.Fatalf("UpAllMigrations (2nd): unexpected error: %v", err)
	}
}

func TestDownAllMigrationsReversesInOrder(t *testing.T) {
	t.Parallel()

	sqlDB := openTestDB(t)

	migrations := []db.Migration{
		fakeMigration{name: "0001", upStmt: `CREATE TABLE a (id TEXT)`, downStmt: `DROP TABLE a`},
		fakeMigration{name: "0002", upStmt: `CREATE TABLE b (id TEXT)`, downStmt: `DROP TABLE b`},
	}

	d, err := db.New(sqlDB, migrations)
	if err != nil {
		t.Fatalf("db.New: unexpected error: %v", err)
	}

	if err := d.UpAllMigrations(); err != nil {
		t.Fatalf("UpAllMigrations: unexpected error: %v", err)
	}

	if err := d.DownAllMigrations(); err != nil {
		t.Fatalf("DownAllMigrations: unexpected error: %v", err)
	}

	for _, table := range []string{"a", "b"} {
		if _, err := sqlDB.Exec("SELECT * FROM " + table); err == nil {
			t.Errorf("table %s: expected to be dropped, query succeeded", table)
		}
	}
}

func TestApplyUpRollsBackOnFailure(t *testing.T) {
	t.Parallel()

	sqlDB := openTestDB(t)

	migrations := []db.Migration{
		fakeMigration{name: "0001", failUp: true},
	}

	d, err := db.New(sqlDB, migrations)
	if err != nil {
		t.Fatalf("db.New: unexpected error: %v", err)
	}

	if err := d.UpAllMigrations(); err == nil {
		t.Fatal("UpAllMigrations: expected error, got nil")
	}

	var count int
	if err := sqlDB.QueryRow(`SELECT COUNT(*) FROM migrations WHERE name = '0001'`).
		Scan(&count); err != nil {
		t.Fatalf("query migrations table: %v", err)
	}

	if count != 0 {
		t.Errorf("migrations table has %d rows for failed migration, want 0", count)
	}
}

func TestUpOneMigration(t *testing.T) {
	t.Parallel()

	sqlDB := openTestDB(t)

	migrations := []db.Migration{
		fakeMigration{name: "0001", upStmt: `CREATE TABLE a (id TEXT)`, downStmt: `DROP TABLE a`},
		fakeMigration{name: "0002", upStmt: `CREATE TABLE b (id TEXT)`, downStmt: `DROP TABLE b`},
	}

	d, err := db.New(sqlDB, migrations)
	if err != nil {
		t.Fatalf("db.New: unexpected error: %v", err)
	}

	applied, err := d.UpOneMigration()
	if err != nil {
		t.Fatalf("UpOneMigration: unexpected error: %v", err)
	}

	if !applied {
		t.Fatal("UpOneMigration: applied = false, want true")
	}

	if _, err := sqlDB.Exec("SELECT * FROM a"); err != nil {
		t.Errorf("table a: expected to exist: %v", err)
	}

	if _, err := sqlDB.Exec("SELECT * FROM b"); err == nil {
		t.Error("table b: expected not to exist yet")
	}
}
