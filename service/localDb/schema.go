package localDb

var createTables = []string{
	`CREATE TABLE IF NOT EXISTS users (
  id         TEXT PRIMARY KEY,
  username   TEXT NOT NULL DEFAULT 'local',
  created_at TEXT NOT NULL DEFAULT (datetime('now'))
)`,
	`CREATE TABLE IF NOT EXISTS instances (
  id               TEXT PRIMARY KEY,
  aws_instance_id  TEXT NOT NULL UNIQUE,
  name             TEXT NOT NULL,
  user_id          TEXT NOT NULL REFERENCES users(id),
  ip_address       TEXT,
  status           TEXT NOT NULL DEFAULT 'pending',
  instance_type    TEXT,
  created_at       TEXT NOT NULL DEFAULT (datetime('now')),
  updated_at       TEXT
)`,
	`CREATE INDEX IF NOT EXISTS idx_instances_user ON instances(user_id)`,
	`CREATE UNIQUE INDEX IF NOT EXISTS idx_instances_name_user ON instances(user_id, name)`,
	`CREATE TABLE IF NOT EXISTS snapshots (
  id         TEXT PRIMARY KEY,
  ami_id     TEXT NOT NULL UNIQUE,
  name       TEXT NOT NULL,
  user_id    TEXT NOT NULL REFERENCES users(id),
  box_id     TEXT REFERENCES instances(id),
  state      TEXT,
  created_at TEXT NOT NULL DEFAULT (datetime('now'))
)`,
	`CREATE INDEX IF NOT EXISTS idx_snapshots_box ON snapshots(box_id)`,
	`CREATE TABLE IF NOT EXISTS templates (
  id              TEXT PRIMARY KEY,
  user_id         TEXT NOT NULL REFERENCES users(id),
  name            TEXT NOT NULL,
  description     TEXT,
  startup_script  TEXT,
  created_at      TEXT NOT NULL DEFAULT (datetime('now')),
  UNIQUE(user_id, name)
)`,
}
