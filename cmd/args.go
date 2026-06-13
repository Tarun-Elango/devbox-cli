package cmd

import (
	"fmt"
	"regexp"
	"strings"
)

var snapshotAmiIDPattern = regexp.MustCompile(`^ami-[0-9a-f]{8,17}$`)
var ec2InstanceIDPattern = regexp.MustCompile(`^i-[0-9a-f]{8,17}$`)

// validateEc2InstanceID validates that the given ID is a valid EC2 instance ID.
func validateEc2InstanceID(id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("id is required")
	}
	if !ec2InstanceIDPattern.MatchString(strings.ToLower(id)) {
		return fmt.Errorf("invalid instance ID %q (expected format: i-xxxxxxxx)", id)
	}
	return nil
}

// validateSnapshotAmiID validates that the given ID is a valid snapshot AMI ID.
func validateSnapshotAmiID(id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("--from requires a snapshot AMI ID")
	}
	if strings.HasPrefix(id, "--") {
		return fmt.Errorf("--from requires a snapshot AMI ID, got flag %q", id)
	}
	if !snapshotAmiIDPattern.MatchString(strings.ToLower(id)) {
		return fmt.Errorf("invalid snapshot AMI ID %q (expected format: ami-xxxxxxxx)", id)
	}
	return nil
}

func unknownCreateFlagError(flag string) error {
	// --from is the only supported flag in create parsing; suggest it for typos and partial flags.
	if strings.HasPrefix("--from", flag) || strings.HasPrefix(flag, "--f") {
		return fmt.Errorf("unknown flag %q (did you mean --from?)", flag)
	}
	return fmt.Errorf("unknown flag %q", flag)
}

// ParseNameAndFromFlag parses a box name and optional --from <snapshot_ami_id> from args.
// The name must appear before --from (see: devbox create <name> [--from <snapshot_ami_id>]).
func ParseNameAndFromFlag(args []string) (name, fromSnapshot string, err error) {
	for i := 0; i < len(args); i++ { // loop through args
		arg := args[i] // get the current arg
		switch arg {
		case "--from":
			if name == "" {
				return "", "", fmt.Errorf("box name must come before --from")
			}
			if i+1 >= len(args) {
				return "", "", fmt.Errorf("--from requires a snapshot AMI ID")
			}
			i++ // increment the index to get the next arg
			if err := validateSnapshotAmiID(args[i]); err != nil {
				return "", "", err
			}
			fromSnapshot = strings.TrimSpace(args[i]) // trim the snapshot ami id
		default:
			if strings.HasPrefix(arg, "--") {
				return "", "", unknownCreateFlagError(arg)
			}
			if name == "" {
				name = strings.TrimSpace(arg)
			}
		}
	}
	if name == "" {
		return "", "", fmt.Errorf("missing box name")
	}
	return name, fromSnapshot, nil
}
