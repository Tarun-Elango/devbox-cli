package localDb

import (
	"path/filepath"
	"testing"

	"outpost-cli/internal/sqliteutil"
)

func newLegacyInstancesDB(t *testing.T) *DB {
	t.Helper()

	conn, err := sqliteutil.Open(filepath.Join(t.TempDir(), "outpost.db"))
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	db := &DB{conn: conn}
	if _, err := conn.Exec(`CREATE TABLE users (
  id         TEXT PRIMARY KEY,
  username   TEXT NOT NULL DEFAULT 'local',
  created_at TEXT NOT NULL DEFAULT (datetime('now'))
)`); err != nil {
		t.Fatalf("create users table: %v", err)
	}
	if _, err := conn.Exec(`CREATE TABLE instances (
  id               TEXT PRIMARY KEY,
  aws_instance_id  TEXT NOT NULL UNIQUE,
  name             TEXT NOT NULL,
  user_id          TEXT NOT NULL REFERENCES users(id),
  ip_address       TEXT,
  status           TEXT NOT NULL DEFAULT 'pending',
  instance_type    TEXT,
  idle_stop_minutes  INTEGER,
  created_at       TEXT NOT NULL DEFAULT (datetime('now')),
  updated_at       TEXT
)`); err != nil {
		t.Fatalf("create legacy instances table: %v", err)
	}
	if err := db.ensureLocalUser(); err != nil {
		t.Fatalf("ensure local user: %v", err)
	}
	return db
}

func TestMigrateInstancesRegionProviderToleratesDuplicateColumn(t *testing.T) {
	db := newLegacyInstancesDB(t)

	if _, err := db.conn.Exec(`ALTER TABLE instances ADD COLUMN region TEXT`); err != nil {
		t.Fatalf("pre-add region column: %v", err)
	}

	if err := db.migrateInstancesRegionProvider(); err != nil {
		t.Fatalf("migrate after concurrent region add: %v", err)
	}

	cols, err := db.tableColumns("instances")
	if err != nil {
		t.Fatalf("inspect instances schema: %v", err)
	}
	if !cols["region"] || !cols["provider"] {
		t.Fatalf("instances schema missing region/provider: %#v", cols)
	}
}

func TestMigrateInstancesRegionProviderIsIdempotent(t *testing.T) {
	db := newLegacyInstancesDB(t)

	if err := db.migrateInstancesRegionProvider(); err != nil {
		t.Fatalf("first migrate: %v", err)
	}
	if err := db.migrateInstancesRegionProvider(); err != nil {
		t.Fatalf("second migrate: %v", err)
	}
}

func newLegacySnapshotsDB(t *testing.T) *DB {
	t.Helper()

	db := newLegacyInstancesDB(t)
	if err := db.migrateInstancesRegionProvider(); err != nil {
		t.Fatalf("migrate instances region/provider: %v", err)
	}
	if _, err := db.conn.Exec(`CREATE TABLE snapshots (
  id         TEXT PRIMARY KEY,
  ami_id     TEXT NOT NULL UNIQUE,
  name       TEXT NOT NULL,
  user_id    TEXT NOT NULL REFERENCES users(id),
  box_id     TEXT REFERENCES instances(id) ON DELETE SET NULL,
  state      TEXT,
  created_at TEXT NOT NULL DEFAULT (datetime('now'))
)`); err != nil {
		t.Fatalf("create legacy snapshots table: %v", err)
	}
	return db
}

func TestMigrateSnapshotsRegionProviderToleratesDuplicateColumn(t *testing.T) {
	db := newLegacySnapshotsDB(t)

	if _, err := db.conn.Exec(`ALTER TABLE snapshots ADD COLUMN region TEXT`); err != nil {
		t.Fatalf("pre-add region column: %v", err)
	}

	if err := db.migrateSnapshotsRegionProvider(); err != nil {
		t.Fatalf("migrate after concurrent region add: %v", err)
	}

	cols, err := db.tableColumns("snapshots")
	if err != nil {
		t.Fatalf("inspect snapshots schema: %v", err)
	}
	if !cols["region"] || !cols["provider"] {
		t.Fatalf("snapshots schema missing region/provider: %#v", cols)
	}
}

func TestMigrateSnapshotsRegionProviderIsIdempotent(t *testing.T) {
	db := newLegacySnapshotsDB(t)

	if err := db.migrateSnapshotsRegionProvider(); err != nil {
		t.Fatalf("first migrate: %v", err)
	}
	if err := db.migrateSnapshotsRegionProvider(); err != nil {
		t.Fatalf("second migrate: %v", err)
	}
}

func TestMigrateSnapshotsRegionProviderBackfillsFromBox(t *testing.T) {
	db := newLegacySnapshotsDB(t)

	if _, err := db.conn.Exec(`
		INSERT INTO instances (id, aws_instance_id, name, user_id, status, region, provider)
		VALUES ('box-1', 'i-1234567890abcdef0', 'my-box', ?, 'running', 'eu-west-1', 'aws')`,
		LocalUserID,
	); err != nil {
		t.Fatalf("insert legacy instance: %v", err)
	}
	if _, err := db.conn.Exec(`
		INSERT INTO snapshots (id, ami_id, name, user_id, box_id, state)
		VALUES ('snap-1', 'ami-1234567890abcdef0', 'before-upgrade', ?, 'box-1', 'pending')`,
		LocalUserID,
	); err != nil {
		t.Fatalf("insert legacy snapshot: %v", err)
	}

	if err := db.migrateSnapshotsRegionProvider(); err != nil {
		t.Fatalf("migrate snapshots region/provider: %v", err)
	}

	record, err := db.GetSnapshotByAmiIDAndUserID("ami-1234567890abcdef0", LocalUserID)
	if err != nil {
		t.Fatalf("get snapshot: %v", err)
	}
	if !record.Region.Valid || record.Region.String != "eu-west-1" {
		t.Fatalf("got region=%+v, want eu-west-1", record.Region)
	}
	if !record.Provider.Valid || record.Provider.String != "aws" {
		t.Fatalf("got provider=%+v, want aws", record.Provider)
	}
}
