package localDb

const LocalUserID = "00000000-0000-0000-0000-000000000001"


// ensure the local user is in the database
func (db *DB) ensureLocalUser() error {
	_, err := db.conn.Exec(
		`INSERT OR IGNORE INTO users (id, username) VALUES (?, 'local')`,
		LocalUserID,
	)
	return err
}
