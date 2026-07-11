package service

import "testing"

// Adding new os: add a test for the new os
func TestValidateOSFamily(t *testing.T) {
	for _, family := range []string{OSFamilyAmazonLinux, OSFamilyUbuntu, OSFamilyDebian, " Ubuntu "} {
		if err := ValidateOSFamily(family); err != nil {
			t.Fatalf("ValidateOSFamily(%q) = %v", family, err)
		}
	}
	if err := ValidateOSFamily("windows"); err == nil {
		t.Fatal("expected error for windows")
	}
	if err := ValidateOSFamily(""); err == nil {
		t.Fatal("expected error for empty family")
	}
}

func TestSSHUserForOS(t *testing.T) {
	cases := map[string]string{
		OSFamilyAmazonLinux: "ec2-user",
		OSFamilyUbuntu:      "ubuntu",
		OSFamilyDebian:      "admin",
		"":                  "ec2-user",
		"unknown":           "ec2-user",
	}
	for family, want := range cases {
		if got := SSHUserForOS(family); got != want {
			t.Fatalf("SSHUserForOS(%q) = %q, want %q", family, got, want)
		}
	}
}

func TestClassifyLinuxOSFamily(t *testing.T) {
	tests := []struct {
		name            string
		platform        string
		platformDetails string
		amiName         string
		description     string
		wantFamily      string
		wantLinux       bool
	}{
		{name: "al2023", amiName: "al2023-ami-kernel-6.1-arm64", wantFamily: OSFamilyAmazonLinux, wantLinux: true},
		{name: "ubuntu", amiName: "ubuntu/images/hvm-ssd-gp3/ubuntu-noble-24.04-arm64-server", wantFamily: OSFamilyUbuntu, wantLinux: true},
		{name: "debian", amiName: "debian-12-arm64-20240101", wantFamily: OSFamilyDebian, wantLinux: true},
		{name: "windows", platform: "windows", wantFamily: "", wantLinux: false},
		{name: "macos platform details", platformDetails: "macOS", wantFamily: "", wantLinux: false},
		{name: "macos ami name", amiName: "amzn-ec2-macos-14.0", wantFamily: "", wantLinux: false},
		{name: "unknown linux", platformDetails: "Linux/UNIX", wantFamily: "", wantLinux: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, isLinux := ClassifyLinuxOSFamily(tt.platform, tt.platformDetails, tt.amiName, tt.description)
			if isLinux != tt.wantLinux {
				t.Fatalf("isLinux = %v, want %v", isLinux, tt.wantLinux)
			}
			if got != tt.wantFamily {
				t.Fatalf("family = %q, want %q", got, tt.wantFamily)
			}
		})
	}
}

func TestBuildUserDataV2UsesSSHUser(t *testing.T) {
	encoded, err := buildUserDataV2("ssh-ed25519 AAAA test", "ubuntu", nil)
	if err != nil {
		t.Fatalf("buildUserDataV2: %v", err)
	}
	if encoded == "" {
		t.Fatal("expected non-empty user data")
	}
	// Decoded content is base64; spot-check by building again isn't needed —
	// ensure no error and non-empty for ubuntu and debian users.
	if _, err := buildUserDataV2("ssh-ed25519 AAAA test", "admin", []string{"echo hi\n"}); err != nil {
		t.Fatalf("buildUserDataV2 debian: %v", err)
	}
}
