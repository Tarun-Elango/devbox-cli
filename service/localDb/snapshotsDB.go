package localDb

import (
	"database/sql"
	"fmt"
)

// SnapshotRecord is a row from the snapshots table.
type SnapshotRecord struct {
	ID        string
	AmiID     string
	Name      string
	UserID    string
	BoxID     sql.NullString
	State     sql.NullString
	CreatedAt string
}

// SnapshotWithBoxAwsID is a snapshot row joined with the source box's AWS instance id.
type SnapshotWithBoxAwsID struct {
	SnapshotRecord
	BoxAwsID sql.NullString
}

// InsertSnapshot creates a new snapshot row owned by userID.
func (db *DB) InsertSnapshot(id, amiID, name, userID, boxID, state string) error {
	_, err := db.conn.Exec(`
		INSERT INTO snapshots (id, ami_id, name, user_id, box_id, state)
		VALUES (?, ?, ?, ?, ?, ?)`,
		id, amiID, name, userID, nullIfEmpty(boxID), nullIfEmpty(state),
	)
	if err != nil {
		return fmt.Errorf("insert snapshot: %w", err)
	}
	return nil
}

// SnapshotNameTaken reports whether userID already has a snapshot named name.
func (db *DB) SnapshotNameTaken(userID, name string) (bool, error) {
	var exists int
	err := db.conn.QueryRow(`
		SELECT 1 FROM snapshots WHERE user_id = ? AND name = ? LIMIT 1`,
		userID, name,
	).Scan(&exists)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("check snapshot name: %w", err)
	}
	return true, nil
}

// SnapshotBelongsToUser reports whether snapshot id is owned by userID.
func (db *DB) SnapshotBelongsToUser(id, userID string) (bool, error) {
	var exists int
	err := db.conn.QueryRow(`
		SELECT 1 FROM snapshots WHERE id = ? AND user_id = ? LIMIT 1`,
		id, userID,
	).Scan(&exists)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("check snapshot owner: %w", err)
	}
	return true, nil
}

// GetSnapshotByAmiIDAndUserID returns the snapshot row for amiID owned by userID,
// or sql.ErrNoRows if not found.
func (db *DB) GetSnapshotByAmiIDAndUserID(amiID, userID string) (*SnapshotRecord, error) {
	var r SnapshotRecord
	err := db.conn.QueryRow(`
		SELECT id, ami_id, name, user_id, box_id, state, created_at
		FROM snapshots
		WHERE ami_id = ? AND user_id = ?`,
		amiID, userID,
	).Scan(
		&r.ID,
		&r.AmiID,
		&r.Name,
		&r.UserID,
		&r.BoxID,
		&r.State,
		&r.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

// ListSnapshotsByBoxIDAndUserID returns snapshots for boxID owned by userID.
func (db *DB) ListSnapshotsByBoxIDAndUserID(boxID, userID string) ([]SnapshotRecord, error) {
	rows, err := db.conn.Query(`
		SELECT id, ami_id, name, user_id, box_id, state, created_at
		FROM snapshots
		WHERE box_id = ? AND user_id = ?
		ORDER BY created_at`,
		boxID, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("list snapshots by box: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var records []SnapshotRecord
	for rows.Next() {
		var r SnapshotRecord
		if err := rows.Scan(
			&r.ID,
			&r.AmiID,
			&r.Name,
			&r.UserID,
			&r.BoxID,
			&r.State,
			&r.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan snapshot: %w", err)
		}
		records = append(records, r)
	}
	return records, rows.Err()
}

// ListSnapshotsByUserID returns all snapshots owned by userID.
func (db *DB) ListSnapshotsByUserID(userID string) ([]SnapshotRecord, error) {
	rows, err := db.conn.Query(`
		SELECT id, ami_id, name, user_id, box_id, state, created_at
		FROM snapshots
		WHERE user_id = ?
		ORDER BY created_at`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("list snapshots: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var records []SnapshotRecord
	for rows.Next() {
		var r SnapshotRecord
		if err := rows.Scan(
			&r.ID,
			&r.AmiID,
			&r.Name,
			&r.UserID,
			&r.BoxID,
			&r.State,
			&r.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan snapshot: %w", err)
		}
		records = append(records, r)
	}
	return records, rows.Err()
}

// ListSnapshotsByUserIDWithBoxAwsID returns snapshots for userID with aws_instance_id joined from instances.
func (db *DB) ListSnapshotsByUserIDWithBoxAwsID(userID string) ([]SnapshotWithBoxAwsID, error) {
	rows, err := db.conn.Query(`
		SELECT s.id, s.ami_id, s.name, s.user_id, s.box_id, s.state, s.created_at, i.aws_instance_id
		FROM snapshots s
		LEFT JOIN instances i ON s.box_id = i.id AND i.user_id = s.user_id
		WHERE s.user_id = ?
		ORDER BY s.created_at`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("list snapshots with box aws id: %w", err)
	}
	defer func() { _ = rows.Close() }()

	// convert the rows to the custom struct
	var records []SnapshotWithBoxAwsID
	for rows.Next() {
		var r SnapshotWithBoxAwsID
		if err := rows.Scan(
			&r.ID,
			&r.AmiID,
			&r.Name,
			&r.UserID,
			&r.BoxID,
			&r.State,
			&r.CreatedAt,
			&r.BoxAwsID,
		); err != nil {
			return nil, fmt.Errorf("scan snapshot with box aws id: %w", err)
		}
		records = append(records, r)
	}
	return records, rows.Err()
}

// UpdateSnapshotState updates the cached state for a snapshot AMI.
func (db *DB) UpdateSnapshotState(amiID, state string) error {
	_, err := db.conn.Exec(`UPDATE snapshots SET state = ? WHERE ami_id = ?`, state, amiID)
	if err != nil {
		return fmt.Errorf("update snapshot state %s: %w", amiID, err)
	}
	return nil
}

// DeleteSnapshotByAmiID removes a snapshot row by its AMI id.
func (db *DB) DeleteSnapshotByAmiID(amiID string) error {
	_, err := db.conn.Exec(`DELETE FROM snapshots WHERE ami_id = ?`, amiID)
	if err != nil {
		return fmt.Errorf("delete snapshot %s: %w", amiID, err)
	}
	return nil
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}
