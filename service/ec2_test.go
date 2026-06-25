package service

import (
	"context"
	"strings"
	"testing"

	localDb "devbox-cli/service/localDb"
)

func TestCreateInstanceRejectsDuplicateNameBeforeAWS(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	db, err := localDb.Open()
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if err := db.InsertInstance("box-1", "i-1234567890abcdef0", "alpha", LocalUserID, "running", DefaultInstanceType); err != nil {
		t.Fatalf("insert existing instance: %v", err)
	}

	rt := &Runtime{ctx: context.Background(), db: db}
	_, err = rt.CreateInstance("alpha", "", "", LocalUserID, DefaultInstanceType, DefaultVolumeSizeGB)
	if err == nil {
		t.Fatal("expected duplicate name error")
	}
	if !strings.Contains(err.Error(), "box name already exists: alpha") {
		t.Fatalf("unexpected error: %v", err)
	}
}
