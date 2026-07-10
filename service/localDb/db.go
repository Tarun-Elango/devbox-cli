package localDb

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"outpost-cli/internal/config"
	"outpost-cli/internal/sqliteutil"
)

const (
	dbDir  = ".outpost"
	dbFile = "outpost.db"
)

// DB wraps a local SQLite connection at ~/.outpost/outpost.db.
type DB struct {
	conn *sql.DB // common database connection, used by all other functions
}

// DBPath returns the absolute path to ~/.outpost/outpost.db.
func DBPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home dir: %w", err)
	}
	return filepath.Join(home, dbDir, dbFile), nil
}

// Open connects to ~/.outpost/outpost.db, creating the directory and schema if needed.
func Open() (*DB, error) {
	//backup.RestoreDBIfNeeded()
	path, err := DBPath()
	if err != nil {
		return nil, err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, fmt.Errorf("create db dir: %w", err)
	}

	conn, err := sqliteutil.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	db := &DB{conn: conn}

	if err := db.migrate(); err != nil { // got to make sure all the migrations are run
		_ = conn.Close()
		return nil, err
	}

	// ensure user exists
	if err := db.ensureLocalUser(); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("ensure local user: %w", err)
	}

	if err := db.seedDefaultTemplates(); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("seed default templates: %w", err)
	}

	// ping the database to ensure it is connected
	if err := conn.Ping(); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("ping db: %w", err)
	}

	// return db connection
	return db, nil
}

// OpenExisting opens path if the database file already exists, without creating
// directories or seeding default templates. Returns (nil, nil) when the file is absent.
// called by cmd/uninstall.go
func OpenExisting(path string) (*DB, error) {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("stat db: %w", err)
	}

	conn, err := sqliteutil.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	db := &DB{conn: conn}
	if err := db.migrate(); err != nil {
		_ = conn.Close()
		return nil, err
	}
	if err := conn.Ping(); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("ping db: %w", err)
	}
	return db, nil
}

// CountTemplates returns the number of rows in the templates table.
func (db *DB) CountTemplates() (int, error) {
	var n int
	if err := db.conn.QueryRow(`SELECT COUNT(*) FROM templates`).Scan(&n); err != nil {
		return 0, fmt.Errorf("count templates: %w", err)
	}
	return n, nil
}

// CountSnapshots returns the number of rows in the snapshots table.
func (db *DB) CountSnapshots() (int, error) {
	var n int
	if err := db.conn.QueryRow(`SELECT COUNT(*) FROM snapshots`).Scan(&n); err != nil {
		return 0, fmt.Errorf("count snapshots: %w", err)
	}
	return n, nil
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
	if err := db.migrateInstancesRegionProvider(); err != nil {
		return err
	}
	return db.migrateSnapshotsRegionProvider()
}

// one time migration to add region and provider columns to instances table
func (db *DB) migrateInstancesRegionProvider() error {
	cols, err := db.tableColumns("instances")
	if err != nil {
		return err
	}
	if !cols["region"] {
		if _, err := db.conn.Exec(`ALTER TABLE instances ADD COLUMN region TEXT`); err != nil && !isDuplicateColumnError(err, "region") {
			return fmt.Errorf("migrate instances region: %w", err)
		}
	}
	if !cols["provider"] {
		if _, err := db.conn.Exec(`ALTER TABLE instances ADD COLUMN provider TEXT`); err != nil && !isDuplicateColumnError(err, "provider") {
			return fmt.Errorf("migrate instances provider: %w", err)
		}
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("migrate instances region/provider backfill: %w", err)
	}
	defaultRegion := cfg.AwsRegion
	_, err = db.conn.Exec(`
		UPDATE instances
		SET region = COALESCE(NULLIF(region, ''), ?),
		    provider = COALESCE(NULLIF(provider, ''), 'aws')
		WHERE region IS NULL OR region = '' OR provider IS NULL OR provider = ''`,
		defaultRegion,
	)
	if err != nil {
		return fmt.Errorf("migrate instances region/provider backfill: %w", err)
	}
	return nil
}

// one time migration to add region and provider columns to snapshots table.
// Snapshots used to derive their region solely from the source box, which
// breaks once the box is deleted; storing region/provider on the snapshot
// itself keeps it independently addressable.
func (db *DB) migrateSnapshotsRegionProvider() error {
	cols, err := db.tableColumns("snapshots")
	if err != nil {
		return err
	}
	if !cols["region"] {
		if _, err := db.conn.Exec(`ALTER TABLE snapshots ADD COLUMN region TEXT`); err != nil && !isDuplicateColumnError(err, "region") {
			return fmt.Errorf("migrate snapshots region: %w", err)
		}
	}
	if !cols["provider"] {
		if _, err := db.conn.Exec(`ALTER TABLE snapshots ADD COLUMN provider TEXT`); err != nil && !isDuplicateColumnError(err, "provider") {
			return fmt.Errorf("migrate snapshots provider: %w", err)
		}
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("migrate snapshots region/provider backfill: %w", err)
	}
	defaultRegion := cfg.AwsRegion
	_, err = db.conn.Exec(`
		UPDATE snapshots
		SET region = COALESCE(NULLIF(region, ''), (SELECT region FROM instances WHERE instances.id = snapshots.box_id), NULLIF(?, '')),
		    provider = COALESCE(NULLIF(provider, ''), (SELECT provider FROM instances WHERE instances.id = snapshots.box_id), 'aws')
		WHERE region IS NULL OR region = '' OR provider IS NULL OR provider = ''`,
		defaultRegion,
	)
	if err != nil {
		return fmt.Errorf("migrate snapshots region/provider backfill: %w", err)
	}
	return nil
}

func isDuplicateColumnError(err error, column string) bool {
	return err != nil && strings.Contains(err.Error(), "duplicate column name: "+column)
}

// returns the list of columns in the table
func (db *DB) tableColumns(table string) (map[string]bool, error) {
	rows, err := db.conn.Query(`PRAGMA table_info(` + table + `)`)
	if err != nil {
		return nil, fmt.Errorf("inspect %s schema: %w", table, err)
	}
	defer func() { _ = rows.Close() }()

	cols := make(map[string]bool)
	for rows.Next() {
		var cid int
		var name, colType string
		var notNull, pk int
		var dflt sql.NullString
		if err := rows.Scan(&cid, &name, &colType, &notNull, &dflt, &pk); err != nil {
			return nil, fmt.Errorf("scan %s column: %w", table, err)
		}
		cols[name] = true
	}
	return cols, rows.Err()
}

// func (db *DB) migrateSnapshotsBoxFK() error {
// 	var createSQL string
// 	err := db.conn.QueryRow(
// 		`SELECT sql FROM sqlite_master WHERE type = 'table' AND name = 'snapshots'`,
// 	).Scan(&createSQL)
// 	if err == sql.ErrNoRows {
// 		return nil
// 	}
// 	if err != nil {
// 		return fmt.Errorf("inspect snapshots schema: %w", err)
// 	}
// 	if strings.Contains(createSQL, "ON DELETE SET NULL") {
// 		return nil
// 	}

// 	tx, err := db.conn.Begin()
// 	if err != nil {
// 		return fmt.Errorf("migrate snapshots fk: %w", err)
// 	}
// 	defer tx.Rollback()

// 	stmts := []string{
// 		`CREATE TABLE snapshots_new (
//   id         TEXT PRIMARY KEY,
//   ami_id     TEXT NOT NULL UNIQUE,
//   name       TEXT NOT NULL,
//   user_id    TEXT NOT NULL REFERENCES users(id),
//   box_id     TEXT REFERENCES instances(id) ON DELETE SET NULL,
//   state      TEXT,
//   created_at TEXT NOT NULL DEFAULT (datetime('now'))
// )`,
// 		`INSERT INTO snapshots_new SELECT id, ami_id, name, user_id, box_id, state, created_at FROM snapshots`,
// 		`DROP TABLE snapshots`,
// 		`ALTER TABLE snapshots_new RENAME TO snapshots`,
// 		`CREATE INDEX IF NOT EXISTS idx_snapshots_box ON snapshots(box_id)`,
// 	}
// 	for _, stmt := range stmts {
// 		if _, err := tx.Exec(stmt); err != nil {
// 			return fmt.Errorf("migrate snapshots fk: %w", err)
// 		}
// 	}
// 	return tx.Commit()
// }
