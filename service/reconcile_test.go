package service

import (
	"database/sql"
	"errors"
	"testing"

	localDb "outpost-cli/service/localDb"
)

func TestReconcileLocalAgainstRemote(t *testing.T) {
	locals := []string{"keep", "orphan", "update"}
	remote := map[string]int{
		"keep":   1,
		"update": 2,
	}

	var deleted []string
	var found []string
	err := reconcileLocalAgainstRemote(locals,
		func(id string) string { return id },
		remote,
		func(id string) error {
			deleted = append(deleted, id)
			return nil
		},
		func(id string, n int) error {
			found = append(found, id)
			if id == "update" && n != 2 {
				return errors.New("unexpected remote value")
			}
			return nil
		},
	)
	if err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if len(deleted) != 1 || deleted[0] != "orphan" {
		t.Fatalf("deleted=%v, want [orphan]", deleted)
	}
	if len(found) != 2 || found[0] != "keep" || found[1] != "update" {
		t.Fatalf("found=%v, want [keep update]", found)
	}
}

func TestRegionForSnapshotRecordRequiresStoredRegion(t *testing.T) {
	_, err := regionForSnapshotRecord(localDb.SnapshotRecord{AmiID: "ami-abc"})
	if err == nil {
		t.Fatal("expected error for empty region")
	}

	got, err := regionForSnapshotRecord(localDb.SnapshotRecord{
		AmiID:  "ami-abc",
		Region: sql.NullString{String: "eu-west-1", Valid: true},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "eu-west-1" {
		t.Fatalf("got %q, want eu-west-1", got)
	}
}
