package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func withUninstallStubs(t *testing.T, home, exe string) {
	t.Helper()

	oldHome := userHomeDirFn
	oldExecutable := osExecutableFn

	userHomeDirFn = func() (string, error) { return home, nil }
	osExecutableFn = func() (string, error) { return exe, nil }

	t.Cleanup(func() {
		userHomeDirFn = oldHome
		osExecutableFn = oldExecutable
	})
}

func TestUninstallRejectsExtraArgs(t *testing.T) {
	stderr := captureStderr(t, func() {
		code, exited := withSetupExit(t, func() { Uninstall([]string{"extra"}) })
		if !exited || code != 1 {
			t.Fatalf("exit = %v exited = %v, want exit 1", code, exited)
		}
	})
	if !strings.Contains(stderr, "usage: devbox uninstall") {
		t.Fatalf("stderr = %q, want usage message", stderr)
	}
}

func TestUninstallDeclined(t *testing.T) {
	withSetupStdin(t, "n\n")

	out := captureStdout(t, func() {
		code, exited := withSetupExit(t, func() { Uninstall(nil) })
		if exited {
			t.Fatalf("unexpected exit %d", code)
		}
	})
	if !strings.Contains(out, "Uninstall skipped.") {
		t.Fatalf("stdout = %q, want skipped message", out)
	}
}

func TestCleanShellRCContentRemovesDevboxBlock(t *testing.T) {
	home := "/home/user"
	installDir := filepath.Join(home, ".local", "bin")
	input := strings.Join([]string{
		"export EDITOR=vim",
		"",
		"# devbox",
		`export PATH="/home/user/.local/bin:$PATH"`,
		"alias ll='ls -la'",
	}, "\n")

	got := cleanShellRCContent(input, installDir, home)
	want := strings.Join([]string{
		"export EDITOR=vim",
		"",
		"alias ll='ls -la'",
	}, "\n") + "\n"

	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestCleanShellRCContentRemovesHomePathVariant(t *testing.T) {
	home := "/home/user"
	installDir := filepath.Join(home, ".local", "bin")
	input := strings.Join([]string{
		"# devbox",
		`export PATH="$HOME/.local/bin:$PATH"`,
	}, "\n") + "\n"

	got := cleanShellRCContent(input, installDir, home)
	if got != "" && got != "\n" {
		t.Fatalf("got %q, want empty shell config", got)
	}
}

// TestCleanShellRCContentIgnoresUnmarkedPathLines guards against removing
// PATH lines added by other tools (e.g. pipx, cargo) that happen to
// reference the same directory as devbox's default install dir, but were
// never written by install.sh (i.e. have no preceding "# devbox" marker).
func TestCleanShellRCContentIgnoresUnmarkedPathLines(t *testing.T) {
	home := "/home/user"
	installDir := filepath.Join(home, ".local", "bin")
	input := strings.Join([]string{
		"export EDITOR=vim",
		`export PATH="$HOME/.local/bin:$PATH"`,
		`export PATH="` + installDir + `:$PATH"`,
	}, "\n") + "\n"

	got := cleanShellRCContent(input, installDir, home)
	if got != input {
		t.Fatalf("got %q, want unchanged %q", got, input)
	}
}

func TestUninstallAcceptedRemovesBinaryDataAndPath(t *testing.T) {
	home := t.TempDir()
	installDir := filepath.Join(home, ".local", "bin")
	exe := filepath.Join(installDir, "devbox")
	if err := os.MkdirAll(installDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(exe, []byte("binary"), 0o755); err != nil {
		t.Fatal(err)
	}

	devboxDir := filepath.Join(home, ".devbox")
	if err := os.MkdirAll(devboxDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(devboxDir, "config.json"), []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}

	backupDir := filepath.Join(home, ".devbox-backup")
	if err := os.MkdirAll(backupDir, 0o700); err != nil {
		t.Fatal(err)
	}

	zshrc := filepath.Join(home, ".zshrc")
	rcContent := strings.Join([]string{
		"export EDITOR=vim",
		"",
		"# devbox",
		`export PATH="` + installDir + `:$PATH"`,
	}, "\n") + "\n"
	if err := os.WriteFile(zshrc, []byte(rcContent), 0o644); err != nil {
		t.Fatal(err)
	}

	withUninstallStubs(t, home, exe)
	withSetupStdin(t, "y\n")

	out := captureStdout(t, func() {
		code, exited := withSetupExit(t, func() { Uninstall(nil) })
		if exited {
			t.Fatalf("unexpected exit %d", code)
		}
	})

	if _, err := os.Stat(exe); !os.IsNotExist(err) {
		t.Fatalf("binary still exists: %v", err)
	}
	if _, err := os.Stat(devboxDir); !os.IsNotExist(err) {
		t.Fatalf(".devbox still exists: %v", err)
	}
	if _, err := os.Stat(backupDir); !os.IsNotExist(err) {
		t.Fatalf(".devbox-backup still exists: %v", err)
	}

	updatedRC, err := os.ReadFile(zshrc)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(updatedRC), "# devbox") || strings.Contains(string(updatedRC), installDir) {
		t.Fatalf("shell rc still contains devbox PATH: %q", string(updatedRC))
	}

	if !strings.Contains(out, "Removed ~/.devbox") {
		t.Fatalf("stdout = %q, want removal messages", out)
	}
	if !strings.Contains(out, "Removed "+exe) {
		t.Fatalf("stdout = %q, want binary removal message", out)
	}
}
