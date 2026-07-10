package cmd

import (
	"encoding/base64"
	"path/filepath"
	"strings"
	"testing"

	"outpost-cli/service"
)

func TestExpandUserPath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	got, err := expandUserPath("  '~/keys/box.pem'  ")
	if err != nil {
		t.Fatalf("expandUserPath: %v", err)
	}
	want := filepath.Join(home, "keys", "box.pem")
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}

	got, err = expandUserPath("")
	if err != nil || got != "" {
		t.Fatalf("blank: got %q err %v", got, err)
	}

	got, err = expandUserPath("/abs/key.pem")
	if err != nil || got != "/abs/key.pem" {
		t.Fatalf("abs: got %q err %v", got, err)
	}
}

func TestBuildAuthorizeRemoteCommand(t *testing.T) {
	pub := "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAITestkey user@host"
	remote := buildAuthorizeRemoteCommand(pub)
	encoded := base64.StdEncoding.EncodeToString([]byte(pub))
	if !strings.Contains(remote, encoded) {
		t.Fatalf("remote missing encoded key: %s", remote)
	}
	if !strings.Contains(remote, "authorized_keys") {
		t.Fatalf("remote missing authorized_keys: %s", remote)
	}
	if !strings.Contains(remote, "grep -qxF") {
		t.Fatalf("remote missing idempotent grep: %s", remote)
	}
	if !strings.Contains(remote, "sudo -n mkdir -p /var/lib/outpost") ||
		!strings.Contains(remote, outpostReadyPath) ||
		!strings.Contains(remote, outpostReadyMessage) {
		t.Fatalf("remote missing ready marker setup: %s", remote)
	}
}

func TestImportAuthorizeOfferedOnlyWhenRunningWithIP(t *testing.T) {
	cases := []struct {
		c     service.ImportCandidate
		offer bool
	}{
		{service.ImportCandidate{Kind: service.ImportKindBox, State: "running", IPAddress: "1.2.3.4"}, true},
		{service.ImportCandidate{Kind: service.ImportKindBox, State: "stopped", IPAddress: "1.2.3.4"}, false},
		{service.ImportCandidate{Kind: service.ImportKindBox, State: "running", IPAddress: ""}, false},
		{service.ImportCandidate{Kind: service.ImportKindSnapshot, State: "available"}, false},
	}
	for _, tc := range cases {
		offer := tc.c.Kind == service.ImportKindBox &&
			tc.c.IPAddress != "" &&
			strings.EqualFold(tc.c.State, "running")
		if offer != tc.offer {
			t.Fatalf("%+v: offer=%v want %v", tc.c, offer, tc.offer)
		}
	}
}
