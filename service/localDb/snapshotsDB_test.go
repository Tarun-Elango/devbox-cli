package localDb

import (
	"database/sql"
	"strings"
	"testing"
)

func openTestDB(t *testing.T) *DB {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	db, err := Open()
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func TestResolveSnapshotByAmiIDOrName(t *testing.T) {
	db := openTestDB(t)

	const (
		userID = LocalUserID
		amiID  = "ami-1234567890abcdef0"
		name   = "before-upgrade"
	)

	if err := db.InsertSnapshot("snap-1", amiID, name, userID, "", "pending"); err != nil {
		t.Fatalf("insert snapshot: %v", err)
	}

	t.Run("resolve by ami id", func(t *testing.T) {
		got, err := db.ResolveSnapshotByAmiIDOrName(amiID, userID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.AmiID != amiID || got.Name != name {
			t.Fatalf("got %+v, want AmiID=%q Name=%q", got, amiID, name)
		}
	})

	t.Run("resolve by uppercase ami id", func(t *testing.T) {
		got, err := db.ResolveSnapshotByAmiIDOrName(strings.ToUpper(amiID), userID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.AmiID != amiID {
			t.Fatalf("got AmiID=%q, want %q", got.AmiID, amiID)
		}
	})

	t.Run("resolve by name", func(t *testing.T) {
		got, err := db.ResolveSnapshotByAmiIDOrName(name, userID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.AmiID != amiID || got.Name != name {
			t.Fatalf("got %+v, want AmiID=%q Name=%q", got, amiID, name)
		}
	})

	t.Run("not found", func(t *testing.T) {
		_, err := db.ResolveSnapshotByAmiIDOrName("missing", userID)
		if err == nil || !strings.Contains(err.Error(), "snapshot not found") {
			t.Fatalf("got %v", err)
		}
	})
}

func TestValidateSnapshotNameAvailable(t *testing.T) {
	db := openTestDB(t)

	if err := db.ValidateSnapshotNameAvailable("before-upgrade", LocalUserID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := db.InsertSnapshot("snap-1", "ami-1234567890abcdef0", "before-upgrade", LocalUserID, "", "pending"); err != nil {
		t.Fatalf("insert snapshot: %v", err)
	}

	err := db.ValidateSnapshotNameAvailable("before-upgrade", LocalUserID)
	if err == nil || !strings.Contains(err.Error(), "snapshot name already exists") {
		t.Fatalf("got %v", err)
	}

	err = db.ValidateSnapshotNameAvailable("ami-1234567890abcdef0", LocalUserID)
	if err == nil || !strings.Contains(err.Error(), "cannot look like an AMI id") {
		t.Fatalf("got %v", err)
	}
}

func TestGetSnapshotByNameAndUserID(t *testing.T) {
	db := openTestDB(t)

	if err := db.InsertSnapshot("snap-1", "ami-1234567890abcdef0", "before-upgrade", LocalUserID, "", "pending"); err != nil {
		t.Fatalf("insert snapshot: %v", err)
	}

	got, err := db.GetSnapshotByNameAndUserID("before-upgrade", LocalUserID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.AmiID != "ami-1234567890abcdef0" {
		t.Fatalf("got AmiID=%q", got.AmiID)
	}

	_, err = db.GetSnapshotByNameAndUserID("missing", LocalUserID)
	if err != sql.ErrNoRows {
		t.Fatalf("got %v, want sql.ErrNoRows", err)
	}
}
