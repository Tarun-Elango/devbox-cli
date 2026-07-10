package service

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeTestSSHConfig(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), ".ssh", "config")
	t.Setenv("HOME", filepath.Dir(filepath.Dir(path)))
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatalf("create ssh dir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatalf("write ssh config: %v", err)
	}
	return path
}

func TestUpdateHostInsertsHostNameWithoutOverwritingFollowingBlock(t *testing.T) {
	path := writeTestSSHConfig(t, "Host outpost-alpha\n    User ec2-user\n\nHost outpost-beta\n    HostName 10.0.0.2\n")

	if err := UpdateHost("alpha", "10.0.0.1"); err != nil {
		t.Fatalf("update host: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read ssh config: %v", err)
	}
	got := string(data)
	want := "Host outpost-alpha\n    HostName 10.0.0.1\n    User ec2-user\n\nHost outpost-beta\n    HostName 10.0.0.2\n"
	if got != want {
		t.Fatalf("ssh config = %q, want %q", got, want)
	}
}

func TestRenameHostPreservesOptionsAndOtherHosts(t *testing.T) {
	path := writeTestSSHConfig(t, "Host github.com\n    HostName github.com\n\nHost outpost-alpha alpha-extra\n    HostName 10.0.0.1\n    User ec2-user\n")

	if err := RenameHost("alpha", "beta"); err != nil {
		t.Fatalf("rename host: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read ssh config: %v", err)
	}
	got := string(data)
	if !strings.Contains(got, "Host outpost-beta alpha-extra\n    HostName 10.0.0.1\n    User ec2-user") {
		t.Fatalf("renamed host block not preserved:\n%s", got)
	}
	if strings.Contains(got, "outpost-alpha") {
		t.Fatalf("old host still present:\n%s", got)
	}
}

func TestRenameHostRejectsExistingTarget(t *testing.T) {
	writeTestSSHConfig(t, "Host outpost-alpha\n    HostName 10.0.0.1\n\nHost outpost-beta\n    HostName 10.0.0.2\n")

	err := RenameHost("alpha", "beta")
	if err == nil {
		t.Fatal("expected duplicate host error")
	}
	if !strings.Contains(err.Error(), `host "outpost-beta" already exists`) {
		t.Fatalf("unexpected duplicate host error: %v", err)
	}
}

func TestEnableDisableForwardAgent(t *testing.T) {
	path := writeTestSSHConfig(t, "Host outpost-alpha\n    HostName 10.0.0.1\n    User ec2-user\n\nHost outpost-beta\n    HostName 10.0.0.2\n")

	enabled, err := ForwardAgentEnabled("alpha")
	if err != nil {
		t.Fatalf("read forward agent: %v", err)
	}
	if enabled {
		t.Fatal("expected ForwardAgent to be disabled initially")
	}

	if err := EnableForwardAgent("alpha"); err != nil {
		t.Fatalf("enable forward agent: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read ssh config: %v", err)
	}
	want := "Host outpost-alpha\n    HostName 10.0.0.1\n    User ec2-user\n    ForwardAgent yes\n\nHost outpost-beta\n    HostName 10.0.0.2\n"
	if string(data) != want {
		t.Fatalf("ssh config = %q, want %q", string(data), want)
	}

	enabled, err = ForwardAgentEnabled("alpha")
	if err != nil {
		t.Fatalf("read forward agent: %v", err)
	}
	if !enabled {
		t.Fatal("expected ForwardAgent to be enabled")
	}

	if err := EnableForwardAgent("alpha"); err != nil {
		t.Fatalf("enable forward agent again: %v", err)
	}
	data, err = os.ReadFile(path)
	if err != nil {
		t.Fatalf("read ssh config: %v", err)
	}
	if string(data) != want {
		t.Fatalf("idempotent enable changed config:\n%s", string(data))
	}

	if err := DisableForwardAgent("alpha"); err != nil {
		t.Fatalf("disable forward agent: %v", err)
	}

	data, err = os.ReadFile(path)
	if err != nil {
		t.Fatalf("read ssh config: %v", err)
	}
	want = "Host outpost-alpha\n    HostName 10.0.0.1\n    User ec2-user\n\nHost outpost-beta\n    HostName 10.0.0.2\n"
	if string(data) != want {
		t.Fatalf("ssh config = %q, want %q", string(data), want)
	}

	if err := DisableForwardAgent("alpha"); err != nil {
		t.Fatalf("disable forward agent again: %v", err)
	}
	data, err = os.ReadFile(path)
	if err != nil {
		t.Fatalf("read ssh config: %v", err)
	}
	if string(data) != want {
		t.Fatalf("idempotent disable changed config:\n%s", string(data))
	}
}

func TestEnableForwardAgentReplacesExistingNo(t *testing.T) {
	path := writeTestSSHConfig(t, "Host outpost-alpha\n    HostName 10.0.0.1\n    ForwardAgent no\n")

	if err := EnableForwardAgent("alpha"); err != nil {
		t.Fatalf("enable forward agent: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read ssh config: %v", err)
	}
	want := "Host outpost-alpha\n    HostName 10.0.0.1\n    ForwardAgent yes\n"
	if string(data) != want {
		t.Fatalf("ssh config = %q, want %q", string(data), want)
	}

	enabled, err := ForwardAgentEnabled("alpha")
	if err != nil {
		t.Fatalf("read forward agent: %v", err)
	}
	if !enabled {
		t.Fatal("expected ForwardAgent to be enabled")
	}
}

func TestEnableForwardAgentCleansDuplicateForwardAgentLines(t *testing.T) {
	path := writeTestSSHConfig(t, "Host outpost-alpha\n    HostName 10.0.0.1\n    ForwardAgent no\n    ForwardAgent yes\n")

	if err := EnableForwardAgent("alpha"); err != nil {
		t.Fatalf("enable forward agent: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read ssh config: %v", err)
	}
	want := "Host outpost-alpha\n    HostName 10.0.0.1\n    ForwardAgent yes\n"
	if string(data) != want {
		t.Fatalf("ssh config = %q, want %q", string(data), want)
	}
}

func TestDisableForwardAgentRemovesNo(t *testing.T) {
	path := writeTestSSHConfig(t, "Host outpost-alpha\n    HostName 10.0.0.1\n    ForwardAgent no\n")

	if err := DisableForwardAgent("alpha"); err != nil {
		t.Fatalf("disable forward agent: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read ssh config: %v", err)
	}
	want := "Host outpost-alpha\n    HostName 10.0.0.1\n"
	if string(data) != want {
		t.Fatalf("ssh config = %q, want %q", string(data), want)
	}
}

func TestForwardAgentEnabledRejectsMissingHost(t *testing.T) {
	writeTestSSHConfig(t, "Host outpost-alpha\n    HostName 10.0.0.1\n")

	_, err := ForwardAgentEnabled("missing")
	if err == nil {
		t.Fatal("expected error for missing host")
	}
	if !errors.Is(err, errSSHHostNotFound) {
		t.Fatalf("expected errSSHHostNotFound, got: %v", err)
	}
	if !strings.Contains(err.Error(), `host "outpost-missing" does not exist`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSyncSSHHostIPUpdatesExistingHostAndAddsMissingHost(t *testing.T) {
	path := writeTestSSHConfig(t, "Host outpost-alpha\n    HostName 10.0.0.1\n")

	if err := syncSSHHostIP("alpha", "10.0.0.9"); err != nil {
		t.Fatalf("sync existing host: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read ssh config: %v", err)
	}
	if !strings.Contains(string(data), "HostName 10.0.0.9") {
		t.Fatalf("updated ip not written:\n%s", string(data))
	}

	if err := syncSSHHostIP("beta", "10.0.0.2"); err != nil {
		t.Fatalf("sync missing host: %v", err)
	}
	data, err = os.ReadFile(path)
	if err != nil {
		t.Fatalf("read ssh config: %v", err)
	}
	if !strings.Contains(string(data), "Host outpost-beta\n    HostName 10.0.0.2") {
		t.Fatalf("missing host not added:\n%s", string(data))
	}
}

func TestDeleteHostRejectsInvalidName(t *testing.T) {
	writeTestSSHConfig(t, "Host outpost-alpha\n    HostName 10.0.0.1\n")

	err := DeleteHost("")
	if err == nil {
		t.Fatal("expected error for empty name")
	}
	if !strings.Contains(err.Error(), "host name cannot be empty") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUpdateHostPreservesBlockIndentAndTrailingNewline(t *testing.T) {
	path := writeTestSSHConfig(t, "Host outpost-alpha\n\tUser ec2-user\n\nHost outpost-beta\n    HostName 10.0.0.2\n")

	if err := UpdateHost("alpha", "10.0.0.1"); err != nil {
		t.Fatalf("update host: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read ssh config: %v", err)
	}
	got := string(data)
	want := "Host outpost-alpha\n\tHostName 10.0.0.1\n\tUser ec2-user\n\nHost outpost-beta\n    HostName 10.0.0.2\n"
	if got != want {
		t.Fatalf("ssh config = %q, want %q", got, want)
	}
}

func TestUpdateSSHConfigWithRetryRetriesFailures(t *testing.T) {
	path := writeTestSSHConfig(t, "")
	attempts := 0

	err := updateSSHConfigWithRetry(func(content string) (string, error) {
		attempts++
		if attempts < sshConfigUpdateAttempts {
			return "", fmt.Errorf("temporary failure")
		}
		return "Host outpost-beta\n", nil
	})
	if err != nil {
		t.Fatalf("update ssh config with retry: %v", err)
	}
	if attempts != sshConfigUpdateAttempts {
		t.Fatalf("attempts = %d, want %d", attempts, sshConfigUpdateAttempts)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read ssh config: %v", err)
	}
	if string(data) != "Host outpost-beta\n" {
		t.Fatalf("ssh config = %q, want %q", string(data), "Host outpost-beta\n")
	}
}
