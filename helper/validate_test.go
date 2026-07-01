package helper

import (
	"context"
	"strings"
	"testing"

	"devbox-cli/service"
)

// helper function to create a test runtime
func newTestRuntime(t *testing.T) *service.Runtime {
	t.Helper()
	t.Setenv("HOME", t.TempDir())

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	rt, err := service.Open(ctx, cancel)
	if err != nil {
		t.Fatalf("open runtime: %v", err)
	}
	t.Cleanup(func() { _ = rt.Close() })
	return rt
}

// ParseNameAndFromFlag test cases
// used in boxes when creating a new box
// return name and from snapshot
func TestValidatePort(t *testing.T) {
	tests := []struct {
		name    string
		port    string
		want    string
		wantErr bool
	}{
		{name: "valid", port: "8080", want: "8080"},
		{name: "min", port: "1", want: "1"},
		{name: "max", port: "65535", want: "65535"},
		{name: "trimmed", port: " 3000 ", want: "3000"},
		{name: "leading zeros", port: "008080", want: "8080"},
		{name: "empty", port: "", wantErr: true},
		{name: "whitespace", port: "   ", wantErr: true},
		{name: "non-numeric", port: "abc", wantErr: true},
		{name: "zero", port: "0", wantErr: true},
		{name: "negative", port: "-1", wantErr: true},
		{name: "too high", port: "65536", wantErr: true},
		{name: "far too high", port: "99999", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ValidatePort(tt.port)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("ValidatePort() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("ValidatePort() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseNameAndFromFlag(t *testing.T) {
	tests := []struct {
		name             string
		args             []string
		wantName         string
		wantFromSnapshot string
		wantErr          bool
	}{
		{name: "no name or from", args: []string{}, wantName: "", wantFromSnapshot: "", wantErr: true},
		{name: "name only", args: []string{"mybox"}, wantName: "mybox", wantFromSnapshot: "", wantErr: false},
		{name: "name and from", args: []string{"mybox", "--from", "ami-1234567890"}, wantName: "mybox", wantFromSnapshot: "ami-1234567890"},
		{name: "--from before name", args: []string{"--from", "ami-1234567890", "mybox"}, wantErr: true},
		// --from with nothing after it
		{name: "--from missing value", args: []string{"mybox", "--from"}, wantErr: true},
		// invalid AMI format
		{name: "invalid ami", args: []string{"mybox", "--from", "not-an-ami"}, wantErr: true},
		// AMI too short (needs ami- + 8–17 hex chars)
		{name: "ami too short", args: []string{"mybox", "--from", "ami-123"}, wantErr: true},
		// empty / whitespace-only AMI
		{name: "empty ami", args: []string{"mybox", "--from", ""}, wantErr: true},
		{name: "whitespace ami", args: []string{"mybox", "--from", "   "}, wantErr: true},
		// next token is another flag
		{name: "ami is a flag", args: []string{"mybox", "--from", "--other"}, wantErr: true},
		{name: "unknown flag", args: []string{"mybox", "--foo"}, wantErr: true},
		{name: "typo suggests --from", args: []string{"mybox", "--fro"}, wantErr: true},
		{name: "name with surrounding spaces", args: []string{"  mybox  "}, wantName: "mybox", wantErr: false},
		{name: "ami with surrounding spaces", args: []string{"mybox", "--from", "  ami-1234567890  "}, wantName: "mybox", wantFromSnapshot: "ami-1234567890"},
		{name: "whitespace-only name", args: []string{"   "}, wantErr: true}, // becomes empty → "missing box name"
		{name: "name contains space", args: []string{"my box"}, wantName: "my box", wantErr: false},
		{name: "two positional args", args: []string{"my", "box"}, wantErr: true},
		{name: "uppercase ami", args: []string{"mybox", "--from", "AMI-1234567890"}, wantName: "mybox", wantFromSnapshot: "AMI-1234567890"},
		{name: "ami min length", args: []string{"mybox", "--from", "ami-12345678"}, wantName: "mybox", wantFromSnapshot: "ami-12345678"},
		{name: "ami max length", args: []string{"mybox", "--from", "ami-12345678901234567"}, wantName: "mybox", wantFromSnapshot: "ami-12345678901234567"},
		{name: "ami too long", args: []string{"mybox", "--from", "ami-123456789012345678"}, wantErr: true},
		{name: "ami non-hex", args: []string{"mybox", "--from", "ami-1234567g"}, wantErr: true},
		{name: "whitespace name before from", args: []string{"   ", "--from", "ami-12345678"}, wantErr: true},
		{name: "typo --f suggests from", args: []string{"mybox", "--f"}, wantErr: true},
		{name: "typo --fo suggests from", args: []string{"mybox", "--fo"}, wantErr: true},
		{name: "trailing extra positional", args: []string{"mybox", "--from", "ami-12345678", "extra"}, wantErr: true},
		{name: "duplicate from", args: []string{"mybox", "--from", "ami-12345678", "--from", "ami-87654321"}, wantErr: true},
		{name: "three positional args", args: []string{"mybox", "othername", "third"}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) { // run each test case
			gotName, gotFromSnapshot, err := ParseNameAndFromFlag(tt.args)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if gotName != tt.wantName || gotFromSnapshot != tt.wantFromSnapshot {
				t.Fatalf("gotName = %v, gotFromSnapshot = %v, wantName = %v, wantFromSnapshot = %v", gotName, gotFromSnapshot, tt.wantName, tt.wantFromSnapshot)
			}
		})
	}
}

func TestParseCreateTemplateArgs(t *testing.T) {
	const ami = "ami-12345678"
	tests := []struct {
		name             string
		args             []string
		wantTemplates    []string
		wantName         string
		wantFromSnapshot string
		wantErr          bool
		wantSubstr       string
	}{
		{name: "one template and name", args: []string{"web", "mybox"}, wantTemplates: []string{"web"}, wantName: "mybox"},
		{name: "two templates and name", args: []string{"web", "db", "mybox"}, wantTemplates: []string{"web", "db"}, wantName: "mybox"},
		{name: "with from at end", args: []string{"web", "mybox", "--from", ami}, wantTemplates: []string{"web"}, wantName: "mybox", wantFromSnapshot: ami},
		{name: "from in middle rejected", args: []string{"web", "--from", ami, "mybox"}, wantErr: true, wantSubstr: "unexpected extra arguments"},
		{name: "missing template and name", args: []string{}, wantErr: true, wantSubstr: "at least one template"},
		{name: "only template", args: []string{"web"}, wantErr: true, wantSubstr: "at least one template"},
		{name: "only from", args: []string{"--from", ami}, wantErr: true, wantSubstr: "at least one template"},
		{name: "from missing value", args: []string{"web", "mybox", "--from"}, wantErr: true, wantSubstr: "--from requires a snapshot AMI ID"},
		{name: "extra after from", args: []string{"web", "mybox", "--from", ami, "extra"}, wantErr: true, wantSubstr: "unexpected extra arguments"},
		{name: "duplicate from", args: []string{"web", "mybox", "--from", ami, "--from", ami}, wantErr: true, wantSubstr: "unexpected extra arguments"},
		{name: "unknown flag", args: []string{"web", "mybox", "--foo"}, wantErr: true, wantSubstr: "unknown flag"},
		{name: "flag-like template", args: []string{"web", "--bad"}, wantErr: true, wantSubstr: "unknown flag"},
		{name: "invalid ami", args: []string{"web", "mybox", "--from", "not-an-ami"}, wantErr: true, wantSubstr: "invalid snapshot AMI ID"},
		{name: "empty template", args: []string{"", "mybox"}, wantErr: true, wantSubstr: "template name is required"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotTemplates, gotName, gotFromSnapshot, err := ParseCreateTemplateArgs(tt.args)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.wantSubstr != "" && !strings.Contains(err.Error(), tt.wantSubstr) {
					t.Fatalf("got %q, want substring %q", err.Error(), tt.wantSubstr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if gotName != tt.wantName || gotFromSnapshot != tt.wantFromSnapshot {
				t.Fatalf("got name=%q from=%q, want name=%q from=%q", gotName, gotFromSnapshot, tt.wantName, tt.wantFromSnapshot)
			}
			if len(gotTemplates) != len(tt.wantTemplates) {
				t.Fatalf("got templates %v, want %v", gotTemplates, tt.wantTemplates)
			}
			for i := range tt.wantTemplates {
				if gotTemplates[i] != tt.wantTemplates[i] {
					t.Fatalf("got templates %v, want %v", gotTemplates, tt.wantTemplates)
				}
			}
		})
	}
}

// UnknownCreateFlagError test cases
// it suggests --from for typos and partial flags,
// input is flag and output is error with a substring to assert on
func TestUnknownCreateFlagError(t *testing.T) {
	tests := []struct {
		name       string
		flag       string
		wantSubstr string // substring to assert on err.Error()
	}{
		// suggests --from: prefix of "--from"
		{name: "bare --", flag: "--", wantSubstr: "did you mean --from?"},
		{name: "partial --f", flag: "--f", wantSubstr: "did you mean --from?"},
		{name: "partial --fo", flag: "--fo", wantSubstr: "did you mean --from?"},
		{name: "partial --fro", flag: "--fro", wantSubstr: "did you mean --from?"},
		// suggests --from: starts with --f
		{name: "typo --foo", flag: "--foo", wantSubstr: "did you mean --from?"},
		{name: "typo --foobar", flag: "--foobar", wantSubstr: "did you mean --from?"},
		{name: "typo --fromm", flag: "--fromm", wantSubstr: "did you mean --from?"},
		{name: "starts with --fr", flag: "--fr", wantSubstr: "did you mean --from?"},
		// no suggestion
		{name: "unrelated --bar", flag: "--bar", wantSubstr: "unknown flag \"--bar\""},
		{name: "unrelated --other", flag: "--other", wantSubstr: "unknown flag \"--other\""},
		{name: "unrelated --e", flag: "--e", wantSubstr: "unknown flag \"--e\""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := UnknownCreateFlagError(tt.flag)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantSubstr) {
				t.Fatalf("got %q, want substring %q", err.Error(), tt.wantSubstr)
			}
		})
	}
}

// ValidateSnapshotAmiID test cases
// check if ami id is valid
func TestValidateSnapshotAmiID(t *testing.T) {
	tests := []struct {
		name       string
		amiID      string
		wantErr    bool
		wantSubstr string // substring to assert on err.Error()
	}{
		{name: "valid ami", amiID: "ami-1234567890", wantErr: false},
		{name: "ami min length", amiID: "ami-12345678", wantErr: false},
		{name: "ami max length", amiID: "ami-12345678901234567", wantErr: false},
		{name: "uppercase ami", amiID: "AMI-12345678", wantErr: false},
		{name: "ami with surrounding spaces", amiID: "  ami-12345678  ", wantErr: false},
		{name: "invalid ami", amiID: "not-an-ami", wantErr: true, wantSubstr: "invalid snapshot AMI ID"},
		{name: "empty ami", amiID: "", wantErr: true, wantSubstr: "--from requires a snapshot AMI ID"},
		{name: "whitespace ami", amiID: "   ", wantErr: true, wantSubstr: "--from requires a snapshot AMI ID"},
		{name: "ami too short", amiID: "ami-123", wantErr: true, wantSubstr: "invalid snapshot AMI ID"},
		{name: "ami too long", amiID: "ami-123456789012345678", wantErr: true, wantSubstr: "invalid snapshot AMI ID"},
		{name: "ami non-hex", amiID: "ami-1234567g", wantErr: true, wantSubstr: "invalid snapshot AMI ID"},
		{name: "ami with flag", amiID: "--from", wantErr: true, wantSubstr: "got flag"},
		{name: "ami with flag prefix", amiID: "--from-ami", wantErr: true, wantSubstr: "got flag"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSnapshotAmiID(tt.amiID)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

// ResolveBoxTarget test cases
// takes mode, runtime, and ref
// ref is box id or name, and returns resolved box target object
func TestResolveBoxTarget(t *testing.T) {
	rt := newTestRuntime(t)

	const (
		boxID = "box-test-1"
		awsID = "i-1234567890abcdef0"
		name  = "mytestbox"
	)

	if err := rt.DB().InsertInstance(
		boxID, awsID, name, service.LocalUserID, "running", "t3.micro",
	); err != nil {
		t.Fatalf("insert instance: %v", err)
	}
	t.Cleanup(func() { // like defer but for tests
		if err := rt.DB().DeleteInstanceByAwsInstanceID(awsID); err != nil {
			t.Errorf("cleanup delete: %v", err)
		}
	})

	t.Run("resolve by name", func(t *testing.T) {
		got, err := ResolveBoxTarget(rt, name) // resolve by name
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Input != name || got.ID != awsID || got.Name != name {
			t.Fatalf("got %+v, want Input=%q ID=%q Name=%q", got, name, awsID, name)
		}
	})

	t.Run("resolve by aws instance id", func(t *testing.T) {
		got, err := ResolveBoxTarget(rt, awsID) // resolve by aws instance id
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Input != awsID || got.ID != awsID || got.Name != name {
			t.Fatalf("got %+v, want Input=%q ID=%q Name=%q", got, awsID, awsID, name)
		}
	})

	t.Run("trims ref whitespace", func(t *testing.T) {
		got, err := ResolveBoxTarget(rt, "  "+name+"  ") // trim whitespace
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Input != name || got.Name != name {
			t.Fatalf("got %+v, want trimmed Input and Name %q", got, name)
		}
	})

	t.Run("empty ref", func(t *testing.T) {
		_, err := ResolveBoxTarget(rt, "") // empty ref, should error
		if err == nil || !strings.Contains(err.Error(), "box id or name is required") {
			t.Fatalf("got %v", err)
		}
	})

	t.Run("nil runtime", func(t *testing.T) {
		_, err := ResolveBoxTarget(nil, name) // nil runtime, should error
		if err == nil || !strings.Contains(err.Error(), "runtime is required") {
			t.Fatalf("got %v", err)
		}
	})

	t.Run("not found", func(t *testing.T) {
		_, err := ResolveBoxTarget(rt, "does-not-exist") // not found, should error
		if err == nil || !strings.Contains(err.Error(), "box not found") {
			t.Fatalf("got %v", err)
		}
	})

}
