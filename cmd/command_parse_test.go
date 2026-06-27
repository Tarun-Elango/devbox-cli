package cmd

import (
	"reflect"
	"testing"
)

func TestParseSSHCommandArgsMatchesMainUsage(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		want    sshCommandArgs
		wantErr bool
	}{
		{
			name: "default identity",
			args: []string{"mybox"},
			want: sshCommandArgs{Identity: "/tmp/default-key", Ref: "mybox"},
		},
		{
			name: "explicit identity",
			args: []string{"-i", "/tmp/custom-key", "box-123"},
			want: sshCommandArgs{Identity: "/tmp/custom-key", Ref: "box-123"},
		},
		{
			name: "raw ssh args after separator",
			args: []string{"-i=/tmp/custom-key", "box-123", "--", "-v", "-L", "8080:localhost:8080"},
			want: sshCommandArgs{
				Identity: "/tmp/custom-key",
				Ref:      "box-123",
				Extra:    []string{"-v", "-L", "8080:localhost:8080"},
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
			got, err := parseSSHCommandArgs(tt.args, "/tmp/default-key")
			if tt.wantErr {
				if err == nil {
					t.Fatalf("parseSSHCommandArgs() error = nil, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("parseSSHCommandArgs() error = %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("parseSSHCommandArgs() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestParseCPCommandArgsMatchesMainUsage(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want copyCommandArgs
	}{
		{
			name: "upload example",
			args: []string{"./main.go", "mybox:/home/ec2-user/app/"},
			want: copyCommandArgs{
				Identity: "/tmp/default-key",
				Source:   "./main.go",
				Dest:     "mybox:/home/ec2-user/app/",
			},
		},
		{
			name: "download example",
			args: []string{"-i", "/tmp/custom-key", "mybox:/home/ec2-user/app/main.go", "./"},
			want: copyCommandArgs{
				Identity: "/tmp/custom-key",
				Source:   "mybox:/home/ec2-user/app/main.go",
				Dest:     "./",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseCPCommandArgs(tt.args, "/tmp/default-key")
			if err != nil {
				t.Fatalf("parseCPCommandArgs() error = %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("parseCPCommandArgs() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestParseSyncCommandArgsMatchesMainUsage(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want syncCommandArgs
	}{
		{
			name: "upload example",
			args: []string{"./project", "mybox:/home/ec2-user/project"},
			want: syncCommandArgs{
				Identity: "/tmp/default-key",
				Source:   "./project",
				Dest:     "mybox:/home/ec2-user/project",
			},
		},
		{
			name: "download example with delete",
			args: []string{"-i", "/tmp/custom-key", "--delete", "mybox:/home/ec2-user/project", "./project"},
			want: syncCommandArgs{
				Identity:    "/tmp/custom-key",
				DeleteExtra: true,
				Source:      "mybox:/home/ec2-user/project",
				Dest:        "./project",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseSyncCommandArgs(tt.args, "/tmp/default-key")
			if err != nil {
				t.Fatalf("parseSyncCommandArgs() error = %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("parseSyncCommandArgs() = %#v, want %#v", got, tt.want)
			}
		})
	}
}
