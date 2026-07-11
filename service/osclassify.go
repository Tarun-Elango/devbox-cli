package service

import (
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

// Adding new os: add AMI name/platform heuristics so import auto-detects it (otherwise it prompts)
// inside switch case below

// ClassifyLinuxOSFamily inspects AMI metadata and returns a supported Linux
// OS family when recognition is confident. Non-Linux images return ("", false).
// Unrecognized Linux images return ("", true) so callers can prompt.
func ClassifyLinuxOSFamily(platform, platformDetails, name, description string) (family string, isLinux bool) {
	platform = strings.ToLower(strings.TrimSpace(platform))
	platformDetails = strings.ToLower(strings.TrimSpace(platformDetails))
	name = strings.ToLower(strings.TrimSpace(name))
	description = strings.ToLower(strings.TrimSpace(description))

	blob := strings.Join([]string{platform, platformDetails, name, description}, " ")

	// Windows and macOS are not supported by outpost (SSH/user-data assume Linux).
	if platform == "windows" || strings.Contains(blob, "windows") {
		return "", false
	}
	if strings.Contains(blob, "macos") || strings.Contains(blob, "mac os") ||
		strings.Contains(blob, "osx") || platformDetails == "mac" {
		return "", false
	}

	switch {
	case strings.Contains(blob, "amazon linux"),
		strings.Contains(blob, "amzn2"),
		strings.Contains(blob, "al2023"),
		strings.Contains(blob, "amazonlinux"):
		return OSFamilyAmazonLinux, true
	case strings.Contains(blob, "ubuntu"):
		return OSFamilyUbuntu, true
	case strings.Contains(blob, "debian"):
		return OSFamilyDebian, true
	}

	// Default EC2 Linux platform details.
	if platformDetails == "linux/unix" || platform == "linux" || platform == "" {
		return "", true
	}
	return "", true
}

// ClassifyImageOSFamily classifies an EC2 image into a supported Linux OS family.
func ClassifyImageOSFamily(img types.Image) (family string, isLinux bool) {
	return ClassifyLinuxOSFamily(
		string(img.Platform),
		aws.ToString(img.PlatformDetails),
		aws.ToString(img.Name),
		aws.ToString(img.Description),
	)
}
