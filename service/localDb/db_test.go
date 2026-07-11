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

func TestMigrateInstancesOSFamilyToleratesDuplicateColumn(t *testing.T) {
	db := newLegacyInstancesDB(t)

	if _, err := db.conn.Exec(`ALTER TABLE instances ADD COLUMN os_family TEXT`); err != nil {
		t.Fatalf("pre-add os_family column: %v", err)
	}

	if err := db.migrateInstancesOSFamily(); err != nil {
		t.Fatalf("migrate after concurrent os_family add: %v", err)
	}

	cols, err := db.tableColumns("instances")
	if err != nil {
		t.Fatalf("inspect instances schema: %v", err)
	}
	if !cols["os_family"] {
		t.Fatalf("instances schema missing os_family: %#v", cols)
	}
}

func TestMigrateInstancesOSFamilyIsIdempotent(t *testing.T) {
	db := newLegacyInstancesDB(t)

	if err := db.migrateInstancesOSFamily(); err != nil {
		t.Fatalf("first migrate: %v", err)
	}
	if err := db.migrateInstancesOSFamily(); err != nil {
		t.Fatalf("second migrate: %v", err)
	}

	cols, err := db.tableColumns("instances")
	if err != nil {
		t.Fatalf("inspect instances schema: %v", err)
	}
	if !cols["os_family"] {
		t.Fatalf("instances schema missing os_family: %#v", cols)
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
	if err := db.migrateSnapshotsOSFamily(); err != nil {
		t.Fatalf("migrate snapshots os_family: %v", err)
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

func newLegacyTemplatesDB(t *testing.T) *DB {
	t.Helper()

	db := newLegacyInstancesDB(t)
	if _, err := db.conn.Exec(`CREATE TABLE templates (
  id              TEXT PRIMARY KEY,
  user_id         TEXT NOT NULL REFERENCES users(id),
  name            TEXT NOT NULL,
  description     TEXT,
  startup_script  TEXT,
  created_at      TEXT NOT NULL DEFAULT (datetime('now')),
  UNIQUE(user_id, name)
)`); err != nil {
		t.Fatalf("create legacy templates table: %v", err)
	}
	return db
}

func TestMigrateTemplatesOSFamilyToleratesDuplicateColumn(t *testing.T) {
	db := newLegacyTemplatesDB(t)

	if _, err := db.conn.Exec(`ALTER TABLE templates ADD COLUMN os_family TEXT`); err != nil {
		t.Fatalf("pre-add os_family column: %v", err)
	}

	if err := db.migrateTemplatesOSFamily(); err != nil {
		t.Fatalf("migrate after concurrent os_family add: %v", err)
	}

	cols, err := db.tableColumns("templates")
	if err != nil {
		t.Fatalf("inspect templates schema: %v", err)
	}
	if !cols["os_family"] {
		t.Fatalf("templates schema missing os_family: %#v", cols)
	}
}

func TestMigrateTemplatesOSFamilyIsIdempotent(t *testing.T) {
	db := newLegacyTemplatesDB(t)

	if err := db.migrateTemplatesOSFamily(); err != nil {
		t.Fatalf("first migrate: %v", err)
	}
	if err := db.migrateTemplatesOSFamily(); err != nil {
		t.Fatalf("second migrate: %v", err)
	}

	cols, err := db.tableColumns("templates")
	if err != nil {
		t.Fatalf("inspect templates schema: %v", err)
	}
	if !cols["os_family"] {
		t.Fatalf("templates schema missing os_family: %#v", cols)
	}
}

func TestOpenExistingMissingAndCounts(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	missing := filepath.Join(t.TempDir(), "missing.db")
	db, err := OpenExisting(missing)
	if err != nil {
		t.Fatalf("OpenExisting(missing) error = %v", err)
	}
	if db != nil {
		t.Fatal("OpenExisting(missing) = non-nil, want nil")
	}

	seeded, err := Open()
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	_ = seeded.Close()

	path, err := DBPath()
	if err != nil {
		t.Fatalf("DBPath() error = %v", err)
	}
	db, err = OpenExisting(path)
	if err != nil {
		t.Fatalf("OpenExisting(existing) error = %v", err)
	}
	if db == nil {
		t.Fatal("OpenExisting(existing) = nil, want db")
	}
	defer func() { _ = db.Close() }()

	templates, err := db.CountTemplates()
	if err != nil {
		t.Fatalf("CountTemplates() error = %v", err)
	}
	if templates == 0 {
		t.Fatal("CountTemplates() = 0, want seeded defaults")
	}
	snapshots, err := db.CountSnapshots()
	if err != nil {
		t.Fatalf("CountSnapshots() error = %v", err)
	}
	if snapshots != 0 {
		t.Fatalf("CountSnapshots() = %d, want 0", snapshots)
	}
}
