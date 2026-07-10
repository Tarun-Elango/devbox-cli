package service

import (
	"testing"

	localDb "outpost-cli/service/localDb"
)

func TestUniqueImportName(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	db, err := localDb.Open()
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if err := db.InsertInstance("box-1", "i-11111111111111111", "alpha", LocalUserID, "running", "t3.micro", "us-east-1", "aws"); err != nil {
		t.Fatalf("insert: %v", err)
	}

	names := make(map[string]struct{})
	got, err := uniqueImportName(db, LocalUserID, "alpha", "i-22222222222222222", false, names)
	if err != nil || got != "alpha-2" {
		t.Fatalf("collision: got %q err %v, want alpha-2", got, err)
	}

	got, err = uniqueImportName(db, LocalUserID, "i-33333333333333333", "i-33333333333333333", false, names)
	if err != nil || got != "imported-33333333" {
		t.Fatalf("id-shaped: got %q err %v, want imported-33333333", got, err)
	}
}

func TestUniqueImportNameAvoidsBatchCollisions(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	db, err := localDb.Open()
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	for _, snapshot := range []bool{false, true} {
		t.Run(map[bool]string{false: "instances", true: "snapshots"}[snapshot], func(t *testing.T) {
			names := make(map[string]struct{})
			first, err := uniqueImportName(db, LocalUserID, "duplicate", "i-11111111111111111", snapshot, names)
			if err != nil || first != "duplicate" {
				t.Fatalf("first: got %q err %v, want duplicate", first, err)
			}
			second, err := uniqueImportName(db, LocalUserID, "duplicate", "i-22222222222222222", snapshot, names)
			if err != nil || second != "duplicate-2" {
				t.Fatalf("second: got %q err %v, want duplicate-2", second, err)
			}
		})
	}
}
