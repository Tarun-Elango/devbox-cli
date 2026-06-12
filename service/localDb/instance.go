package localDb
// database operations for instances table
import (
	"database/sql"
	"fmt"
)

// InstanceRecord is a row from the instances table.
type InstanceRecord struct {
	ID            string
	AwsInstanceID string
	Name          string
	UserID        string
	IPAddress     sql.NullString
	Status        string
	InstanceType  sql.NullString
}

// ListInstancesByUserID returns all instances owned by userID.
func (db *DB) ListInstancesByUserID(userID string) ([]InstanceRecord, error) {
	rows, err := db.conn.Query(`
		SELECT id, aws_instance_id, name, user_id, ip_address, status, instance_type
		FROM instances
		WHERE user_id = ?
		ORDER BY created_at`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("list instances: %w", err)
	}
	defer rows.Close()

	var records []InstanceRecord
	for rows.Next() {
		var r InstanceRecord
		if err := rows.Scan(
			&r.ID,
			&r.AwsInstanceID,
			&r.Name,
			&r.UserID,
			&r.IPAddress,
			&r.Status,
			&r.InstanceType,
		); err != nil {
			return nil, fmt.Errorf("scan instance: %w", err)
		}
		records = append(records, r)
	}
	return records, rows.Err()
}

// GetInstanceByID returns the instance row for internal id, or sql.ErrNoRows if not found.
func (db *DB) GetInstanceByID(id string) (*InstanceRecord, error) {
	var r InstanceRecord
	err := db.conn.QueryRow(`
		SELECT id, aws_instance_id, name, user_id, ip_address, status, instance_type
		FROM instances
		WHERE id = ?`,
		id,
	).Scan(
		&r.ID,
		&r.AwsInstanceID,
		&r.Name,
		&r.UserID,
		&r.IPAddress,
		&r.Status,
		&r.InstanceType,
	)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

// GetInstanceByAwsInstanceIDAndUserID returns the instance row for awsInstanceID
// owned by userID, or sql.ErrNoRows if not found.
func (db *DB) GetInstanceByAwsInstanceIDAndUserID(awsInstanceID, userID string) (*InstanceRecord, error) {
	var r InstanceRecord
	err := db.conn.QueryRow(`
		SELECT id, aws_instance_id, name, user_id, ip_address, status, instance_type
		FROM instances
		WHERE aws_instance_id = ? AND user_id = ?`,
		awsInstanceID, userID,
	).Scan(
		&r.ID,
		&r.AwsInstanceID,
		&r.Name,
		&r.UserID,
		&r.IPAddress,
		&r.Status,
		&r.InstanceType,
	)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

// DeleteInstanceByAwsInstanceID removes an instance row by its AWS instance id.
// Referencing snapshots keep their row with box_id cleared (ON DELETE SET NULL).
func (db *DB) DeleteInstanceByAwsInstanceID(awsInstanceID string) error {
	_, err := db.conn.Exec(`DELETE FROM instances WHERE aws_instance_id = ?`, awsInstanceID)
	if err != nil {
		return fmt.Errorf("delete instance %s: %w", awsInstanceID, err)
	}
	return nil
}

// InsertInstance creates a new instance row owned by userID.
func (db *DB) InsertInstance(id, awsInstanceID, name, userID, status, instanceType string) error {
	_, err := db.conn.Exec(`
		INSERT INTO instances (id, aws_instance_id, name, user_id, status, instance_type)
		VALUES (?, ?, ?, ?, ?, ?)`,
		id, awsInstanceID, name, userID, status, instanceType,
	)
	if err != nil {
		return fmt.Errorf("insert instance: %w", err)
	}
	return nil
}

// SyncInstanceFromAWS updates cached fields from the latest AWS state.
// basically update the instance by aws instance id
func (db *DB) SyncInstanceFromAWS(awsInstanceID, status, ipAddress, instanceType, name string) error {
	_, err := db.conn.Exec(`
		UPDATE instances
		SET status = ?,
		    ip_address = ?,
		    instance_type = ?,
		    name = CASE WHEN ? != '' THEN ? ELSE name END,
		    updated_at = datetime('now')
		WHERE aws_instance_id = ?`,
		status, ipAddress, instanceType, name, name, awsInstanceID,
	)
	if err != nil {
		return fmt.Errorf("sync instance %s: %w", awsInstanceID, err)
	}
	return nil
}
