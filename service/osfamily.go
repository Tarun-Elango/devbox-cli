package service

import (
	"fmt"
	"strings"
)

// Adding new os: add a const + entry in osProfiles, and ValidateOSFamily error message

// Supported Linux OS families for Outpost boxes.
const (
	OSFamilyAmazonLinux = "amazon-linux"
	OSFamilyUbuntu      = "ubuntu"
	OSFamilyDebian      = "debian"

	DefaultOSFamily = OSFamilyAmazonLinux
)

// OSProfile describes Linux-specific launch and SSH behavior for a box.
type OSProfile struct {
	Family      string
	DisplayName string
	SSHUser     string
	HomeDir     string
	// SSMParameterArm64 is the AWS public SSM parameter for the latest ARM64 AMI.
	SSMParameterArm64 string
	// SSMParameterX86_64 is the AWS public SSM parameter for the latest x86_64 AMI.
	SSMParameterX86_64 string
}

var osProfiles = []OSProfile{
	{
		Family:             OSFamilyAmazonLinux,
		DisplayName:        "Amazon Linux 2023",
		SSHUser:            "ec2-user",
		HomeDir:            "/home/ec2-user",
		SSMParameterArm64:  "/aws/service/ami-amazon-linux-latest/al2023-ami-kernel-default-arm64",
		SSMParameterX86_64: "/aws/service/ami-amazon-linux-latest/al2023-ami-kernel-default-x86_64",
	},
	{
		Family:             OSFamilyUbuntu,
		DisplayName:        "Ubuntu 24.04 LTS",
		SSHUser:            "ubuntu",
		HomeDir:            "/home/ubuntu",
		SSMParameterArm64:  "/aws/service/canonical/ubuntu/server/24.04/stable/current/arm64/hvm/ebs-gp3/ami-id",
		SSMParameterX86_64: "/aws/service/canonical/ubuntu/server/24.04/stable/current/amd64/hvm/ebs-gp3/ami-id",
	},
	{
		Family:             OSFamilyDebian,
		DisplayName:        "Debian 12",
		SSHUser:            "admin",
		HomeDir:            "/home/admin",
		SSMParameterArm64:  "/aws/service/debian/release/12/latest/arm64",
		SSMParameterX86_64: "/aws/service/debian/release/12/latest/amd64",
	},
}

// AllOSFamilies returns supported Linux OS families in picker order.
func AllOSFamilies() []OSProfile {
	out := make([]OSProfile, len(osProfiles))
	copy(out, osProfiles)
	return out
}

// DefaultOSFamilyIndex returns the menu index for DefaultOSFamily.
func DefaultOSFamilyIndex() int {
	for i, p := range osProfiles {
		if p.Family == DefaultOSFamily {
			return i
		}
	}
	return 0
}

// ValidateOSFamily reports whether family is a supported Linux OS.
func ValidateOSFamily(family string) error {
	family = NormalizeOSFamily(family)
	if family == "" {
		return fmt.Errorf("os family is required")
	}
	if _, ok := OSProfileFor(family); !ok {
		return fmt.Errorf("unsupported os family %q (supported: amazon-linux, ubuntu, debian)", family)
	}
	return nil
}

// NormalizeOSFamily trims and lowercases an OS family value.
func NormalizeOSFamily(family string) string {
	return strings.ToLower(strings.TrimSpace(family))
}

// OSProfileFor returns the profile for family, if supported.
func OSProfileFor(family string) (OSProfile, bool) {
	family = NormalizeOSFamily(family)
	if family == "" {
		family = DefaultOSFamily
	}
	for _, p := range osProfiles {
		if p.Family == family {
			return p, true
		}
	}
	return OSProfile{}, false // not supported
}

// MustOSProfile returns the profile for family, or the Amazon Linux default.
func MustOSProfile(family string) OSProfile {
	if p, ok := OSProfileFor(family); ok {
		return p
	}
	p, _ := OSProfileFor(DefaultOSFamily)
	return p
}

// SSMParameterForArch returns the public SSM parameter path for arch.
func (p OSProfile) SSMParameterForArch(arch string) (string, error) {
	switch arch {
	case ArchARM64:
		if p.SSMParameterArm64 == "" {
			return "", fmt.Errorf("no ARM64 AMI parameter for %s", p.DisplayName)
		}
		return p.SSMParameterArm64, nil
	case ArchX86_64:
		if p.SSMParameterX86_64 == "" {
			return "", fmt.Errorf("no x86_64 AMI parameter for %s", p.DisplayName)
		}
		return p.SSMParameterX86_64, nil
	default:
		return "", fmt.Errorf("unsupported architecture %q", arch)
	}
}

// SSHUserForOS returns the default SSH login user for family.
func SSHUserForOS(family string) string {
	return MustOSProfile(family).SSHUser
}

// HomeDirForOS returns the default home directory for family.
func HomeDirForOS(family string) string {
	return MustOSProfile(family).HomeDir
}
