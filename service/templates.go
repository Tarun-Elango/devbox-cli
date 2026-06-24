package service

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// Template mirrors Lighthouse TemplateDto (id, name, description, startupScript).
type Template struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Description   string `json:"description"`
	StartupScript string `json:"startupScript"`
}

// ListTemplates returns templates for userID from the local database.
// User-owned templates use name as id, matching Lighthouse TemplateDto behavior.
func (r *Runtime) ListTemplates(userID string) ([]*Template, error) {
	db := r.DB()

	records, err := db.ListTemplatesByUserID(userID)
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, nil
	}

	templates := make([]*Template, 0, len(records))
	// loop through records and append to templates
	for _, rec := range records {
		templates = append(templates, &Template{
			ID:            rec.Name,
			Name:          rec.Name,
			Description:   nullStringValue(rec.Description),
			StartupScript: normalizeStartupScript(nullStringValue(rec.StartupScript)),
		})
	}
	return templates, nil
}

func nullStringValue(n sql.NullString) string {
	if !n.Valid {
		return ""
	}
	return n.String
}

// normalizeStartupScript strips a leading shebang and normalizes line endings.
// Mirrors Lighthouse Ec2Service.normalizeStartupScript.
func normalizeStartupScript(script string) string {
	script = strings.TrimSpace(script)
	if script == "" {
		return ""
	}
	normalized := strings.ReplaceAll(script, "\r\n", "\n")
	normalized = strings.ReplaceAll(normalized, "\r", "\n")
	normalized = strings.TrimSpace(normalized)
	if strings.HasPrefix(normalized, "#!") {
		idx := strings.IndexByte(normalized, '\n')
		if idx < 0 {
			return ""
		}
		normalized = strings.TrimLeft(normalized[idx+1:], " \t")
	}
	return strings.TrimSpace(normalized)
}

// CreateTemplate creates a user-owned startup template in the local database.
// The startup script is stored for later injection into EC2 user data at box launch.
// Mirrors Lighthouse template creation (POST /v1/boxes/templates).
func (r *Runtime) CreateTemplate(name, startupScript, userID string) (*Template, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("template name is required")
	}

	normalizedScript := normalizeStartupScript(startupScript)
	db := r.DB()

	taken, err := db.TemplateNameTaken(userID, name) // check if the template name is already taken
	if err != nil {
		return nil, err
	}
	if taken {
		return nil, fmt.Errorf("template name already exists: %s", name)
	}

	if err := db.InsertTemplate(uuid.New().String(), userID, name, normalizedScript); err != nil {
		return nil, err
	}

	return &Template{
		ID:            name,
		Name:          name,
		StartupScript: normalizedScript,
	}, nil
}

// DeleteTemplate removes a user-owned template from the local database.
// templateID is the template name (user-facing id in local mode).
func (r *Runtime) DeleteTemplate(templateID, userID string) error {
	templateID = strings.TrimSpace(templateID)
	if templateID == "" {
		return fmt.Errorf("template id is required")
	}

	db := r.DB()

	_, err := db.GetTemplateByNameAndUserID(templateID, userID) // check if the template exists
	if err == sql.ErrNoRows {
		return fmt.Errorf("template not found: %s", templateID)
	}
	if err != nil {
		return err
	}

	return db.DeleteTemplateByNameAndUserID(templateID, userID) // delete the template from the local database
}

// CreateBoxFromTemplates creates a new box applying the given templates' startup scripts.
func (r *Runtime) CreateBoxFromTemplates(name string, templateIDs []string, publicKey, fromSnapshot, userID, instanceType string) (*Instance, error) {
	if len(templateIDs) == 0 {
		return nil, fmt.Errorf("template not found")
	}

	db := r.DB()

	startupScripts := make([]string, 0, len(templateIDs))

	// loop through templateIDs and get the startup script for each template
	for _, templateID := range templateIDs {
		ref := strings.TrimSpace(templateID)
		if ref == "" {
			return nil, fmt.Errorf("template not found")
		}

		record, err := db.GetTemplateByNameAndUserID(ref, userID)
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("template not found: %s", ref)
		}
		if err != nil {
			return nil, err
		}

		script := normalizeStartupScript(nullStringValue(record.StartupScript))
		if script != "" {
			startupScripts = append(startupScripts, script)
		}
	}

	return r.createInstanceWithStartupScripts(name, publicKey, fromSnapshot, userID, instanceType, startupScripts)
}
