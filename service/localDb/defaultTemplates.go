package localDb

import (
	"database/sql"
	"fmt"
)

// seedDefaultTemplates offers every built-in template once. The seed record is
// intentionally retained after a user deletes a template, so it is not restored
// on a later Open(). Existing built-ins have their content refreshed by ID while
// keeping their user-selected name.
func (db *DB) seedDefaultTemplates() error {
	tx, err := db.conn.Begin()
	if err != nil {
		return fmt.Errorf("begin default template seed: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	for _, tmpl := range defaultTemplates {
		seeded, err := defaultTemplateSeeded(tx, tmpl.ID)
		if err != nil {
			return fmt.Errorf("check seed state for %s: %w", tmpl.Name, err)
		}
		if seeded {
			if err := syncDefaultTemplate(tx, tmpl); err != nil {
				return fmt.Errorf("sync template %s: %w", tmpl.Name, err)
			}
			continue
		}

		var existingID string
		err = tx.QueryRow(
			`SELECT id FROM templates WHERE user_id = ? AND name = ? AND os_family = ?`,
			LocalUserID, tmpl.Name, tmpl.OSFamily,
		).Scan(&existingID)
		switch {
		case err == sql.ErrNoRows:
			if _, err := tx.Exec(`
				INSERT INTO templates (id, user_id, name, description, startup_script, os_family, created_at)
				VALUES (?, ?, ?, ?, ?, ?, ?)`,
				tmpl.ID,
				LocalUserID,
				tmpl.Name,
				nullIfEmpty(tmpl.Description),
				nullIfEmpty(tmpl.Script),
				tmpl.OSFamily,
				tmpl.CreatedAt,
			); err != nil {
				return fmt.Errorf("insert template %s: %w", tmpl.Name, err)
			}
		case err != nil:
			return fmt.Errorf("check existing template %s: %w", tmpl.Name, err)
		case existingID == tmpl.ID:
			if err := syncDefaultTemplate(tx, tmpl); err != nil {
				return fmt.Errorf("sync template %s: %w", tmpl.Name, err)
			}
		}

		// A same-name user template counts as an offer; never replace it later.
		if _, err := tx.Exec(
			`INSERT INTO default_template_seeds (template_id) VALUES (?)`,
			tmpl.ID,
		); err != nil {
			return fmt.Errorf("record seed for %s: %w", tmpl.Name, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit default template seed: %w", err)
	}
	return nil
}

// syncDefaultTemplate updates built-in content without changing the name, so a
// user rename survives future database opens. A missing row was deleted by the
// user and must stay deleted.
func syncDefaultTemplate(tx *sql.Tx, tmpl defaultTemplate) error {
	var userID string
	err := tx.QueryRow(`SELECT user_id FROM templates WHERE id = ?`, tmpl.ID).Scan(&userID)
	if err == sql.ErrNoRows || userID != LocalUserID {
		return nil
	}
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
		UPDATE templates
		SET description = ?, startup_script = ?, os_family = ?
		WHERE id = ? AND user_id = ?`,
		nullIfEmpty(tmpl.Description),
		nullIfEmpty(tmpl.Script),
		tmpl.OSFamily,
		tmpl.ID,
		LocalUserID,
	)
	return err
}

func defaultTemplateSeeded(tx *sql.Tx, templateID string) (bool, error) {
	var exists int
	err := tx.QueryRow(
		`SELECT 1 FROM default_template_seeds WHERE template_id = ? LIMIT 1`,
		templateID,
	).Scan(&exists)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (db *DB) defaultTemplateAlreadySeeded(templateID string) (bool, error) {
	var exists int
	err := db.conn.QueryRow(
		`SELECT 1 FROM default_template_seeds WHERE template_id = ? LIMIT 1`,
		templateID,
	).Scan(&exists)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}
