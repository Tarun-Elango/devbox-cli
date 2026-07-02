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
	CreatedAt     string
}

// ListTemplatesByUserID returns all templates owned by userID.
func (db *DB) ListTemplatesByUserID(userID string) ([]TemplateRecord, error) {
	rows, err := db.conn.Query(`
		SELECT id, user_id, name, description, startup_script, created_at
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
		var r TemplateRecord
		if err := rows.Scan(
			&r.ID,
			&r.UserID,
			&r.Name,
			&r.Description,
			&r.StartupScript,
			&r.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan template: %w", err)
		}
		records = append(records, r)
	}
	return records, rows.Err()
}

// SearchTemplatesByUserID returns templates owned by userID whose name contains query.
func (db *DB) SearchTemplatesByUserID(userID, query string) ([]TemplateRecord, error) {
	rows, err := db.conn.Query(`
		SELECT id, user_id, name, description, startup_script, created_at
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
		var r TemplateRecord
		if err := rows.Scan(
			&r.ID,
			&r.UserID,
			&r.Name,
			&r.Description,
			&r.StartupScript,
			&r.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan template: %w", err)
		}
		records = append(records, r)
	}
	return records, rows.Err()
}

// InsertTemplate creates a new template row owned by userID.
func (db *DB) InsertTemplate(id, userID, name, startupScript string) error {
	_, err := db.conn.Exec(`
		INSERT INTO templates (id, user_id, name, startup_script)
		VALUES (?, ?, ?, ?)`,
		id, userID, name, nullIfEmpty(startupScript),
	)
	if err != nil {
		return fmt.Errorf("insert template: %w", err)
	}
	return nil
}

// GetTemplateByID returns the template row for id, or sql.ErrNoRows if not found.
func (db *DB) GetTemplateByID(id string) (*TemplateRecord, error) {
	var r TemplateRecord
	err := db.conn.QueryRow(`
		SELECT id, user_id, name, description, startup_script, created_at
		FROM templates
		WHERE id = ?`,
		id,
	).Scan(
		&r.ID,
		&r.UserID,
		&r.Name,
		&r.Description,
		&r.StartupScript,
		&r.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

// GetTemplateByNameAndUserID returns the template row for name owned by userID,
// or sql.ErrNoRows if not found.
func (db *DB) GetTemplateByNameAndUserID(name, userID string) (*TemplateRecord, error) {
	var r TemplateRecord
	err := db.conn.QueryRow(`
		SELECT id, user_id, name, description, startup_script, created_at
		FROM templates
		WHERE name = ? AND user_id = ?`,
		name, userID,
	).Scan(
		&r.ID,
		&r.UserID,
		&r.Name,
		&r.Description,
		&r.StartupScript,
		&r.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

// DeleteTemplateByNameAndUserID removes a template row by name for userID.
func (db *DB) DeleteTemplateByNameAndUserID(name, userID string) error {
	_, err := db.conn.Exec(`DELETE FROM templates WHERE name = ? AND user_id = ?`, name, userID)
	if err != nil {
		return fmt.Errorf("delete template %s: %w", name, err)
	}
	return nil
}

// ValidateTemplateNameAvailableForRename verifies that newName can be used for an
// existing template. The current template may keep its own name.
func (db *DB) ValidateTemplateNameAvailableForRename(newName, userID, currentName string) error {
	newName = strings.TrimSpace(newName)
	if newName == "" {
		return fmt.Errorf("template name is required")
	}

	record, err := db.GetTemplateByNameAndUserID(newName, userID)
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

// UpdateTemplateName updates a template's name in the local database.
func (db *DB) UpdateTemplateName(oldName, userID, newName string) error {
	newName = strings.TrimSpace(newName)
	if newName == "" {
		return fmt.Errorf("template name is required")
	}

	result, err := db.conn.Exec(`
		UPDATE templates
		SET name = ?
		WHERE name = ? AND user_id = ?`,
		newName, oldName, userID,
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

// TemplateNameTaken reports whether userID already has a template named name.
func (db *DB) TemplateNameTaken(userID, name string) (bool, error) {
	var exists int
	err := db.conn.QueryRow(`
		SELECT 1 FROM templates WHERE user_id = ? AND name = ? LIMIT 1`,
		userID, name,
	).Scan(&exists)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("check template name: %w", err)
	}
	return true, nil
}
