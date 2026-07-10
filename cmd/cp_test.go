package cmd

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"outpost-cli/service"
)

// TestParseCPTransferUpload locks local → remote copy direction.
// It parses a local source and remote dest, and expects Upload=true with
// BoxRef/Local/Remote filled from the dest box and both paths.
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

// TestParseCPTransferDownload locks remote → local copy direction.
// It parses a remote source and local dest, and expects Upload=false with
// BoxRef/Remote/Local filled from the source box and both paths.
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

// TestParseCPTransferPathsWithSpaces locks quote stripping on spaced paths.
// It feeds quoted local and remote args containing spaces, and expects the
// surrounding quotes removed while the spaces inside the paths remain.
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

// TestParseCPTransferRequiresOneRemotePath locks invalid transfer shapes.
// It tries local↔local, remote↔remote, and malformed remote paths, and expects
// each case to return an error instead of a transfer.
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

// TestParseCPLocation locks the single-path parser used by parseCPTransfer.
// It feeds raw location strings (local paths, remote box:path forms, and bad inputs)
// and expects either a filled cpLocation or a clear validation error.
func TestParseCPLocation(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		want    cpLocation
		wantErr string
	}{
		{
			name: "local path",
			raw:  "./main.go",
			want: cpLocation{Raw: "./main.go", Path: "./main.go"},
		},
		{
			name: "local path with surrounding quotes and spaces",
			raw:  `  "./my file.go"  `,
			want: cpLocation{Raw: "./my file.go", Path: "./my file.go"},
		},
		{
			name: "remote box and path",
			raw:  "mybox:/home/ec2-user/app/",
			want: cpLocation{
				Raw:    "mybox:/home/ec2-user/app/",
				Remote: true,
				Box:    "mybox",
				Path:   "/home/ec2-user/app/",
			},
		},
		{
			name: "remote path keeps characters after first colon",
			raw:  "mybox:/tmp/file:with:colons",
			want: cpLocation{
				Raw:    "mybox:/tmp/file:with:colons",
				Remote: true,
				Box:    "mybox",
				Path:   "/tmp/file:with:colons",
			},
		},
		{
			name: "trims box whitespace but preserves path whitespace",
			raw:  "  mybox  : /tmp/spaced ",
			want: cpLocation{
				Raw:    "mybox  : /tmp/spaced",
				Remote: true,
				Box:    "mybox",
				Path:   " /tmp/spaced",
			},
		},
		{
			name:    "empty input",
			raw:     "   ",
			wantErr: "path is required",
		},
		{
			name:    "missing box name",
			raw:     ":/tmp/main.go",
			wantErr: `remote path ":/tmp/main.go" is missing a box name or id`,
		},
		{
			name:    "missing remote path",
			raw:     "mybox:",
			wantErr: `remote path "mybox:" is missing a path`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseCPLocation(tt.raw)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("parseCPLocation() error = nil, want %q", tt.wantErr)
				}
				if err.Error() != tt.wantErr {
					t.Fatalf("parseCPLocation() error = %q, want %q", err.Error(), tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseCPLocation() error = %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("parseCPLocation() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

// TestCPSCPBaseArgs locks the shared scp option prefix.
// It builds argv for a given identity/port and expects ConnectTimeout plus
// StrictHostKeyChecking, with -i only when an identity path is provided.
func TestCPSCPBaseArgs(t *testing.T) {
	t.Run("with identity", func(t *testing.T) {
		got := cpSCPBaseArgs("/tmp/key", "22")
		want := []string{
			"-i", "/tmp/key",
			"-P", "22",
			"-o", "ConnectTimeout=15",
			"-o", "StrictHostKeyChecking=accept-new",
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("cpSCPBaseArgs() = %#v, want %#v", got, want)
		}
	})

	t.Run("without identity", func(t *testing.T) {
		got := cpSCPBaseArgs("", "2222")
		want := []string{
			"-P", "2222",
			"-o", "ConnectTimeout=15",
			"-o", "StrictHostKeyChecking=accept-new",
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("cpSCPBaseArgs() = %#v, want %#v", got, want)
		}
	})
}

// TestBuildSCPArgsUpload locks the scp argv for a local → remote copy.
// It builds args with an identity key for an upload transfer, and expects
// -i/-P/options first, then local path, then user@host:remote.
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

// TestBuildSCPArgsDownload locks the scp argv for a remote → local copy.
// It builds args with no identity for a download transfer, and expects
// -P/options first, then user@host:remote, then the local destination path.
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

// TestCPDefaultKeyPath locks the default identity lookup used when -i is omitted.
// It points HOME at a temp dir, optionally creates ~/.ssh/id_ed25519, and expects
// either that private key path or an empty string when the key is missing.
func TestCPDefaultKeyPath(t *testing.T) {
	t.Run("returns ed25519 key when present", func(t *testing.T) {
		home := t.TempDir()
		t.Setenv("HOME", home)

		sshDir := filepath.Join(home, ".ssh")
		if err := os.MkdirAll(sshDir, 0o700); err != nil {
			t.Fatalf("mkdir .ssh: %v", err)
		}
		keyPath := filepath.Join(sshDir, "id_ed25519")
		if err := os.WriteFile(keyPath, []byte("test-key"), 0o600); err != nil {
			t.Fatalf("write key: %v", err)
		}

		got := cpDefaultKeyPath()
		if got != keyPath {
			t.Fatalf("cpDefaultKeyPath() = %q, want %q", got, keyPath)
		}
	})

	t.Run("returns empty when key is missing", func(t *testing.T) {
		home := t.TempDir()
		t.Setenv("HOME", home)

		got := cpDefaultKeyPath()
		if got != "" {
			t.Fatalf("cpDefaultKeyPath() = %q, want empty", got)
		}
	})
}

// TestCPStatusFromInstance locks the instance→Box mapping used by cpStatusForBox.
func TestCPStatusFromInstance(t *testing.T) {
	t.Run("with instance", func(t *testing.T) {
		got := cpStatusFromInstance(&service.Instance{
			ID:               "i-abc",
			Name:             "mybox",
			Status:           "running",
			InstanceType:     "t3.micro",
			IPAddress:        "203.0.113.10",
			PrivateIPAddress: "10.0.0.5",
			Region:           "us-east-1",
			Provider:         "aws",
		})

		want := &cpStatusResponse{
			Instance: &Box{
				ID:           "i-abc",
				Name:         "mybox",
				Status:       "running",
				InstanceType: "t3.micro",
				PublicIP:     "203.0.113.10",
				PrivateIP:    "10.0.0.5",
				Region:       "us-east-1",
				Provider:     "aws",
			},
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("cpStatusFromInstance() = %#v, want %#v", got, want)
		}
	})

	t.Run("nil instance", func(t *testing.T) {
		got := cpStatusFromInstance(nil)
		want := &cpStatusResponse{}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("cpStatusFromInstance() = %#v, want %#v", got, want)
		}
	})
}
