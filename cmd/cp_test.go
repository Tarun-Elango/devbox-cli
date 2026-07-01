package cmd

import (
	"reflect"
	"testing"
)

func TestParseCPTransferUpload(t *testing.T) {
	got, err := parseCPTransfer("./main.go", "mybox:/home/ec2-user/app/")
	if err != nil {
		t.Fatalf("parseCPTransfer() error = %v", err)
	}

	if !got.Upload {
		t.Fatalf("parseCPTransfer() Upload = false, want true")
	}
	if got.BoxRef != "mybox" {
		t.Fatalf("parseCPTransfer() BoxRef = %q, want %q", got.BoxRef, "mybox")
	}
	if got.Local != "./main.go" {
		t.Fatalf("parseCPTransfer() Local = %q, want %q", got.Local, "./main.go")
	}
	if got.Remote != "/home/ec2-user/app/" {
		t.Fatalf("parseCPTransfer() Remote = %q, want %q", got.Remote, "/home/ec2-user/app/")
	}
}

func TestParseCPTransferDownload(t *testing.T) {
	got, err := parseCPTransfer("mybox:/home/ec2-user/app/main.go", "./")
	if err != nil {
		t.Fatalf("parseCPTransfer() error = %v", err)
	}

	if got.Upload {
		t.Fatalf("parseCPTransfer() Upload = true, want false")
	}
	if got.BoxRef != "mybox" {
		t.Fatalf("parseCPTransfer() BoxRef = %q, want %q", got.BoxRef, "mybox")
	}
	if got.Remote != "/home/ec2-user/app/main.go" {
		t.Fatalf("parseCPTransfer() Remote = %q, want %q", got.Remote, "/home/ec2-user/app/main.go")
	}
	if got.Local != "./" {
		t.Fatalf("parseCPTransfer() Local = %q, want %q", got.Local, "./")
	}
}

func TestParseCPTransferPathsWithSpaces(t *testing.T) {
	got, err := parseCPTransfer(`"./my file.go"`, `"mybox:/home/ec2-user/my dir/"`)
	if err != nil {
		t.Fatalf("parseCPTransfer() error = %v", err)
	}
	if got.Local != "./my file.go" {
		t.Fatalf("parseCPTransfer() Local = %q, want %q", got.Local, "./my file.go")
	}
	if got.Remote != "/home/ec2-user/my dir/" {
		t.Fatalf("parseCPTransfer() Remote = %q, want %q", got.Remote, "/home/ec2-user/my dir/")
	}
}

func TestParseCPTransferRequiresOneRemotePath(t *testing.T) {
	tests := []struct {
		name   string
		source string
		dest   string
	}{
		{name: "local to local", source: "./main.go", dest: "./copy.go"},
		{name: "remote to remote", source: "box1:/tmp/main.go", dest: "box2:/tmp/main.go"},
		{name: "missing remote box", source: "./main.go", dest: ":/tmp/main.go"},
		{name: "missing remote path", source: "./main.go", dest: "box:"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := parseCPTransfer(tt.source, tt.dest); err == nil {
				t.Fatalf("parseCPTransfer() error = nil, want error")
			}
		})
	}
}

func TestBuildSCPArgsUpload(t *testing.T) {
	transfer, err := parseCPTransfer("./main.go", "mybox:/home/ec2-user/app/")
	if err != nil {
		t.Fatalf("parseCPTransfer() error = %v", err)
	}

	got := buildSCPArgs("/tmp/key", transfer, "ec2-user", "203.0.113.10", "22")
	want := []string{
		"-i", "/tmp/key",
		"-P", "22",
		"-o", "ConnectTimeout=15",
		"-o", "StrictHostKeyChecking=accept-new",
		"./main.go",
		"ec2-user@203.0.113.10:/home/ec2-user/app/",
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("buildSCPArgs() = %#v, want %#v", got, want)
	}
}

func TestBuildSCPArgsDownload(t *testing.T) {
	transfer, err := parseCPTransfer("mybox:/home/ec2-user/app/main.go", "./")
	if err != nil {
		t.Fatalf("parseCPTransfer() error = %v", err)
	}

	got := buildSCPArgs("", transfer, "ec2-user", "203.0.113.10", "22")
	want := []string{
		"-P", "22",
		"-o", "ConnectTimeout=15",
		"-o", "StrictHostKeyChecking=accept-new",
		"ec2-user@203.0.113.10:/home/ec2-user/app/main.go",
		"./",
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("buildSCPArgs() = %#v, want %#v", got, want)
	}
}
