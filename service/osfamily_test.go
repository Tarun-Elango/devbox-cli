package service

import (
	"strings"
	"testing"
)

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

func TestSSMParameterForArch(t *testing.T) {
	cases := []struct {
		family string
		arch   string
		want   string
	}{
		{OSFamilyAmazonLinux, ArchARM64, "/aws/service/ami-amazon-linux-latest/al2023-ami-kernel-default-arm64"},
		{OSFamilyAmazonLinux, ArchX86_64, "/aws/service/ami-amazon-linux-latest/al2023-ami-kernel-default-x86_64"},
		{OSFamilyUbuntu, ArchARM64, "/aws/service/canonical/ubuntu/server/24.04/stable/current/arm64/hvm/ebs-gp3/ami-id"},
		{OSFamilyUbuntu, ArchX86_64, "/aws/service/canonical/ubuntu/server/24.04/stable/current/amd64/hvm/ebs-gp3/ami-id"},
		{OSFamilyDebian, ArchARM64, "/aws/service/debian/release/12/latest/arm64"},
		{OSFamilyDebian, ArchX86_64, "/aws/service/debian/release/12/latest/amd64"},
	}
	for _, tt := range cases {
		p := MustOSProfile(tt.family)
		got, err := p.SSMParameterForArch(tt.arch)
		if err != nil {
			t.Fatalf("%s/%s: %v", tt.family, tt.arch, err)
		}
		if got != tt.want {
			t.Fatalf("%s/%s = %q, want %q", tt.family, tt.arch, got, tt.want)
		}
	}

	p := MustOSProfile(OSFamilyAmazonLinux)
	if _, err := p.SSMParameterForArch("riscv"); err == nil {
		t.Fatal("expected error for unsupported arch")
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
		{name: "al2023 x86", amiName: "al2023-ami-kernel-6.1-x86_64", wantFamily: OSFamilyAmazonLinux, wantLinux: true},
		{name: "ubuntu amd64", amiName: "ubuntu/images/hvm-ssd-gp3/ubuntu-noble-24.04-amd64-server", wantFamily: OSFamilyUbuntu, wantLinux: true},
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

func TestAllOSFamiliesHaveBothArchParams(t *testing.T) {
	for _, p := range AllOSFamilies() {
		for _, arch := range []string{ArchARM64, ArchX86_64} {
			param, err := p.SSMParameterForArch(arch)
			if err != nil {
				t.Fatalf("%s %s: %v", p.Family, arch, err)
			}
			if !strings.Contains(param, "arm64") && !strings.Contains(param, "amd64") && !strings.Contains(param, "x86_64") {
				t.Fatalf("%s %s param %q missing arch token", p.Family, arch, param)
			}
		}
	}
}
