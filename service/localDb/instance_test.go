package localDb

import (
	"path/filepath"
	"strings"
	"testing"

	"outpost-cli/internal/sqliteutil"
)

const (
	testRegion   = "us-east-1"
	testProvider = "aws"
)

func newTestDB(t *testing.T) *DB {
	t.Helper()

	conn, err := sqliteutil.Open(filepath.Join(t.TempDir(), "outpost.db"))
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	db := &DB{conn: conn}
	for _, stmt := range createTables {
		if _, err := conn.Exec(stmt); err != nil {
			t.Fatalf("create schema: %v", err)
		}
	}
	if err := db.ensureLocalUser(); err != nil {
		t.Fatalf("ensure local user: %v", err)
	}
	return db
}

func insertLegacyInstance(t *testing.T, db *DB, id, awsInstanceID, name string) {
	t.Helper()

	_, err := db.conn.Exec(`
		INSERT INTO instances (id, aws_instance_id, name, user_id, status, instance_type)
		VALUES (?, ?, ?, ?, ?, ?)`,
		id, awsInstanceID, name, LocalUserID, "running", "t3.micro",
	)
	if err != nil {
		t.Fatalf("insert legacy instance: %v", err)
	}
}

func TestResolveInstanceByNameOrAwsInstanceID(t *testing.T) {
	db := newTestDB(t)

	if err := db.InsertInstance("box-1", "i-1234567890abcdef0", "alpha", LocalUserID, "running", "t3.micro", testRegion, testProvider); err != nil {
		t.Fatalf("insert instance: %v", err)
	}

	byID, err := db.ResolveInstanceByNameOrAwsInstanceID("i-1234567890abcdef0", LocalUserID)
	if err != nil {
		t.Fatalf("resolve by id: %v", err)
	}
	if byID.Name != "alpha" {
		t.Fatalf("resolve by id returned name %q, want %q", byID.Name, "alpha")
	}

	byName, err := db.ResolveInstanceByNameOrAwsInstanceID("alpha", LocalUserID)
	if err != nil {
		t.Fatalf("resolve by name: %v", err)
	}
	if byName.AwsInstanceID != "i-1234567890abcdef0" {
		t.Fatalf("resolve by name returned id %q, want %q", byName.AwsInstanceID, "i-1234567890abcdef0")
	}
}

func TestResolveInstanceByNameOrAwsInstanceIDPrefersIDOverName(t *testing.T) {
	db := newTestDB(t)

	insertLegacyInstance(t, db, "box-1", "i-1234567890abcdef0", "alpha")
	insertLegacyInstance(t, db, "box-2", "i-abcdef01234567890", "i-1234567890abcdef0")

	got, err := db.ResolveInstanceByNameOrAwsInstanceID("i-1234567890abcdef0", LocalUserID)
	if err != nil {
		t.Fatalf("resolve by id: %v", err)
	}
	if got.ID != "box-1" {
		t.Fatalf("resolve by id returned box %q, want %q", got.ID, "box-1")
	}
}

func TestInsertInstanceRejectsEC2InstanceIDShapedName(t *testing.T) {
	db := newTestDB(t)

	err := db.InsertInstance("box-1", "i-1234567890abcdef0", "i-abcdef01234567890", LocalUserID, "running", "t3.micro", testRegion, testProvider)
	if err == nil {
		t.Fatal("expected EC2 instance ID-shaped name error")
	}
	if !strings.Contains(err.Error(), "box name cannot look like an EC2 instance id") {
		t.Fatalf("unexpected EC2 instance ID-shaped name error: %v", err)
	}
}

func TestInsertInstanceTrimsNameBeforePersisting(t *testing.T) {
	db := newTestDB(t)

	if err := db.InsertInstance("box-1", "i-1234567890abcdef0", " alpha ", LocalUserID, "running", "t3.micro", testRegion, testProvider); err != nil {
		t.Fatalf("insert instance: %v", err)
	}

	got, err := db.GetInstanceByNameAndUserID("alpha", LocalUserID)
	if err != nil {
		t.Fatalf("get trimmed instance name: %v", err)
	}
	if got.Name != "alpha" {
		t.Fatalf("stored name %q, want %q", got.Name, "alpha")
	}
}

func TestInsertInstanceRejectsBlankNameAfterTrim(t *testing.T) {
	db := newTestDB(t)

	err := db.InsertInstance("box-1", "i-1234567890abcdef0", " \t\n ", LocalUserID, "running", "t3.micro", testRegion, testProvider)
	if err == nil {
		t.Fatal("expected blank name error")
	}
	if !strings.Contains(err.Error(), "box name is required") {
		t.Fatalf("unexpected blank name error: %v", err)
	}
}

func TestInsertInstanceDuplicateNameError(t *testing.T) {
	db := newTestDB(t)

	if err := db.InsertInstance("box-1", "i-1234567890abcdef0", "alpha", LocalUserID, "running", "t3.micro", testRegion, testProvider); err != nil {
		t.Fatalf("insert first instance: %v", err)
	}

	err := db.InsertInstance("box-2", "i-abcdef01234567890", "alpha", LocalUserID, "running", "t3.micro", testRegion, testProvider)
	if err == nil {
		t.Fatal("expected duplicate name error")
	}
	if !strings.Contains(err.Error(), "box name already exists: alpha") {
		t.Fatalf("unexpected duplicate name error: %v", err)
	}
}

func TestValidateInstanceNameAvailableRejectsDuplicateName(t *testing.T) {
	db := newTestDB(t)

	if err := db.InsertInstance("box-1", "i-1234567890abcdef0", "alpha", LocalUserID, "running", "t3.micro", testRegion, testProvider); err != nil {
		t.Fatalf("insert instance: %v", err)
	}

	err := db.ValidateInstanceNameAvailable("alpha", LocalUserID)
	if err == nil {
		t.Fatal("expected duplicate name error")
	}
	if !strings.Contains(err.Error(), "box name already exists: alpha") {
		t.Fatalf("unexpected duplicate name error: %v", err)
	}
}

func TestValidateInstanceNameAvailableForRenameAllowsCurrentBoxName(t *testing.T) {
	db := newTestDB(t)

	if err := db.InsertInstance("box-1", "i-1234567890abcdef0", "alpha", LocalUserID, "running", "t3.micro", testRegion, testProvider); err != nil {
		t.Fatalf("insert instance: %v", err)
	}

	if err := db.ValidateInstanceNameAvailableForRename("alpha", LocalUserID, "i-1234567890abcdef0"); err != nil {
		t.Fatalf("validate current name for rename: %v", err)
	}
}

func TestValidateInstanceNameAvailableForRenameRejectsAnotherBoxName(t *testing.T) {
	db := newTestDB(t)

	if err := db.InsertInstance("box-1", "i-1234567890abcdef0", "alpha", LocalUserID, "running", "t3.micro", testRegion, testProvider); err != nil {
		t.Fatalf("insert first instance: %v", err)
	}
	if err := db.InsertInstance("box-2", "i-abcdef01234567890", "beta", LocalUserID, "running", "t3.micro", testRegion, testProvider); err != nil {
		t.Fatalf("insert second instance: %v", err)
	}

	err := db.ValidateInstanceNameAvailableForRename("beta", LocalUserID, "i-1234567890abcdef0")
	if err == nil {
		t.Fatal("expected duplicate name error")
	}
	if !strings.Contains(err.Error(), "box name already exists: beta") {
		t.Fatalf("unexpected duplicate name error: %v", err)
	}
}

func TestUpdateInstanceNamePersistsTrimmedName(t *testing.T) {
	db := newTestDB(t)

	if err := db.InsertInstance("box-1", "i-1234567890abcdef0", "alpha", LocalUserID, "running", "t3.micro", testRegion, testProvider); err != nil {
		t.Fatalf("insert instance: %v", err)
	}

	if err := db.UpdateInstanceName("i-1234567890abcdef0", LocalUserID, " beta "); err != nil {
		t.Fatalf("update instance name: %v", err)
	}

	got, err := db.GetInstanceByAwsInstanceIDAndUserID("i-1234567890abcdef0", LocalUserID)
	if err != nil {
		t.Fatalf("get renamed instance: %v", err)
	}
	if got.Name != "beta" {
		t.Fatalf("stored name %q, want %q", got.Name, "beta")
	}
}
