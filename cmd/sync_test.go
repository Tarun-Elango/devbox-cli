package cmd

import (
	"reflect"
	"testing"
)

func TestSyncRemoteShellWithIdentity(t *testing.T) {
	got := syncRemoteShell("/tmp/key", "22")
	want := "ssh -i /tmp/key -p 22 -o ConnectTimeout=15 -o StrictHostKeyChecking=accept-new"

	if got != want {
		t.Fatalf("syncRemoteShell() = %q, want %q", got, want)
	}
}

func TestBuildRsyncArgsUpload(t *testing.T) {
	transfer, err := parseCPTransfer("./project", "mybox:/home/ec2-user/project")
	if err != nil {
		t.Fatalf("parseCPTransfer() error = %v", err)
	}

	got := buildRsyncArgs("/tmp/key", transfer, "ec2-user", "203.0.113.10", "22", false)
	want := []string{
		"-az",
		"-e", "ssh -i /tmp/key -p 22 -o ConnectTimeout=15 -o StrictHostKeyChecking=accept-new",
		"./project",
		"ec2-user@203.0.113.10:/home/ec2-user/project",
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("buildRsyncArgs() = %#v, want %#v", got, want)
	}
}

func TestBuildRsyncArgsDownload(t *testing.T) {
	transfer, err := parseCPTransfer("mybox:/home/ec2-user/project", "./project")
	if err != nil {
		t.Fatalf("parseCPTransfer() error = %v", err)
	}

	got := buildRsyncArgs("", transfer, "ec2-user", "203.0.113.10", "22", false)
	want := []string{
		"-az",
		"-e", "ssh -p 22 -o ConnectTimeout=15 -o StrictHostKeyChecking=accept-new",
		"ec2-user@203.0.113.10:/home/ec2-user/project",
		"./project",
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("buildRsyncArgs() = %#v, want %#v", got, want)
	}
}

func TestBuildRsyncArgsWithDelete(t *testing.T) {
	transfer, err := parseCPTransfer("./project", "mybox:/home/ec2-user/project")
	if err != nil {
		t.Fatalf("parseCPTransfer() error = %v", err)
	}

	got := buildRsyncArgs("", transfer, "ec2-user", "203.0.113.10", "22", true)
	want := []string{
		"-az",
		"-e", "ssh -p 22 -o ConnectTimeout=15 -o StrictHostKeyChecking=accept-new",
		"--delete",
		"./project",
		"ec2-user@203.0.113.10:/home/ec2-user/project",
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("buildRsyncArgs() = %#v, want %#v", got, want)
	}
}
