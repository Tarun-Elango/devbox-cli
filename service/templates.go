package service

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/google/uuid"

	localDb "outpost-cli/service/localDb"
)

// Template mirrors Lighthouse TemplateDto (id, name, description, startupScript).
type Template struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Description   string `json:"description"`
	StartupScript string `json:"startupScript"`
	OSFamily      string `json:"osFamily"`
}

// create a template from a template record
func templateFromRecord(rec localDb.TemplateRecord) *Template {
	return &Template{
		ID:            rec.Name,
		Name:          rec.Name,
		Description:   localDb.StringValue(rec.Description),
		StartupScript: normalizeStartupScript(localDb.StringValue(rec.StartupScript)),
		OSFamily:      localDb.StringValue(rec.OSFamily),
	}
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
	for _, rec := range records {
		templates = append(templates, templateFromRecord(rec))
	}
	return templates, nil
}

// ListTemplatesForOS returns templates owned by userID for the given OS family.
func (r *Runtime) ListTemplatesForOS(userID, osFamily string) ([]*Template, error) {
	osFamily = NormalizeOSFamily(osFamily)
	if err := ValidateOSFamily(osFamily); err != nil {
		return nil, err
	}
	db := r.DB()
	records, err := db.ListTemplatesByUserIDAndOSFamily(userID, osFamily)
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, nil
	}
	templates := make([]*Template, 0, len(records))
	for _, rec := range records {
		templates = append(templates, templateFromRecord(rec))
	}
	return templates, nil
}

// SearchTemplates returns templates for userID whose name contains query.
func (r *Runtime) SearchTemplates(userID, query string) ([]*Template, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, fmt.Errorf("search query is required")
	}

	db := r.DB()

	records, err := db.SearchTemplatesByUserID(userID, query)
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, nil
	}

	templates := make([]*Template, 0, len(records))
	for _, rec := range records {
		templates = append(templates, templateFromRecord(rec))
	}
	return templates, nil
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
func (r *Runtime) CreateTemplate(name, startupScript, userID, osFamily string) (*Template, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("template name is required")
	}
	osFamily = NormalizeOSFamily(osFamily)
	if osFamily == "" {
		osFamily = DefaultOSFamily
	}
	if err := ValidateOSFamily(osFamily); err != nil {
		return nil, err
	}

	normalizedScript := normalizeStartupScript(startupScript)
	db := r.DB()

	taken, err := db.TemplateNameTaken(userID, name, osFamily)
	if err != nil {
		return nil, err
	}
	if taken {
		return nil, fmt.Errorf("template name already exists: %s", name)
	}

	if err := db.InsertTemplate(uuid.New().String(), userID, name, normalizedScript, osFamily); err != nil {
		return nil, err
	}

	return &Template{
		ID:            name,
		Name:          name,
		StartupScript: normalizedScript,
		OSFamily:      osFamily,
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

	_, err := db.GetTemplateByNameAndUserID(templateID, userID)
	if err == sql.ErrNoRows {
		return fmt.Errorf("template not found: %s", templateID)
	}
	if err != nil {
		return err
	}

	return db.DeleteTemplateByNameAndUserID(templateID, userID)
}

// RenameTemplate updates a user-owned template name in the local database.
func (r *Runtime) RenameTemplate(oldName, newName, userID string) (*Template, error) {
	oldName = strings.TrimSpace(oldName)
	newName = strings.TrimSpace(newName)
	if oldName == "" {
		return nil, fmt.Errorf("template id is required")
	}
	if newName == "" {
		return nil, fmt.Errorf("template name is required")
	}

	db := r.DB()

	record, err := db.GetTemplateByNameAndUserID(oldName, userID)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("template not found: %s", oldName)
	}
	if err != nil {
		return nil, err
	}

	osFamily := localDb.StringValue(record.OSFamily)
	if osFamily == "" {
		osFamily = DefaultOSFamily
	}

	if oldName == newName {
		return templateFromRecord(*record), nil
	}

	if err := db.ValidateTemplateNameAvailableForRename(newName, userID, oldName, osFamily); err != nil {
		return nil, err
	}
	if err := db.UpdateTemplateName(oldName, userID, newName, osFamily); err != nil {
		return nil, err
	}

	updated := templateFromRecord(*record)
	updated.ID = newName
	updated.Name = newName
	return updated, nil
}

// CreateBoxFromTemplates creates a new box applying the given templates' startup scripts.
func (r *Runtime) CreateBoxFromTemplates(name string, templateIDs []string, publicKey, fromSnapshot, userID, instanceType, osFamily string, volumeSizeGB int) (*Instance, error) {
	if len(templateIDs) == 0 {
		return nil, fmt.Errorf("template not found")
	}

	osFamily = NormalizeOSFamily(osFamily)
	if fromSnapshot == "" {
		if osFamily == "" {
			osFamily = DefaultOSFamily
		}
		if err := ValidateOSFamily(osFamily); err != nil {
			return nil, err
		}
	}

	db := r.DB()

	if fromSnapshot != "" {
		record, err := db.GetSnapshotByAmiIDAndUserID(fromSnapshot, userID)
		if err == nil {
			if snapOS := localDb.StringValue(record.OSFamily); snapOS != "" {
				osFamily = NormalizeOSFamily(snapOS)
			}
		}
	}

	startupScripts := make([]string, 0, len(templateIDs))

	for _, templateID := range templateIDs {
		ref := strings.TrimSpace(templateID)
		if ref == "" {
			return nil, fmt.Errorf("template not found")
		}

		record, err := db.GetTemplateByNameUserIDAndOSFamily(ref, userID, osFamily)
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("template not found for os %s: %s", osFamily, ref)
		}
		if err != nil {
			return nil, err
		}

		script := normalizeStartupScript(localDb.StringValue(record.StartupScript))
		if script != "" {
			startupScripts = append(startupScripts, script)
		}
	}

	return r.createInstanceWithStartupScripts(name, publicKey, fromSnapshot, userID, instanceType, osFamily, volumeSizeGB, startupScripts)
}
