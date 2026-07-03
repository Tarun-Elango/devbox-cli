package localDb

import (
	"strings"
	"testing"
)

func TestValidateTemplateNameAvailableForRenameAllowsCurrentName(t *testing.T) {
	db := newTestDB(t)

	if err := db.InsertTemplate("tmpl-1", LocalUserID, "alpha", ""); err != nil {
		t.Fatalf("insert template: %v", err)
	}

	if err := db.ValidateTemplateNameAvailableForRename("alpha", LocalUserID, "alpha"); err != nil {
		t.Fatalf("validate current name for rename: %v", err)
	}
}

func TestValidateTemplateNameAvailableForRenameRejectsAnotherTemplateName(t *testing.T) {
	db := newTestDB(t)

	if err := db.InsertTemplate("tmpl-1", LocalUserID, "alpha", ""); err != nil {
		t.Fatalf("insert first template: %v", err)
	}
	if err := db.InsertTemplate("tmpl-2", LocalUserID, "beta", ""); err != nil {
		t.Fatalf("insert second template: %v", err)
	}

	err := db.ValidateTemplateNameAvailableForRename("beta", LocalUserID, "alpha")
	if err == nil {
		t.Fatal("expected duplicate name error")
	}
	if !strings.Contains(err.Error(), "template name already exists: beta") {
		t.Fatalf("unexpected duplicate name error: %v", err)
	}
}

func TestUpdateTemplateNameRejectsDuplicateName(t *testing.T) {
	db := newTestDB(t)

	if err := db.InsertTemplate("tmpl-1", LocalUserID, "alpha", ""); err != nil {
		t.Fatalf("insert first template: %v", err)
	}
	if err := db.InsertTemplate("tmpl-2", LocalUserID, "beta", ""); err != nil {
		t.Fatalf("insert second template: %v", err)
	}

	err := db.UpdateTemplateName("alpha", LocalUserID, "beta")
	if err == nil {
		t.Fatal("expected duplicate name error")
	}
	if !strings.Contains(err.Error(), "template name already exists: beta") {
		t.Fatalf("unexpected duplicate name error: %v", err)
	}
}

func TestUpdateTemplateNamePersistsTrimmedName(t *testing.T) {
	db := newTestDB(t)

	if err := db.InsertTemplate("tmpl-1", LocalUserID, "alpha", "echo hello"); err != nil {
		t.Fatalf("insert template: %v", err)
	}

	if err := db.UpdateTemplateName("alpha", LocalUserID, " beta "); err != nil {
		t.Fatalf("update template name: %v", err)
	}

	record, err := db.GetTemplateByNameAndUserID("beta", LocalUserID)
	if err != nil {
		t.Fatalf("get renamed template: %v", err)
	}
	if record.Name != "beta" {
		t.Fatalf("expected name beta, got %q", record.Name)
	}
	if record.StartupScript.String != "echo hello" {
		t.Fatalf("expected startup script preserved, got %q", record.StartupScript.String)
	}
}

func TestSearchTemplatesByUserIDMatchesPartialName(t *testing.T) {
	db := newTestDB(t)

	for _, tmpl := range []struct {
		id, name string
	}{
		{"tmpl-1", "java"},
		{"tmpl-2", "java25"},
		{"tmpl-3", "javaOld"},
		{"tmpl-4", "springWithJava"},
		{"tmpl-5", "python"},
	} {
		if err := db.InsertTemplate(tmpl.id, LocalUserID, tmpl.name, ""); err != nil {
			t.Fatalf("insert template %s: %v", tmpl.name, err)
		}
	}

	records, err := db.SearchTemplatesByUserID(LocalUserID, "java")
	if err != nil {
		t.Fatalf("search templates: %v", err)
	}
	if len(records) != 4 {
		t.Fatalf("got %d matches, want 4", len(records))
	}

	got := make([]string, len(records))
	for i, r := range records {
		got[i] = r.Name
	}
	want := []string{"java", "java25", "javaOld", "springWithJava"}
	for i, name := range want {
		if got[i] != name {
			t.Fatalf("match %d = %q, want %q (all: %v)", i, got[i], name, got)
		}
	}
}

func TestSearchTemplatesByUserIDIsCaseInsensitive(t *testing.T) {
	db := newTestDB(t)

	if err := db.InsertTemplate("tmpl-1", LocalUserID, "JavaSDK", ""); err != nil {
		t.Fatalf("insert template: %v", err)
	}

	records, err := db.SearchTemplatesByUserID(LocalUserID, "java")
	if err != nil {
		t.Fatalf("search templates: %v", err)
	}
	if len(records) != 1 || records[0].Name != "JavaSDK" {
		t.Fatalf("got %#v, want [JavaSDK]", records)
	}
}

func TestSeedDefaultTemplates(t *testing.T) {
	db := newTestDB(t)

	if err := db.seedDefaultTemplates(); err != nil {
		t.Fatalf("seed default templates: %v", err)
	}

	records, err := db.ListTemplatesByUserID(LocalUserID)
	if err != nil {
		t.Fatalf("list templates: %v", err)
	}
	if len(records) != len(defaultTemplates) {
		t.Fatalf("got %d templates, want %d", len(records), len(defaultTemplates))
	}

	python, err := db.GetTemplateByNameAndUserID("python3", LocalUserID)
	if err != nil {
		t.Fatalf("get python3 template: %v", err)
	}
	if !strings.Contains(python.StartupScript.String, "dnf install -y python3") {
		t.Fatalf("unexpected python3 script: %q", python.StartupScript.String)
	}
	if !python.Description.Valid || python.Description.String == "" {
		t.Fatal("expected python3 template description")
	}

	// Re-seeding syncs script/description from code but must not duplicate rows.
	if err := db.seedDefaultTemplates(); err != nil {
		t.Fatalf("re-seed default templates: %v", err)
	}
	records, err = db.ListTemplatesByUserID(LocalUserID)
	if err != nil {
		t.Fatalf("list templates after re-seed: %v", err)
	}
	if len(records) != len(defaultTemplates) {
		t.Fatalf("after re-seed got %d templates, want %d", len(records), len(defaultTemplates))
	}

	if err := db.DeleteTemplateByNameAndUserID("python3", LocalUserID); err != nil {
		t.Fatalf("delete python3 template: %v", err)
	}
	if err := db.seedDefaultTemplates(); err != nil {
		t.Fatalf("seed after delete: %v", err)
	}
	if _, err := db.GetTemplateByNameAndUserID("python3", LocalUserID); err == nil {
		t.Fatal("expected deleted python3 template to stay deleted after re-seed")
	}
	records, err = db.ListTemplatesByUserID(LocalUserID)
	if err != nil {
		t.Fatalf("list templates after delete and re-seed: %v", err)
	}
	if len(records) != len(defaultTemplates)-1 {
		t.Fatalf("after delete and re-seed got %d templates, want %d", len(records), len(defaultTemplates)-1)
	}
}

func TestSeedDefaultTemplatesSkipsUserNameCollision(t *testing.T) {
	db := newTestDB(t)

	if err := db.InsertTemplate("user-python3", LocalUserID, "python3", "echo user"); err != nil {
		t.Fatalf("insert user template: %v", err)
	}

	if err := db.seedDefaultTemplates(); err != nil {
		t.Fatalf("seed default templates: %v", err)
	}

	userTmpl, err := db.GetTemplateByNameAndUserID("python3", LocalUserID)
	if err != nil {
		t.Fatalf("get user python3 template: %v", err)
	}
	if userTmpl.ID != "user-python3" {
		t.Fatalf("expected user template preserved, got id %q", userTmpl.ID)
	}
	if userTmpl.StartupScript.String != "echo user" {
		t.Fatalf("expected user script preserved, got %q", userTmpl.StartupScript.String)
	}

	records, err := db.ListTemplatesByUserID(LocalUserID)
	if err != nil {
		t.Fatalf("list templates: %v", err)
	}
	if len(records) != len(defaultTemplates) {
		t.Fatalf("got %d templates, want %d (built-in python3 skipped)", len(records), len(defaultTemplates))
	}

	seeded, err := db.defaultTemplateAlreadySeeded("00000000-0000-0000-0001-000000000001")
	if err != nil {
		t.Fatalf("check python3 seed state: %v", err)
	}
	if !seeded {
		t.Fatal("expected python3 built-in marked seeded after name collision")
	}

	if err := db.seedDefaultTemplates(); err != nil {
		t.Fatalf("re-seed default templates: %v", err)
	}
	userTmpl, err = db.GetTemplateByNameAndUserID("python3", LocalUserID)
	if err != nil {
		t.Fatalf("get user python3 template after re-seed: %v", err)
	}
	if userTmpl.ID != "user-python3" {
		t.Fatalf("expected user template preserved after re-seed, got id %q", userTmpl.ID)
	}
}

func TestSeedDefaultTemplatesPreservesUserRename(t *testing.T) {
	db := newTestDB(t)

	if err := db.seedDefaultTemplates(); err != nil {
		t.Fatalf("seed default templates: %v", err)
	}

	pythonID := "00000000-0000-0000-0001-000000000001"
	if err := db.UpdateTemplateName("python3", LocalUserID, "my-python"); err != nil {
		t.Fatalf("rename python3 template: %v", err)
	}

	if err := db.seedDefaultTemplates(); err != nil {
		t.Fatalf("re-seed default templates: %v", err)
	}

	record, err := db.GetTemplateByID(pythonID)
	if err != nil {
		t.Fatalf("get renamed template: %v", err)
	}
	if record.Name != "my-python" {
		t.Fatalf("expected renamed name preserved, got %q", record.Name)
	}
}

func TestSeedDefaultTemplatesSyncsScriptUpdates(t *testing.T) {
	db := newTestDB(t)

	if err := db.seedDefaultTemplates(); err != nil {
		t.Fatalf("seed default templates: %v", err)
	}

	_, err := db.conn.Exec(`
		UPDATE templates SET startup_script = ? WHERE name = ? AND user_id = ?`,
		"echo stale",
		"python3",
		LocalUserID,
	)
	if err != nil {
		t.Fatalf("stale python3 script: %v", err)
	}

	if err := db.seedDefaultTemplates(); err != nil {
		t.Fatalf("re-seed default templates: %v", err)
	}

	python, err := db.GetTemplateByNameAndUserID("python3", LocalUserID)
	if err != nil {
		t.Fatalf("get python3 template: %v", err)
	}
	if strings.Contains(python.StartupScript.String, "echo stale") {
		t.Fatalf("expected python3 script resynced from defaults, got %q", python.StartupScript.String)
	}
	if !strings.Contains(python.StartupScript.String, "dnf install -y python3") {
		t.Fatalf("unexpected python3 script after sync: %q", python.StartupScript.String)
	}
}
