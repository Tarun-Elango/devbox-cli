package cmd

import (
	"strings"
	"testing"

	"outpost-cli/service"
)

func TestSnapshotsToItemsEmpty(t *testing.T) {
	got := snapshotsToItems(nil)
	if len(got) != 0 {
		t.Fatalf("snapshotsToItems(nil) len = %d, want 0", len(got))
	}

	got = snapshotsToItems([]*service.Snapshot{})
	if len(got) != 0 {
		t.Fatalf("snapshotsToItems([]) len = %d, want 0", len(got))
	}
}

func TestSnapshotsToItemsMapsFields(t *testing.T) {
	snaps := []*service.Snapshot{
		{
			AmiID:    "ami-111",
			Name:     "before-upgrade",
			State:    "available",
			BoxAwsID: "i-abc",
			Region:   "us-east-1",
			Provider: "aws",
			OSFamily: "amazon-linux",
		},
		{
			AmiID:    "ami-222",
			Name:     "backup",
			State:    "pending",
			BoxAwsID: "i-def",
			Region:   "eu-west-1",
			Provider: "aws",
			OSFamily: "ubuntu",
		},
	}

	got := snapshotsToItems(snaps)
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}

	want := []snapshotItem{
		{AmiID: "ami-111", Name: "before-upgrade", State: "available", BoxAwsID: "i-abc", Region: "us-east-1", Provider: "aws", OSFamily: "amazon-linux"},
		{AmiID: "ami-222", Name: "backup", State: "pending", BoxAwsID: "i-def", Region: "eu-west-1", Provider: "aws", OSFamily: "ubuntu"},
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("item[%d] = %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestPrintSnapshotTable(t *testing.T) {
	items := []snapshotItem{
		{
			AmiID:    "ami-111",
			Name:     "before-upgrade",
			State:    "available",
			BoxAwsID: "i-abc",
			Region:   "us-east-1",
			Provider: "aws",
		},
	}

	out := captureStdout(t, func() {
		printSnapshotTableWidth(items, 120)
	})

	if !strings.Contains(out, "AMI ID") || !strings.Contains(out, "NAME") || !strings.Contains(out, "STATE") {
		t.Fatalf("missing header columns: %q", out)
	}
	if !strings.Contains(out, "ami-111") {
		t.Fatalf("missing ami id: %q", out)
	}
	if !strings.Contains(out, "before-upgrade") {
		t.Fatalf("missing name: %q", out)
	}
	if !strings.Contains(out, "available") {
		t.Fatalf("missing state: %q", out)
	}
	if !strings.Contains(out, "us-east-1") {
		t.Fatalf("missing region: %q", out)
	}
	if !strings.Contains(out, "i-abc") {
		t.Fatalf("missing box id: %q", out)
	}
}

func TestPrintSnapshotTableFitsWidth(t *testing.T) {
	items := []snapshotItem{
		{
			AmiID:    "ami-0abcdefghijklmnopqrstuvwxyz",
			Name:     "very-long-snapshot-name-here",
			State:    "available",
			BoxAwsID: "i-0abcdefghijklmnopqrstuvwxyz",
			Region:   "ap-southeast-2",
			Provider: "aws",
			OSFamily: "ubuntu",
		},
	}
	const width = 80
	out := captureStdout(t, func() {
		printSnapshotTableWidth(items, width)
	})
	for i, line := range strings.Split(strings.TrimRight(out, "\n"), "\n") {
		if len(line) > width {
			t.Fatalf("line %d length %d exceeds %d: %q", i, len(line), width, line)
		}
	}
}
