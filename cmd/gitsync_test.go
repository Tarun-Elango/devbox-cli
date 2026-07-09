package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestKeyInSSHAgentParsesLoadedIdentity checks that keyInSSHAgent reports a key
// as loaded when ssh-add -l lists the same fingerprint as ssh-keygen -lf.
// It stubs both commands via execCommand and expects loaded=true with no error.
func TestKeyInSSHAgentParsesLoadedIdentity(t *testing.T) {
	keyPath := "/tmp/id_ed25519"
	fingerprint := "SHA256:abc123"

	orig := execCommand
	t.Cleanup(func() { execCommand = orig })
	execCommand = func(name string, args ...string) *exec.Cmd {
		if len(args) == 2 && args[0] == "-lf" && strings.HasSuffix(name, "ssh-keygen") {
			return fakeCommand(t, 0, fmt.Sprintf("256 %s user@example.com (ED25519)\n", fingerprint))
		}
		if len(args) == 1 && args[0] == "-l" && strings.HasSuffix(name, "ssh-add") {
			return fakeCommand(t, 0, fmt.Sprintf("256 %s user@example.com (ED25519)\n", fingerprint))
		}
		return orig(name, args...)
	}

	loaded, err := keyInSSHAgent(keyPath)
	if err != nil {
		t.Fatalf("keyInSSHAgent: %v", err)
	}
	if !loaded {
		t.Fatal("expected key to be reported as loaded")
	}
}

// TestKeyInSSHAgentEmptyAgent checks that keyInSSHAgent treats an empty agent
// as "not loaded". It stubs ssh-keygen -lf with a fingerprint and ssh-add -l
// with exit code 1, and expects loaded=false with no error.
func TestKeyInSSHAgentEmptyAgent(t *testing.T) {
	orig := execCommand
	t.Cleanup(func() { execCommand = orig })
	execCommand = func(name string, args ...string) *exec.Cmd {
		if len(args) == 2 && args[0] == "-lf" && strings.HasSuffix(name, "ssh-keygen") {
			return fakeCommand(t, 0, "256 SHA256:abc123 user@example.com (ED25519)\n")
		}
		if len(args) == 1 && args[0] == "-l" && strings.HasSuffix(name, "ssh-add") {
			return fakeCommand(t, 1, "The agent has no identities.\n")
		}
		return orig(name, args...)
	}

	loaded, err := keyInSSHAgent("/tmp/id_ed25519")
	if err != nil {
		t.Fatalf("keyInSSHAgent: %v", err)
	}
	if loaded {
		t.Fatal("expected key not to be loaded")
	}
}

// TestSSHAgentAddSuccess checks that sshAgentAdd returns nil when ssh-add
// succeeds. It stubs execCommand for ssh-add <keyPath> with exit 0 and expects
// no error.
func TestSSHAgentAddSuccess(t *testing.T) {
	keyPath := "/tmp/id_ed25519"
	called := false

	orig := execCommand
	t.Cleanup(func() { execCommand = orig })
	execCommand = func(name string, args ...string) *exec.Cmd {
		if len(args) == 1 && args[0] == keyPath && strings.HasSuffix(name, "ssh-add") {
			called = true
			return fakeCommand(t, 0, "")
		}
		return orig(name, args...)
	}

	if err := sshAgentAdd(keyPath); err != nil {
		t.Fatalf("sshAgentAdd: %v", err)
	}
	if !called {
		t.Fatal("expected ssh-add to be invoked")
	}
}

// TestSSHAgentAddFailure checks that sshAgentAdd surfaces a failed ssh-add.
// It stubs execCommand for ssh-add <keyPath> with a non-zero exit and expects
// an error wrapping that failure.
func TestSSHAgentAddFailure(t *testing.T) {
	keyPath := "/tmp/id_ed25519"

	orig := execCommand
	t.Cleanup(func() { execCommand = orig })
	execCommand = func(name string, args ...string) *exec.Cmd {
		if len(args) == 1 && args[0] == keyPath && strings.HasSuffix(name, "ssh-add") {
			return fakeCommand(t, 1, "Identity added failed\n")
		}
		return orig(name, args...)
	}

	err := sshAgentAdd(keyPath)
	if err == nil {
		t.Fatal("expected sshAgentAdd to fail")
	}
	if !strings.Contains(err.Error(), "ssh-add failed") {
		t.Fatalf("error = %q, want it to mention ssh-add failed", err)
	}
}

// TestSSHAgentRemoveSkipsWhenNotLoaded checks that sshAgentRemove is a no-op
// when the key is not in the agent. It stubs key lookup so the key appears
// unloaded and expects nil without invoking ssh-add -d.
func TestSSHAgentRemoveSkipsWhenNotLoaded(t *testing.T) {
	keyPath := "/tmp/id_ed25519"
	removeCalled := false

	orig := execCommand
	t.Cleanup(func() { execCommand = orig })
	execCommand = func(name string, args ...string) *exec.Cmd {
		if len(args) == 2 && args[0] == "-lf" && strings.HasSuffix(name, "ssh-keygen") {
			return fakeCommand(t, 0, "256 SHA256:abc123 user@example.com (ED25519)\n")
		}
		if len(args) == 1 && args[0] == "-l" && strings.HasSuffix(name, "ssh-add") {
			return fakeCommand(t, 1, "The agent has no identities.\n")
		}
		if len(args) == 2 && args[0] == "-d" && args[1] == keyPath && strings.HasSuffix(name, "ssh-add") {
			removeCalled = true
			return fakeCommand(t, 0, "")
		}
		return orig(name, args...)
	}

	if err := sshAgentRemove(keyPath); err != nil {
		t.Fatalf("sshAgentRemove: %v", err)
	}
	if removeCalled {
		t.Fatal("expected ssh-add -d not to be invoked when key is unloaded")
	}
}

// TestSSHAgentRemoveSuccess checks that sshAgentRemove deletes a loaded key.
// It stubs fingerprint lookup and ssh-add -l so the key appears loaded, then
// stubs ssh-add -d with exit 0, and expects nil with -d invoked.
func TestSSHAgentRemoveSuccess(t *testing.T) {
	keyPath := "/tmp/id_ed25519"
	fingerprint := "SHA256:abc123"
	removeCalled := false

	orig := execCommand
	t.Cleanup(func() { execCommand = orig })
	execCommand = func(name string, args ...string) *exec.Cmd {
		if len(args) == 2 && args[0] == "-lf" && strings.HasSuffix(name, "ssh-keygen") {
			return fakeCommand(t, 0, fmt.Sprintf("256 %s user@example.com (ED25519)\n", fingerprint))
		}
		if len(args) == 1 && args[0] == "-l" && strings.HasSuffix(name, "ssh-add") {
			return fakeCommand(t, 0, fmt.Sprintf("256 %s user@example.com (ED25519)\n", fingerprint))
		}
		if len(args) == 2 && args[0] == "-d" && args[1] == keyPath && strings.HasSuffix(name, "ssh-add") {
			removeCalled = true
			return fakeCommand(t, 0, "")
		}
		return orig(name, args...)
	}

	if err := sshAgentRemove(keyPath); err != nil {
		t.Fatalf("sshAgentRemove: %v", err)
	}
	if !removeCalled {
		t.Fatal("expected ssh-add -d to be invoked")
	}
}

// TestSSHAgentRemoveFailure checks that sshAgentRemove surfaces a failed
// ssh-add -d. It stubs the key as loaded and -d with a non-zero exit, and
// expects an error wrapping that failure.
func TestSSHAgentRemoveFailure(t *testing.T) {
	keyPath := "/tmp/id_ed25519"
	fingerprint := "SHA256:abc123"

	orig := execCommand
	t.Cleanup(func() { execCommand = orig })
	execCommand = func(name string, args ...string) *exec.Cmd {
		if len(args) == 2 && args[0] == "-lf" && strings.HasSuffix(name, "ssh-keygen") {
			return fakeCommand(t, 0, fmt.Sprintf("256 %s user@example.com (ED25519)\n", fingerprint))
		}
		if len(args) == 1 && args[0] == "-l" && strings.HasSuffix(name, "ssh-add") {
			return fakeCommand(t, 0, fmt.Sprintf("256 %s user@example.com (ED25519)\n", fingerprint))
		}
		if len(args) == 2 && args[0] == "-d" && args[1] == keyPath && strings.HasSuffix(name, "ssh-add") {
			return fakeCommand(t, 1, "Could not remove identity\n")
		}
		return orig(name, args...)
	}

	err := sshAgentRemove(keyPath)
	if err == nil {
		t.Fatal("expected sshAgentRemove to fail")
	}
	if !strings.Contains(err.Error(), "ssh-add -d failed") {
		t.Fatalf("error = %q, want it to mention ssh-add -d failed", err)
	}
}

func fakeCommand(t *testing.T, code int, output string) *exec.Cmd {
	t.Helper()
	script := filepath.Join(t.TempDir(), "fake-cmd.sh")
	content := "#!/bin/sh\n"
	content += "cat <<'EOF'\n" + output
	if !strings.HasSuffix(output, "\n") {
		content += "\n"
	}
	content += "EOF\n"
	content += fmt.Sprintf("exit %d\n", code)
	if err := os.WriteFile(script, []byte(content), 0700); err != nil {
		t.Fatalf("write fake ssh-add: %v", err)
	}
	return exec.Command(script)
}
