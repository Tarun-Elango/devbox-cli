package localDb

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
)

// TemplateRecord is a row from the templates table.
type TemplateRecord struct {
	ID            string
	UserID        string
	Name          string
	Description   sql.NullString
	StartupScript sql.NullString
	OSFamily      sql.NullString
	CreatedAt     string
}

const templateSelectColumns = `id, user_id, name, description, startup_script, os_family, created_at`

// sql row to template record
func scanTemplateRecord(scanner interface {
	Scan(dest ...any) error
}) (*TemplateRecord, error) {
	var r TemplateRecord
	err := scanner.Scan(
		&r.ID,
		&r.UserID,
		&r.Name,
		&r.Description,
		&r.StartupScript,
		&r.OSFamily,
		&r.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

// ListTemplatesByUserID returns all templates owned by userID.
func (db *DB) ListTemplatesByUserID(userID string) ([]TemplateRecord, error) {
	rows, err := db.conn.Query(`
		SELECT `+templateSelectColumns+`
		FROM templates
		WHERE user_id = ?
		ORDER BY created_at`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("list templates: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var records []TemplateRecord
	for rows.Next() {
		r, err := scanTemplateRecord(rows)
		if err != nil {
			return nil, fmt.Errorf("scan template: %w", err)
		}
		records = append(records, *r)
	}
	return records, rows.Err()
}

// ListTemplatesByUserIDAndOSFamily returns templates owned by userID for osFamily.
func (db *DB) ListTemplatesByUserIDAndOSFamily(userID, osFamily string) ([]TemplateRecord, error) {
	rows, err := db.conn.Query(`
		SELECT `+templateSelectColumns+`
		FROM templates
		WHERE user_id = ? AND os_family = ?
		ORDER BY created_at`,
		userID, osFamily,
	)
	if err != nil {
		return nil, fmt.Errorf("list templates by os: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var records []TemplateRecord
	for rows.Next() {
		r, err := scanTemplateRecord(rows)
		if err != nil {
			return nil, fmt.Errorf("scan template: %w", err)
		}
		records = append(records, *r)
	}
	return records, rows.Err()
}

// SearchTemplatesByUserID returns templates owned by userID whose name contains query.
func (db *DB) SearchTemplatesByUserID(userID, query string) ([]TemplateRecord, error) {
	rows, err := db.conn.Query(`
		SELECT `+templateSelectColumns+`
		FROM templates
		WHERE user_id = ? AND INSTR(LOWER(name), LOWER(?)) > 0
		ORDER BY name`,
		userID, query,
	)
	if err != nil {
		return nil, fmt.Errorf("search templates: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var records []TemplateRecord
	for rows.Next() {
		r, err := scanTemplateRecord(rows)
		if err != nil {
			return nil, fmt.Errorf("scan template: %w", err)
		}
		records = append(records, *r)
	}
	return records, rows.Err()
}

// InsertTemplate creates a new template row owned by userID for osFamily.
func (db *DB) InsertTemplate(id, userID, name, startupScript, osFamily string) error {
	_, err := db.conn.Exec(`
		INSERT INTO templates (id, user_id, name, startup_script, os_family)
		VALUES (?, ?, ?, ?, ?)`,
		id, userID, name, nullIfEmpty(startupScript), nullIfEmpty(osFamily),
	)
	if err != nil {
		return fmt.Errorf("insert template: %w", err)
	}
	return nil
}

// GetTemplateByID returns the template row for id, or sql.ErrNoRows if not found.
func (db *DB) GetTemplateByID(id string) (*TemplateRecord, error) {
	row := db.conn.QueryRow(`
		SELECT `+templateSelectColumns+`
		FROM templates
		WHERE id = ?`,
		id,
	)
	return scanTemplateRecord(row)
}

// GetTemplateByNameAndUserID returns a template row for name owned by userID.
// If multiple OS variants share the name, the first match is returned.
func (db *DB) GetTemplateByNameAndUserID(name, userID string) (*TemplateRecord, error) {
	row := db.conn.QueryRow(`
		SELECT `+templateSelectColumns+`
		FROM templates
		WHERE name = ? AND user_id = ?
		ORDER BY created_at
		LIMIT 1`,
		name, userID,
	)
	return scanTemplateRecord(row)
}

// GetTemplateByNameUserIDAndOSFamily returns the template for name+os owned by userID.
func (db *DB) GetTemplateByNameUserIDAndOSFamily(name, userID, osFamily string) (*TemplateRecord, error) {
	row := db.conn.QueryRow(`
		SELECT `+templateSelectColumns+`
		FROM templates
		WHERE name = ? AND user_id = ? AND os_family = ?`,
		name, userID, osFamily,
	)
	return scanTemplateRecord(row)
}

// DeleteTemplateByNameAndUserID removes all template rows with name for userID.
func (db *DB) DeleteTemplateByNameAndUserID(name, userID string) error {
	_, err := db.conn.Exec(`DELETE FROM templates WHERE name = ? AND user_id = ?`, name, userID)
	if err != nil {
		return fmt.Errorf("delete template %s: %w", name, err)
	}
	return nil
}

// ValidateTemplateNameAvailableForRename verifies that newName can be used for an
// existing template within the same os_family. The current template may keep its own name.
func (db *DB) ValidateTemplateNameAvailableForRename(newName, userID, currentName, osFamily string) error {
	newName = strings.TrimSpace(newName)
	if newName == "" {
		return fmt.Errorf("template name is required")
	}

	record, err := db.GetTemplateByNameUserIDAndOSFamily(newName, userID, osFamily)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		return err
	}
	if record.Name == currentName {
		return nil
	}
	return fmt.Errorf("template name already exists: %s", newName)
}

// UpdateTemplateName updates a template's name in the local database for the given OS.
func (db *DB) UpdateTemplateName(oldName, userID, newName, osFamily string) error {
	newName = strings.TrimSpace(newName)
	if newName == "" {
		return fmt.Errorf("template name is required")
	}

	result, err := db.conn.Exec(`
		UPDATE templates
		SET name = ?
		WHERE name = ? AND user_id = ? AND os_family = ?`,
		newName, oldName, userID, osFamily,
	)
	if err != nil {
		var sqliteErr *sqlite.Error
		if errors.As(err, &sqliteErr) && sqliteErr.Code() == sqlite3.SQLITE_CONSTRAINT_UNIQUE {
			return fmt.Errorf("template name already exists: %s", newName)
		}
		return fmt.Errorf("update template name for %s: %w", oldName, err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("update template name for %s: %w", oldName, err)
	}
	if rows == 0 {
		return fmt.Errorf("template not found: %s", oldName)
	}
	return nil
}

// TemplateNameTaken reports whether userID already has a template named name for osFamily.
func (db *DB) TemplateNameTaken(userID, name, osFamily string) (bool, error) {
	var exists int
	err := db.conn.QueryRow(`
		SELECT 1 FROM templates WHERE user_id = ? AND name = ? AND os_family = ? LIMIT 1`,
		userID, name, osFamily,
	).Scan(&exists)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("check template name: %w", err)
	}
	return true, nil
}
