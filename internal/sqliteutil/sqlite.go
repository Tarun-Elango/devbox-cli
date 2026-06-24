package sqliteutil

import (
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"

	_ "modernc.org/sqlite"
)

const (
	BusyTimeoutMs = 5000
	maxRetries    = 5
)

// DSN returns a modernc.org/sqlite connection string with WAL, busy timeout,
// and foreign keys applied to every pooled connection.
func DSN(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		abs = path
	}
	return fmt.Sprintf(
		"file:%s?_pragma=busy_timeout(%d)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)",
		filepath.ToSlash(abs),
		BusyTimeoutMs,
	)
}

// Open connects to path with SQLite concurrency pragmas and a single connection.
func Open(path string) (*sql.DB, error) {
	conn, err := sql.Open("sqlite", DSN(path))
	if err != nil {
		return nil, err
	}
	conn.SetMaxOpenConns(1)
	return conn, nil
}

// IsBusy reports whether err is a SQLITE_BUSY-family error.
// by checking the error code, we can determine if the error is a SQLITE_BUSY-family error
func IsBusy(err error) bool {
	var sqliteErr *sqlite.Error
	if errors.As(err, &sqliteErr) {
		switch sqliteErr.Code() {
		case sqlite3.SQLITE_BUSY, sqlite3.SQLITE_BUSY_RECOVERY, sqlite3.SQLITE_BUSY_SNAPSHOT, sqlite3.SQLITE_BUSY_TIMEOUT:
			return true
		}
	}
	return false
}

// WithRetry runs fn up to maxRetries times when SQLITE_BUSY is returned.
func WithRetry(fn func() error) error {
	var err error
	for attempt := 0; attempt < maxRetries; attempt++ {
		err = fn()
		if err == nil || !IsBusy(err) {
			return err
		}
		time.Sleep(time.Duration(attempt+1) * 100 * time.Millisecond)
	}
	return err
}
