package localDb

import (
	"database/sql"
	"fmt"
)

// seedDefaultTemplates offers each built-in template at most once.
// default_template_seeds records offerings so user deletes are not restored on Open().
// When a seeded row still exists, description and startup_script are synced from
// default_templates_data.go.
func (db *DB) seedDefaultTemplates() error {
	if err := db.backfillDefaultTemplateSeeds(); err != nil {
		return err
	}

	for _, tmpl := range defaultTemplates {
		seeded, err := db.defaultTemplateAlreadySeeded(tmpl.ID)
		if err != nil {
			return fmt.Errorf("check seed state for %s: %w", tmpl.Name, err)
		}
		if seeded {
			if err := db.syncDefaultTemplate(tmpl); err != nil {
				return fmt.Errorf("sync template %s: %w", tmpl.Name, err)
			}
			continue
		}

		existing, err := db.GetTemplateByNameUserIDAndOSFamily(tmpl.Name, LocalUserID, tmpl.OSFamily)
		if err != nil && err != sql.ErrNoRows {
			return fmt.Errorf("check existing template %s: %w", tmpl.Name, err)
		}
		if err == nil {
			if existing.ID != tmpl.ID {
				// User-owned template already uses this name; skip offering the built-in.
				if err := db.recordDefaultTemplateSeed(tmpl.ID); err != nil {
					return fmt.Errorf("record seed for %s: %w", tmpl.Name, err)
				}
				continue
			}
			// Built-in row exists but seed metadata is missing.
			if err := db.recordDefaultTemplateSeed(tmpl.ID); err != nil {
				return fmt.Errorf("record seed for %s: %w", tmpl.Name, err)
			}
			if err := db.syncDefaultTemplate(tmpl); err != nil {
				return fmt.Errorf("sync template %s: %w", tmpl.Name, err)
			}
			continue
		}

		tx, err := db.conn.Begin()
		if err != nil {
			return fmt.Errorf("seed template %s: %w", tmpl.Name, err)
		}

		_, err = tx.Exec(`
			INSERT INTO templates (id, user_id, name, description, startup_script, os_family, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?)`,
			tmpl.ID,
			LocalUserID,
			tmpl.Name,
			nullIfEmpty(tmpl.Description),
			nullIfEmpty(tmpl.Script),
			nullIfEmpty(tmpl.OSFamily),
			tmpl.CreatedAt,
		)
		if err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("seed template %s: %w", tmpl.Name, err)
		}

		_, err = tx.Exec(
			`INSERT INTO default_template_seeds (template_id) VALUES (?)`,
			tmpl.ID,
		)
		if err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("record seed for %s: %w", tmpl.Name, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("seed template %s: %w", tmpl.Name, err)
		}
	}
	return nil
}

// syncDefaultTemplate updates description and startup_script when the built-in row
// still exists. User renames and user-deleted templates are left unchanged.
func (db *DB) syncDefaultTemplate(tmpl defaultTemplate) error {
	record, err := db.GetTemplateByID(tmpl.ID)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		return err
	}
	if record.UserID != LocalUserID {
		return nil
	}

	wantDescription := tmpl.Description
	wantScript := tmpl.Script
	if StringValue(record.Description) == wantDescription &&
		StringValue(record.OSFamily) == tmpl.OSFamily &&
		StringValue(record.StartupScript) == wantScript {
		return nil
	}

	_, err = db.conn.Exec(`
		UPDATE templates
		SET description = ?, startup_script = ?, os_family = ?
		WHERE id = ? AND user_id = ?`,
		nullIfEmpty(wantDescription),
		nullIfEmpty(wantScript),
		nullIfEmpty(tmpl.OSFamily),
		tmpl.ID,
		LocalUserID,
	)
	if err != nil {
		return fmt.Errorf("update template content: %w", err)
	}
	return nil
}

func (db *DB) recordDefaultTemplateSeed(templateID string) error {
	_, err := db.conn.Exec(
		`INSERT OR IGNORE INTO default_template_seeds (template_id) VALUES (?)`,
		templateID,
	)
	if err != nil {
		return err
	}
	return nil
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

// backfillDefaultTemplateSeeds upgrades DBs seeded before default_template_seeds existed.
// If any built-in template row remains, treat the whole initial batch as already offered
// (including ones the user deleted before upgrading).
func (db *DB) backfillDefaultTemplateSeeds() error {
	var seedCount int
	if err := db.conn.QueryRow(`SELECT COUNT(*) FROM default_template_seeds`).Scan(&seedCount); err != nil {
		return fmt.Errorf("count default template seeds: %w", err)
	}
	if seedCount > 0 {
		return nil
	}

	var existing int
	if err := db.conn.QueryRow(`
		SELECT COUNT(*) FROM templates WHERE id LIKE '00000000-0000-0000-0001-%'`,
	).Scan(&existing); err != nil {
		return fmt.Errorf("count existing default templates: %w", err)
	}
	if existing == 0 {
		return nil
	}

	for _, tmpl := range defaultTemplates {
		_, err := db.conn.Exec(
			`INSERT OR IGNORE INTO default_template_seeds (template_id) VALUES (?)`,
			tmpl.ID,
		)
		if err != nil {
			return fmt.Errorf("backfill seed for %s: %w", tmpl.Name, err)
		}
	}
	return nil
}
