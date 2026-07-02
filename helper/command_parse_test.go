package helper

import (
	"reflect"
	"testing"
)

func TestParseSSHCommandArgsMatchesMainUsage(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		want    SSHCommandArgs
		wantErr bool
	}{
		{
			name: "default identity",
			args: []string{"mybox"},
			want: SSHCommandArgs{Identity: "/tmp/default-key", Ref: "mybox"},
		},
		{
			name: "explicit identity",
			args: []string{"-i", "/tmp/custom-key", "box-123"},
			want: SSHCommandArgs{Identity: "/tmp/custom-key", Ref: "box-123"},
		},
		{
			name: "quoted identity",
			args: []string{"-i", `"/tmp/custom key"`, "box-123"},
			want: SSHCommandArgs{Identity: "/tmp/custom key", Ref: "box-123"},
		},
		{
			name: "raw ssh options after separator",
			args: []string{"-i=/tmp/custom-key", "box-123", "--", "-v", "-L", "8080:localhost:8080"},
			want: SSHCommandArgs{
				Identity:   "/tmp/custom-key",
				Ref:        "box-123",
				SSHOptions: []string{"-v", "-L", "8080:localhost:8080"},
			},
		},
		{
			name:    "rejects unexpected trailing args",
			args:    []string{"box-123", "-v"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseSSHCommandArgs(tt.args, "/tmp/default-key")
			if tt.wantErr {
				if err == nil {
					t.Fatalf("ParseSSHCommandArgs() error = nil, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseSSHCommandArgs() error = %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("ParseSSHCommandArgs() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestParseCPCommandArgsMatchesMainUsage(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want CopyCommandArgs
	}{
		{
			name: "upload example",
			args: []string{"./main.go", "mybox:/home/ec2-user/app/"},
			want: CopyCommandArgs{
				Identity: "/tmp/default-key",
				Source:   "./main.go",
				Dest:     "mybox:/home/ec2-user/app/",
			},
		},
		{
			name: "download example",
			args: []string{"-i", "/tmp/custom-key", "mybox:/home/ec2-user/app/main.go", "./"},
			want: CopyCommandArgs{
				Identity: "/tmp/custom-key",
				Source:   "mybox:/home/ec2-user/app/main.go",
				Dest:     "./",
			},
		},
		{
			name: "quoted identity",
			args: []string{"-i", `"/tmp/custom key"`, "mybox:/home/ec2-user/app/main.go", "./"},
			want: CopyCommandArgs{
				Identity: "/tmp/custom key",
				Source:   "mybox:/home/ec2-user/app/main.go",
				Dest:     "./",
			},
		},
		{
			name: "paths with spaces",
			args: []string{"./my file.go", "mybox:/home/ec2-user/my dir/"},
			want: CopyCommandArgs{
				Identity: "/tmp/default-key",
				Source:   "./my file.go",
				Dest:     "mybox:/home/ec2-user/my dir/",
			},
		},
		{
			name: "surrounding quotes stripped",
			args: []string{`"./my file.go"`, `"mybox:/home/ec2-user/my dir/"`},
			want: CopyCommandArgs{
				Identity: "/tmp/default-key",
				Source:   "./my file.go",
				Dest:     "mybox:/home/ec2-user/my dir/",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseCPCommandArgs(tt.args, "/tmp/default-key")
			if err != nil {
				t.Fatalf("ParseCPCommandArgs() error = %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("ParseCPCommandArgs() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestParseSingleBoxRefRejectsWrongArgCount(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "no args", args: nil},
		{name: "extra args", args: []string{"mybox", "extra"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var exitCode int
			oldExit := CommandParseExit
			CommandParseExit = func(code int) { exitCode = code; panic("exit") }
			defer func() { CommandParseExit = oldExit }()

			defer func() {
				if recover() == nil {
					t.Fatal("ParseSingleBoxRef() did not exit")
				}
				if exitCode != 1 {
					t.Fatalf("exit code = %d, want 1", exitCode)
				}
			}()

			ParseSingleBoxRef(tt.args, "usage: devbox status <id|name>")
		})
	}
}

func TestParseSingleBoxRefAcceptsOneArg(t *testing.T) {
	got := ParseSingleBoxRef([]string{"mybox"}, "usage: devbox status <id|name>")
	if got != "mybox" {
		t.Fatalf("ParseSingleBoxRef() = %q, want %q", got, "mybox")
	}
}

func TestParseRenameBoxArgsRejectsWrongArgCount(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "no args", args: nil},
		{name: "one arg", args: []string{"mybox"}},
		{name: "extra args", args: []string{"mybox", "new-name", "extra"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var exitCode int
			oldExit := CommandParseExit
			CommandParseExit = func(code int) { exitCode = code; panic("exit") }
			defer func() { CommandParseExit = oldExit }()

			defer func() {
				if recover() == nil {
					t.Fatal("ParseRenameBoxArgs() did not exit")
				}
				if exitCode != 1 {
					t.Fatalf("exit code = %d, want 1", exitCode)
				}
			}()

			ParseRenameBoxArgs(tt.args, "usage: devbox rename <id|name> <new-name>")
		})
	}
}

func TestParseRenameBoxArgsAcceptsTwoArgs(t *testing.T) {
	ref, newName := ParseRenameBoxArgs([]string{"mybox", " new-name "}, "usage: devbox rename <id|name> <new-name>")
	if ref != "mybox" || newName != "new-name" {
		t.Fatalf("ParseRenameBoxArgs() = (%q, %q), want (%q, %q)", ref, newName, "mybox", "new-name")
	}
}

func TestParseForwardArgsRejectsWrongArgCount(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "no args", args: nil},
		{name: "one arg", args: []string{"mybox"}},
		{name: "extra args", args: []string{"mybox", "8080", "extra"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var exitCode int
			oldExit := CommandParseExit
			CommandParseExit = func(code int) { exitCode = code; panic("exit") }
			defer func() { CommandParseExit = oldExit }()

			defer func() {
				if recover() == nil {
					t.Fatal("ParseForwardArgs() did not exit")
				}
				if exitCode != 1 {
					t.Fatalf("exit code = %d, want 1", exitCode)
				}
			}()

			ParseForwardArgs(tt.args, "usage: devbox forward <id|name> <port>")
		})
	}
}

func TestParseForwardArgsAcceptsTwoArgs(t *testing.T) {
	ref, port := ParseForwardArgs([]string{"mybox", "8080"}, "usage: devbox forward <id|name> <port>")
	if ref != "mybox" || port != "8080" {
		t.Fatalf("ParseForwardArgs() = (%q, %q), want (%q, %q)", ref, port, "mybox", "8080")
	}
}

func TestParseForwardArgsTrimsPort(t *testing.T) {
	ref, port := ParseForwardArgs([]string{"mybox", " 8080 "}, "usage: devbox forward <id|name> <port>")
	if ref != "mybox" || port != "8080" {
		t.Fatalf("ParseForwardArgs() = (%q, %q), want (%q, %q)", ref, port, "mybox", "8080")
	}
}

func TestParseForwardArgsRejectsInvalidPort(t *testing.T) {
	tests := []struct {
		name string
		port string
	}{
		{name: "non-numeric", port: "abc"},
		{name: "out of range high", port: "99999"},
		{name: "zero", port: "0"},
		{name: "negative", port: "-1"},
		{name: "empty", port: ""},
		{name: "whitespace only", port: "   "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var exitCode int
			oldExit := CommandParseExit
			CommandParseExit = func(code int) { exitCode = code; panic("exit") }
			defer func() { CommandParseExit = oldExit }()

			defer func() {
				if recover() == nil {
					t.Fatal("ParseForwardArgs() did not exit")
				}
				if exitCode != 1 {
					t.Fatalf("exit code = %d, want 1", exitCode)
				}
			}()

			ParseForwardArgs([]string{"mybox", tt.port}, "usage: devbox forward <id|name> <port>")
		})
	}
}

func TestParseForwardArgsAcceptsPortBoundaries(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{in: "1", want: "1"},
		{in: "65535", want: "65535"},
		{in: "008080", want: "8080"},
	}

	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			_, port := ParseForwardArgs([]string{"mybox", tt.in}, "usage: devbox forward <id|name> <port>")
			if port != tt.want {
				t.Fatalf("ParseForwardArgs() port = %q, want %q", port, tt.want)
			}
		})
	}
}

func TestParseSnapshotArgsRejectsWrongArgCount(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "no args", args: nil},
		{name: "one arg", args: []string{"mybox"}},
		{name: "extra args", args: []string{"mybox", "snap-name", "extra"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var exitCode int
			oldExit := CommandParseExit
			CommandParseExit = func(code int) { exitCode = code; panic("exit") }
			defer func() { CommandParseExit = oldExit }()

			defer func() {
				if recover() == nil {
					t.Fatal("ParseSnapshotArgs() did not exit")
				}
				if exitCode != 1 {
					t.Fatalf("exit code = %d, want 1", exitCode)
				}
			}()

			ParseSnapshotArgs(tt.args, "usage: devbox snapshot create <id|name> <name>")
		})
	}
}

func TestParseSnapshotArgsAcceptsTwoArgs(t *testing.T) {
	ref, snapshotName := ParseSnapshotArgs([]string{"mybox", " snap-name "}, "usage: devbox snapshot create <id|name> <name>")
	if ref != "mybox" || snapshotName != "snap-name" {
		t.Fatalf("ParseSnapshotArgs() = (%q, %q), want (%q, %q)", ref, snapshotName, "mybox", "snap-name")
	}
}

func TestParseSingleSnapshotAmiIDArgRejectsWrongArgCount(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "no args", args: nil},
		{name: "extra args", args: []string{"ami-12345678", "extra"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var exitCode int
			oldExit := CommandParseExit
			CommandParseExit = func(code int) { exitCode = code; panic("exit") }
			defer func() { CommandParseExit = oldExit }()

			defer func() {
				if recover() == nil {
					t.Fatal("ParseSingleSnapshotAmiIDArg() did not exit")
				}
				if exitCode != 1 {
					t.Fatalf("exit code = %d, want 1", exitCode)
				}
			}()

			ParseSingleSnapshotAmiIDArg(tt.args, "usage: devbox snapshot ls <amiId>")
		})
	}
}

func TestParseSingleSnapshotAmiIDArgAcceptsOneArg(t *testing.T) {
	got := ParseSingleSnapshotAmiIDArg([]string{"ami-12345678"}, "usage: devbox snapshot ls <amiId>")
	if got != "ami-12345678" {
		t.Fatalf("ParseSingleSnapshotAmiIDArg() = %q, want %q", got, "ami-12345678")
	}
}

func TestParseTemplateDeleteArgsRejectsWrongArgCount(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "no args", args: nil},
		{name: "extra args", args: []string{"my-template", "extra"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var exitCode int
			oldExit := CommandParseExit
			CommandParseExit = func(code int) { exitCode = code; panic("exit") }
			defer func() { CommandParseExit = oldExit }()

			defer func() {
				if recover() == nil {
					t.Fatal("ParseTemplateDeleteArgs() did not exit")
				}
				if exitCode != 1 {
					t.Fatalf("exit code = %d, want 1", exitCode)
				}
			}()

			ParseTemplateDeleteArgs(tt.args, "usage: devbox template delete <name>")
		})
	}
}

func TestParseTemplateDeleteArgsAcceptsOneArg(t *testing.T) {
	got := ParseTemplateDeleteArgs([]string{"my-template"}, "usage: devbox template delete <name>")
	if got != "my-template" {
		t.Fatalf("ParseTemplateDeleteArgs() = %q, want %q", got, "my-template")
	}
}

func TestParseTemplateRenameArgsRejectsWrongArgCount(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "no args", args: nil},
		{name: "one arg", args: []string{"my-template"}},
		{name: "extra args", args: []string{"my-template", "new-name", "extra"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var exitCode int
			oldExit := CommandParseExit
			CommandParseExit = func(code int) { exitCode = code; panic("exit") }
			defer func() { CommandParseExit = oldExit }()

			defer func() {
				if recover() == nil {
					t.Fatal("ParseTemplateRenameArgs() did not exit")
				}
				if exitCode != 1 {
					t.Fatalf("exit code = %d, want 1", exitCode)
				}
			}()

			ParseTemplateRenameArgs(tt.args, "usage: devbox template rename <name> <new-name>")
		})
	}
}

func TestParseTemplateRenameArgsAcceptsTwoArgs(t *testing.T) {
	id, newName := ParseTemplateRenameArgs([]string{"my-template", " new-name "}, "usage: devbox template rename <name> <new-name>")
	if id != "my-template" || newName != " new-name " {
		t.Fatalf("ParseTemplateRenameArgs() = (%q, %q), want (%q, %q)", id, newName, "my-template", " new-name ")
	}
}

func TestParseSyncCommandArgsMatchesMainUsage(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want SyncCommandArgs
	}{
		{
			name: "upload example",
			args: []string{"./project", "mybox:/home/ec2-user/project"},
			want: SyncCommandArgs{
				Identity: "/tmp/default-key",
				Source:   "./project",
				Dest:     "mybox:/home/ec2-user/project",
			},
		},
		{
			name: "download example with delete",
			args: []string{"-i", "/tmp/custom-key", "--delete", "mybox:/home/ec2-user/project", "./project"},
			want: SyncCommandArgs{
				Identity:    "/tmp/custom-key",
				DeleteExtra: true,
				Source:      "mybox:/home/ec2-user/project",
				Dest:        "./project",
			},
		},
		{
			name: "quoted identity",
			args: []string{"-i", `"/tmp/custom key"`, "--delete", "mybox:/home/ec2-user/project", "./project"},
			want: SyncCommandArgs{
				Identity:    "/tmp/custom key",
				DeleteExtra: true,
				Source:      "mybox:/home/ec2-user/project",
				Dest:        "./project",
			},
		},
		{
			name: "paths with spaces",
			args: []string{"./project dir", "mybox:/home/ec2-user/project dir"},
			want: SyncCommandArgs{
				Identity: "/tmp/default-key",
				Source:   "./project dir",
				Dest:     "mybox:/home/ec2-user/project dir",
			},
		},
		{
			name: "surrounding quotes stripped",
			args: []string{`"./project dir"`, `"mybox:/home/ec2-user/project dir"`},
			want: SyncCommandArgs{
				Identity: "/tmp/default-key",
				Source:   "./project dir",
				Dest:     "mybox:/home/ec2-user/project dir",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseSyncCommandArgs(tt.args, "/tmp/default-key")
			if err != nil {
				t.Fatalf("ParseSyncCommandArgs() error = %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("ParseSyncCommandArgs() = %#v, want %#v", got, tt.want)
			}
		})
	}
}
