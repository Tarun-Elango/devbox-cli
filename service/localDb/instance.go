package localDb

// database operations for instances table
import (
	"database/sql"
	"fmt"
	"regexp"
	"strings"
)

var ec2InstanceIDPattern = regexp.MustCompile(`(?i)^i-[0-9a-f]{8}([0-9a-f]{9})?$`)

// InstanceRecord is a row from the instances table.
type InstanceRecord struct {
	ID              string
	AwsInstanceID   string
	Name            string
	UserID          string
	IPAddress       sql.NullString
	Status          string
	InstanceType    sql.NullString
	Region          sql.NullString
	Provider        sql.NullString
	OSFamily        sql.NullString
	IdleStopMinutes sql.NullInt64 // NULL = off
}

const instanceSelectColumns = `id, aws_instance_id, name, user_id, ip_address, status, instance_type, region, provider, os_family, idle_stop_minutes`

// scanInstanceRecord scans a row from the instances table into an InstanceRecord.
func scanInstanceRecord(scanner interface {
	Scan(dest ...any) error
}) (*InstanceRecord, error) {
	var r InstanceRecord
	err := scanner.Scan(
		&r.ID,
		&r.AwsInstanceID,
		&r.Name,
		&r.UserID,
		&r.IPAddress,
		&r.Status,
		&r.InstanceType,
		&r.Region,
		&r.Provider,
		&r.OSFamily,
		&r.IdleStopMinutes,
	)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

// ListInstancesByUserID returns all instances owned by userID.
func (db *DB) ListInstancesByUserID(userID string) ([]InstanceRecord, error) {
	rows, err := db.conn.Query(`
		SELECT `+instanceSelectColumns+`
		FROM instances
		WHERE user_id = ?
		ORDER BY created_at`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("list instances: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var records []InstanceRecord
	for rows.Next() {
		r, err := scanInstanceRecord(rows)
		if err != nil {
			return nil, fmt.Errorf("scan instance: %w", err)
		}
		records = append(records, *r)
	}
	return records, rows.Err()
}

// GetInstanceByID returns the instance row for internal id, or sql.ErrNoRows if not found.
func (db *DB) GetInstanceByID(id string) (*InstanceRecord, error) {
	row := db.conn.QueryRow(`
		SELECT `+instanceSelectColumns+`
		FROM instances
		WHERE id = ?`,
		id,
	)
	return scanInstanceRecord(row)
}

// GetInstanceByAwsInstanceIDAndUserID returns the instance row for awsInstanceID
// owned by userID, or sql.ErrNoRows if not found.
func (db *DB) GetInstanceByAwsInstanceIDAndUserID(awsInstanceID, userID string) (*InstanceRecord, error) {
	row := db.conn.QueryRow(`
		SELECT `+instanceSelectColumns+`
		FROM instances
		WHERE aws_instance_id = ? AND user_id = ?`,
		awsInstanceID, userID,
	)
	return scanInstanceRecord(row)
}

// GetInstanceByAwsInstanceID returns the instance row for awsInstanceID,
// or sql.ErrNoRows if not found.
func (db *DB) GetInstanceByAwsInstanceID(awsInstanceID string) (*InstanceRecord, error) {
	row := db.conn.QueryRow(`
		SELECT `+instanceSelectColumns+`
		FROM instances
		WHERE aws_instance_id = ?`,
		awsInstanceID,
	)
	return scanInstanceRecord(row)
}

// GetInstanceByNameAndUserID returns the instance row for name owned by userID,
// or sql.ErrNoRows if not found.
func (db *DB) GetInstanceByNameAndUserID(name, userID string) (*InstanceRecord, error) {
	row := db.conn.QueryRow(`
		SELECT `+instanceSelectColumns+`
		FROM instances
		WHERE name = ? AND user_id = ?`,
		name, userID,
	)
	return scanInstanceRecord(row)
}

// ResolveInstanceByNameOrAwsInstanceID resolves a user-provided box reference.
// The reference may be either an AWS instance id or a unique box name.
func (db *DB) ResolveInstanceByNameOrAwsInstanceID(ref, userID string) (*InstanceRecord, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return nil, fmt.Errorf("box id or name is required")
	}

	// Instance IDs are authoritative so existing ID-based commands keep working
	// even if another box has the same value as its name.
	byID, idErr := db.GetInstanceByAwsInstanceIDAndUserID(ref, userID)
	if idErr == nil {
		return byID, nil
	}
	if idErr != sql.ErrNoRows {
		return nil, idErr
	}

	byName, nameErr := db.GetInstanceByNameAndUserID(ref, userID)
	if nameErr == nil {
		return byName, nil
	}
	if nameErr != sql.ErrNoRows {
		return nil, nameErr
	}

	return nil, fmt.Errorf("box not found: %s", ref)
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
func (db *DB) InsertInstance(id, awsInstanceID, name, userID, status, instanceType, region, provider, osFamily string) error {
	name = strings.TrimSpace(name) // trim the name

	// before inserting, check if the name is available, in case there is conflict/race condition
	if err := db.ValidateInstanceNameAvailable(name, userID); err != nil {
		return err
	}

	_, err := db.conn.Exec(`
		INSERT INTO instances (id, aws_instance_id, name, user_id, status, instance_type, region, provider, os_family)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, awsInstanceID, name, userID, status, instanceType, region, provider, nullIfEmpty(osFamily),
	)
	if err != nil {
		// if the error is because the name already exists, return a specific error
		if strings.Contains(err.Error(), "UNIQUE constraint failed: instances.user_id, instances.name") {
			return fmt.Errorf("box name already exists: %s", name)
		}
		return fmt.Errorf("insert instance: %w", err)
	}
	return nil
}

func validateInstanceName(name string) error {
	name = strings.TrimSpace(name) // trim the name
	if name == "" {
		return fmt.Errorf("box name is required")
	}

	if ec2InstanceIDPattern.MatchString(name) {
		return fmt.Errorf("box name cannot look like an EC2 instance id: %s", name)
	}
	return nil
}

// ValidateInstanceNameAvailable verifies that name can be used as command identity
// for a new box owned by userID.
func (db *DB) ValidateInstanceNameAvailable(name, userID string) error {
	name = strings.TrimSpace(name)
	if err := validateInstanceName(name); err != nil {
		return err
	}

	_, err := db.GetInstanceByNameAndUserID(name, userID)
	if err == nil {
		return fmt.Errorf("box name already exists: %s", name)
	}
	if err != sql.ErrNoRows {
		return err
	}
	return nil
}

// ValidateInstanceNameAvailableForRename verifies that name can be used for an
// existing box. The current box may keep its own name.
func (db *DB) ValidateInstanceNameAvailableForRename(name, userID, currentAwsInstanceID string) error {
	name = strings.TrimSpace(name)
	if err := validateInstanceName(name); err != nil {
		return err
	}

	record, err := db.GetInstanceByNameAndUserID(name, userID)
	if err == sql.ErrNoRows { // if the name is not found, return nil
		return nil
	}
	if err != nil { // if there is an error, return it
		return err
	}
	// at this point, the name is found, but we need to check if its the same box

	if record.AwsInstanceID == currentAwsInstanceID { //if the aws instance id is the same as the current instance id, return nil, meaning its the same box
		return nil
	}
	return fmt.Errorf("box name already exists: %s", name)
}

// UpdateInstanceName updates a local instance name after AWS accepts the Name tag.
func (db *DB) UpdateInstanceName(awsInstanceID, userID, name string) error {
	name = strings.TrimSpace(name)
	if err := validateInstanceName(name); err != nil {
		return err
	}

	result, err := db.conn.Exec(`
		UPDATE instances
		SET name = ?,
		    updated_at = datetime('now')
		WHERE aws_instance_id = ? AND user_id = ?`,
		name, awsInstanceID, userID,
	)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed: instances.user_id, instances.name") {
			return fmt.Errorf("box name already exists: %s", name)
		}
		return fmt.Errorf("update instance name for %s: %w", awsInstanceID, err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("update instance name for %s: %w", awsInstanceID, err)
	}
	if rows == 0 {
		return fmt.Errorf("instance not found: %s", awsInstanceID)
	}
	return nil
}

// SetInstanceIdleStopMinutes sets idle_stop_minutes for an instance (nil = off).
func (db *DB) SetInstanceIdleStopMinutes(awsInstanceID string, minutes *int) error {
	var result sql.Result
	var err error
	if minutes == nil {
		result, err = db.conn.Exec(`
			UPDATE instances
			SET idle_stop_minutes = NULL,
			    updated_at = datetime('now')
			WHERE aws_instance_id = ?`,
			awsInstanceID,
		)
	} else {
		result, err = db.conn.Exec(`
			UPDATE instances
			SET idle_stop_minutes = ?,
			    updated_at = datetime('now')
			WHERE aws_instance_id = ?`,
			*minutes, awsInstanceID,
		)
	}
	if err != nil {
		return fmt.Errorf("set idle_stop_minutes for %s: %w", awsInstanceID, err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("set idle_stop_minutes for %s: %w", awsInstanceID, err)
	}
	if rows == 0 {
		return fmt.Errorf("instance not found: %s", awsInstanceID)
	}
	return nil
}

// NeedsAWSSync reports whether cached fields differ from the latest AWS values.
// Mirrors the field updates performed by SyncInstanceFromAWS.
func (r InstanceRecord) NeedsAWSSync(status, ipAddress, instanceType, name string) bool {
	if r.Status != status {
		return true
	}
	if StringValue(r.IPAddress) != ipAddress {
		return true
	}
	if StringValue(r.InstanceType) != instanceType {
		return true
	}
	if name != "" && r.Name != name {
		return true
	}
	return false
}

// StringValue returns n.String when Valid, otherwise "".
func StringValue(n sql.NullString) string {
	if !n.Valid {
		return ""
	}
	return n.String
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
