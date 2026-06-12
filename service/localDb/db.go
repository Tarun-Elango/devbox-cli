package localDb

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
)

const (
	dbDir  = ".devbox"
	dbFile = "devbox.db"
)

// DB wraps a local SQLite connection at ~/.devbox/devbox.db.
type DB struct {
	conn *sql.DB // common database connection, used by all other functions
}

// DBPath returns the absolute path to ~/.devbox/devbox.db.
func DBPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home dir: %w", err)
	}
	return filepath.Join(home, dbDir, dbFile), nil
}

// Open connects to ~/.devbox/devbox.db, creating the directory and schema if needed.
func Open() (*DB, error) {
	path, err := DBPath()
	if err != nil {
		return nil, err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, fmt.Errorf("create db dir: %w", err)
	}

	conn, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	conn.SetMaxOpenConns(1)

	db := &DB{conn: conn}

	if _, err := conn.Exec("PRAGMA foreign_keys = ON"); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("enable foreign keys: %w", err)
	}
	
	if err := db.migrate(); err != nil {
		_ = conn.Close()
		return nil, err
	}

	// ensure user exists
	if err := db.ensureLocalUser(); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("ensure local user: %w", err)
	}

	// ping the database to ensure it is connected
	if err := conn.Ping(); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("ping db: %w", err)
	}

	// return db connection
	return db, nil
}

func (db *DB) Close() error {
	return db.conn.Close()
}

func (db *DB) migrate() error {
	for _, stmt := range createTables {
		if _, err := db.conn.Exec(stmt); err != nil {
			return fmt.Errorf("migrate: %w", err)
		}
	}
	// CREATE TABLE IF NOT EXISTS does not alter existing tables; upgrade old DBs once.
	return db.migrateSnapshotsBoxFK()
}

func (db *DB) migrateSnapshotsBoxFK() error {
	var createSQL string
	err := db.conn.QueryRow(
		`SELECT sql FROM sqlite_master WHERE type = 'table' AND name = 'snapshots'`,
	).Scan(&createSQL)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		return fmt.Errorf("inspect snapshots schema: %w", err)
	}
	if strings.Contains(createSQL, "ON DELETE SET NULL") {
		return nil
	}

	tx, err := db.conn.Begin()
	if err != nil {
		return fmt.Errorf("migrate snapshots fk: %w", err)
	}
	defer tx.Rollback()

	stmts := []string{
		`CREATE TABLE snapshots_new (
  id         TEXT PRIMARY KEY,
  ami_id     TEXT NOT NULL UNIQUE,
  name       TEXT NOT NULL,
  user_id    TEXT NOT NULL REFERENCES users(id),
  box_id     TEXT REFERENCES instances(id) ON DELETE SET NULL,
  state      TEXT,
  created_at TEXT NOT NULL DEFAULT (datetime('now'))
)`,
		`INSERT INTO snapshots_new SELECT id, ami_id, name, user_id, box_id, state, created_at FROM snapshots`,
		`DROP TABLE snapshots`,
		`ALTER TABLE snapshots_new RENAME TO snapshots`,
		`CREATE INDEX IF NOT EXISTS idx_snapshots_box ON snapshots(box_id)`,
	}
	for _, stmt := range stmts {
		if _, err := tx.Exec(stmt); err != nil {
			return fmt.Errorf("migrate snapshots fk: %w", err)
		}
	}
	return tx.Commit()
}
