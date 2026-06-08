package localDb

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

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
	return nil
}
