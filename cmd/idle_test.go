package cmd

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
)

func withIdleExit(t *testing.T, fn func()) (code int, exited bool) {
	t.Helper()

	oldExit := idleStopExit
	idleStopExit = func(c int) {
		code = c
		exited = true
		panic("idleTestExit")
	}
	t.Cleanup(func() { idleStopExit = oldExit })

	defer func() {
		if recover() == "idleTestExit" {
			return
		}
		if r := recover(); r != nil {
			panic(r)
		}
	}()

	fn()
	return code, exited
}

func captureIdleStderr(t *testing.T, fn func()) string {
	t.Helper()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	oldStderr := os.Stderr
	os.Stderr = w
	t.Cleanup(func() {
		os.Stderr = oldStderr
		_ = r.Close()
	})

	done := make(chan string, 1)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		done <- buf.String()
	}()

	fn()
	_ = w.Close()
	return <-done
}

// test idle router rejects extra args
func TestIdleRouterRejectsExtraArgs(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "show extra", args: []string{"show", "mybox", "extra"}},
		{name: "delete extra", args: []string{"delete", "mybox", "extra"}},
		{name: "set extra", args: []string{"set", "mybox", "30", "extra"}},
		{name: "update extra", args: []string{"update", "mybox", "30", "extra"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			stderr := captureIdleStderr(t, func() {
				code, exited := withIdleExit(t, func() { IdleRouter(tt.args) })
				if !exited || code != 1 {
					t.Fatalf("IdleRouter() exit = %v code = %d, want exit 1", exited, code)
				}
			})
			if !strings.Contains(stderr, "usage:") {
				t.Fatalf("stderr = %q, want usage message", stderr)
			}
		})
	}
}

// TestInstallIdleStopSuccess checks that installIdleStop invokes ssh with the
// expected target and remote command, and returns nil when the remote script
// exits 0.
func TestInstallIdleStopSuccess(t *testing.T) {
	var gotName string
	var gotArgs []string

	orig := execCommand
	t.Cleanup(func() { execCommand = orig })
	execCommand = func(name string, args ...string) *exec.Cmd {
		gotName = name
		gotArgs = args
		return fakeCommand(t, 0, "")
	}

	if err := installIdleStop("/usr/bin/ssh", "/tmp/id_ed25519", "ec2-user", "1.2.3.4", 30); err != nil {
		t.Fatalf("installIdleStop() error = %v", err)
	}
	if gotName != "/usr/bin/ssh" {
		t.Fatalf("execCommand name = %q, want ssh binary", gotName)
	}
	if len(gotArgs) < 4 {
		t.Fatalf("execCommand args = %v, too few args", gotArgs)
	}
	target := gotArgs[len(gotArgs)-4]
	if target != "ec2-user@1.2.3.4" {
		t.Fatalf("target = %q, want ec2-user@1.2.3.4", target)
	}
	if gotArgs[len(gotArgs)-3] != "sudo" || gotArgs[len(gotArgs)-2] != "bash" || gotArgs[len(gotArgs)-1] != "-s" {
		t.Fatalf("remote command args = %v, want [... sudo bash -s]", gotArgs)
	}
}

// TestInstallIdleStopFailure checks that installIdleStop surfaces the remote
// stderr when the ssh command exits non-zero.
func TestInstallIdleStopFailure(t *testing.T) {
	orig := execCommand
	t.Cleanup(func() { execCommand = orig })
	execCommand = func(name string, args ...string) *exec.Cmd {
		return fakeCommandStderr(t, 1, "systemctl: command not found\n")
	}

	err := installIdleStop("/usr/bin/ssh", "/tmp/id_ed25519", "ec2-user", "1.2.3.4", 30)
	if err == nil {
		t.Fatal("installIdleStop() expected error, got nil")
	}
	if !strings.Contains(err.Error(), "idle-stop install failed") {
		t.Fatalf("error = %q, want it to mention idle-stop install failed", err)
	}
	if !strings.Contains(err.Error(), "systemctl: command not found") {
		t.Fatalf("error = %q, want it to include remote stderr", err)
	}
}

// TestUninstallIdleStopSuccess checks that uninstallIdleStop invokes ssh
// against the right target and returns nil on success.
func TestUninstallIdleStopSuccess(t *testing.T) {
	var gotArgs []string

	orig := execCommand
	t.Cleanup(func() { execCommand = orig })
	execCommand = func(name string, args ...string) *exec.Cmd {
		gotArgs = args
		return fakeCommand(t, 0, "")
	}

	if err := uninstallIdleStop("/usr/bin/ssh", "/tmp/id_ed25519", "ec2-user", "1.2.3.4"); err != nil {
		t.Fatalf("uninstallIdleStop() error = %v", err)
	}
	target := gotArgs[len(gotArgs)-4]
	if target != "ec2-user@1.2.3.4" {
		t.Fatalf("target = %q, want ec2-user@1.2.3.4", target)
	}
}

// TestUninstallIdleStopFailure checks that uninstallIdleStop surfaces the
// remote stderr when the ssh command exits non-zero.
func TestUninstallIdleStopFailure(t *testing.T) {
	orig := execCommand
	t.Cleanup(func() { execCommand = orig })
	execCommand = func(name string, args ...string) *exec.Cmd {
		return fakeCommandStderr(t, 1, "permission denied\n")
	}

	err := uninstallIdleStop("/usr/bin/ssh", "/tmp/id_ed25519", "ec2-user", "1.2.3.4")
	if err == nil {
		t.Fatal("uninstallIdleStop() expected error, got nil")
	}
	if !strings.Contains(err.Error(), "idle-stop uninstall failed") {
		t.Fatalf("error = %q, want it to mention idle-stop uninstall failed", err)
	}
	if !strings.Contains(err.Error(), "permission denied") {
		t.Fatalf("error = %q, want it to include remote stderr", err)
	}
}

// TestUpdateIdleStopOnHostSuccess checks that updateIdleStopOnHost invokes
// ssh against the right target and returns nil on success.
func TestUpdateIdleStopOnHostSuccess(t *testing.T) {
	var gotArgs []string

	orig := execCommand
	t.Cleanup(func() { execCommand = orig })
	execCommand = func(name string, args ...string) *exec.Cmd {
		gotArgs = args
		return fakeCommand(t, 0, "")
	}

	if err := updateIdleStopOnHost("/usr/bin/ssh", "/tmp/id_ed25519", "ec2-user", "1.2.3.4", 45); err != nil {
		t.Fatalf("updateIdleStopOnHost() error = %v", err)
	}
	target := gotArgs[len(gotArgs)-4]
	if target != "ec2-user@1.2.3.4" {
		t.Fatalf("target = %q, want ec2-user@1.2.3.4", target)
	}
}

// TestUpdateIdleStopOnHostFailure checks that updateIdleStopOnHost surfaces
// the remote stderr when the ssh command exits non-zero.
func TestUpdateIdleStopOnHostFailure(t *testing.T) {
	orig := execCommand
	t.Cleanup(func() { execCommand = orig })
	execCommand = func(name string, args ...string) *exec.Cmd {
		return fakeCommandStderr(t, 1, "disk full\n")
	}

	err := updateIdleStopOnHost("/usr/bin/ssh", "/tmp/id_ed25519", "ec2-user", "1.2.3.4", 45)
	if err == nil {
		t.Fatal("updateIdleStopOnHost() expected error, got nil")
	}
	if !strings.Contains(err.Error(), "idle-stop update failed") {
		t.Fatalf("error = %q, want it to mention idle-stop update failed", err)
	}
	if !strings.Contains(err.Error(), "disk full") {
		t.Fatalf("error = %q, want it to include remote stderr", err)
	}
}

// fakeCommandStderr returns an *exec.Cmd backed by a small shell script that
// writes output to stderr and exits with the given code, for exercising code
// paths that capture a command's stderr (e.g. idle-stop install/uninstall).
func fakeCommandStderr(t *testing.T, code int, output string) *exec.Cmd {
	t.Helper()
	script := t.TempDir() + "/fake-cmd-stderr.sh"
	content := "#!/bin/sh\n"
	content += "cat <<'EOF' >&2\n" + output
	if !strings.HasSuffix(output, "\n") {
		content += "\n"
	}
	content += "EOF\n"
	content += "exit " + strconv.Itoa(code) + "\n"
	if err := os.WriteFile(script, []byte(content), 0o700); err != nil {
		t.Fatalf("write fake stderr cmd: %v", err)
	}
	return exec.Command(script)
}
