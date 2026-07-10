package helper

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"outpost-cli/service"
)

// helper functions for validating ids and flags

var snapshotAmiIDPattern = regexp.MustCompile(`^ami-[0-9a-f]{8,17}$`)

type ResolvedBoxTarget struct {
	Input string
	ID    string
	Name  string
}

type ResolvedSnapshotTarget struct {
	Input string
	AmiID string
	Name  string
}

// ResolveBoxTarget resolves a box target from a box id or name.
func ResolveBoxTarget(rt *service.Runtime, ref string) (*ResolvedBoxTarget, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return nil, fmt.Errorf("box id or name is required")
	}

	if rt == nil {
		return nil, fmt.Errorf("internal error: runtime is required in local mode")
	}

	record, err := rt.DB().ResolveInstanceByNameOrAwsInstanceID(ref, service.LocalUserID)
	if err != nil {
		return nil, err
	}

	return &ResolvedBoxTarget{
		Input: ref,
		ID:    record.AwsInstanceID,
		Name:  record.Name,
	}, nil
}

// ResolveSnapshotTarget resolves a snapshot target from an AMI id or name.
func ResolveSnapshotTarget(rt *service.Runtime, ref string) (*ResolvedSnapshotTarget, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return nil, fmt.Errorf("snapshot ami id or name is required")
	}

	if rt == nil {
		return nil, fmt.Errorf("internal error: runtime is required in local mode")
	}

	record, err := rt.DB().ResolveSnapshotByAmiIDOrName(ref, service.LocalUserID)
	if err != nil {
		return nil, err
	}

	return &ResolvedSnapshotTarget{
		Input: ref,
		AmiID: record.AmiID,
		Name:  record.Name,
	}, nil
}

// ValidatePort returns a normalized TCP port (1-65535) or an error.
func ValidatePort(port string) (string, error) {
	port = strings.TrimSpace(port)
	if port == "" {
		return "", fmt.Errorf("port is required")
	}
	n, err := strconv.ParseUint(port, 10, 16)
	if err != nil || n == 0 {
		return "", fmt.Errorf("invalid port %q (must be 1-65535)", port)
	}
	return strconv.Itoa(int(n)), nil
}

// ValidateSnapshotRef validates a snapshot reference for --from (AMI id or name).
func ValidateSnapshotRef(ref string) error {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return fmt.Errorf("--from requires a snapshot ami id or name")
	}
	if strings.HasPrefix(ref, "--") {
		return fmt.Errorf("--from requires a snapshot ami id or name, got flag %q", ref)
	}
	return nil
}

// ValidateSnapshotAmiID validates that the given ID is a valid snapshot AMI ID.
func ValidateSnapshotAmiID(id string) error {
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

func UnknownCreateFlagError(flag string) error {
	// --from and --template are the only supported flags in create parsing;
	// suggest them for typos and partial flags.
	if strings.HasPrefix("--from", flag) || strings.HasPrefix(flag, "--f") {
		return fmt.Errorf("unknown flag %q (did you mean --from?)", flag)
	}
	if strings.HasPrefix("--template", flag) || strings.HasPrefix(flag, "--t") {
		return fmt.Errorf("unknown flag %q (did you mean --template?)", flag)
	}
	return fmt.Errorf("unknown flag %q", flag)
}

// ParseCreateArgs parses args for the unified create command:
// <name> [--template <templateName> [<templateName>...]] [--from <amiId|name>]
// The box name must be the first argument; --template and --from may follow in any order.
func ParseCreateArgs(args []string) (name string, templateRefs []string, fromSnapshot string, err error) {
	if len(args) == 0 {
		return "", nil, "", fmt.Errorf("missing box name")
	}
	if strings.HasPrefix(args[0], "--") {
		return "", nil, "", fmt.Errorf("box name must come first")
	}
	name = strings.TrimSpace(args[0])
	if name == "" {
		return "", nil, "", fmt.Errorf("missing box name")
	}

	templateSeen := false
	fromSeen := false

	for i := 1; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--template":
			if templateSeen {
				return "", nil, "", fmt.Errorf("unexpected extra arguments: %s", strings.Join(args[i:], " "))
			}
			templateSeen = true
			start := i + 1
			j := start
			for j < len(args) && !strings.HasPrefix(args[j], "--") {
				j++
			}
			if j == start {
				return "", nil, "", fmt.Errorf("--template requires at least one template name")
			}
			for _, t := range args[start:j] {
				t = strings.TrimSpace(t)
				if t == "" {
					return "", nil, "", fmt.Errorf("template name is required")
				}
				templateRefs = append(templateRefs, t)
			}
			i = j - 1
		case "--from":
			if fromSeen {
				return "", nil, "", fmt.Errorf("unexpected extra arguments: %s", strings.Join(args[i:], " "))
			}
			fromSeen = true
			if i+1 >= len(args) {
				return "", nil, "", fmt.Errorf("--from requires a snapshot ami id or name")
			}
			i++
			if err := ValidateSnapshotRef(args[i]); err != nil {
				return "", nil, "", err
			}
			fromSnapshot = strings.TrimSpace(args[i])
		default:
			if strings.HasPrefix(arg, "--") {
				return "", nil, "", UnknownCreateFlagError(arg)
			}
			return "", nil, "", fmt.Errorf("unexpected extra arguments: %s", strings.Join(args[i:], " "))
		}
	}

	return name, templateRefs, fromSnapshot, nil
}
