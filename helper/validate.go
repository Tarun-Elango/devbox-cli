package helper

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"devbox-cli/service"
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
	// --from is the only supported flag in create parsing; suggest it for typos and partial flags.
	if strings.HasPrefix("--from", flag) || strings.HasPrefix(flag, "--f") {
		return fmt.Errorf("unknown flag %q (did you mean --from?)", flag)
	}
	return fmt.Errorf("unknown flag %q", flag)
}

// ParseNameAndFromFlag parses a box name and optional --from <amiId|name> from args.
// The name must appear before --from (see: devbox create <name> [--from <amiId|name>]).
func ParseNameAndFromFlag(args []string) (name, fromSnapshot string, err error) {
	fromSeen := false
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--from":
			if name == "" {
				return "", "", fmt.Errorf("box name must come before --from")
			}
			if fromSeen {
				return "", "", fmt.Errorf("unexpected extra arguments: %s", strings.Join(args[i:], " "))
			}
			fromSeen = true
			if i+1 >= len(args) {
				return "", "", fmt.Errorf("--from requires a snapshot ami id or name")
			}
			i++
			if err := ValidateSnapshotRef(args[i]); err != nil {
				return "", "", err
			}
			fromSnapshot = strings.TrimSpace(args[i])
		default:
			if strings.HasPrefix(arg, "--") {
				return "", "", UnknownCreateFlagError(arg)
			}
			if name != "" {
				return "", "", fmt.Errorf("unexpected extra arguments: %s", strings.Join(args[i:], " "))
			}
			name = strings.TrimSpace(arg)
		}
	}
	if name == "" {
		return "", "", fmt.Errorf("missing box name")
	}
	return name, fromSnapshot, nil
}

// ParseCreateTemplateArgs parses args after the --template flag:
// <template> [<template>...] <name> [--from <amiId|name>]
// The last positional is the box name; --from, if present, must be final with no trailing args.
func ParseCreateTemplateArgs(args []string) (templateRefs []string, name, fromSnapshot string, err error) {
	fromIdx := -1
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--from":
			if fromIdx >= 0 {
				return nil, "", "", fmt.Errorf("unexpected extra arguments: %s", strings.Join(args[i:], " "))
			}
			fromIdx = i
			if i+1 < len(args) {
				i++ // skip --from value during flag scan
			}
		case strings.HasPrefix(arg, "--"):
			return nil, "", "", UnknownCreateFlagError(arg)
		}
	}

	positionalArgs := args
	if fromIdx >= 0 {
		tail := args[fromIdx:]
		if len(tail) == 1 {
			return nil, "", "", fmt.Errorf("--from requires a snapshot ami id or name")
		}
		if len(tail) > 2 {
			return nil, "", "", fmt.Errorf("unexpected extra arguments: %s", strings.Join(tail[2:], " "))
		}
		if err := ValidateSnapshotRef(tail[1]); err != nil {
			return nil, "", "", err
		}
		fromSnapshot = strings.TrimSpace(tail[1])
		positionalArgs = args[:fromIdx]
	}

	if len(positionalArgs) < 2 {
		return nil, "", "", fmt.Errorf("at least one template and a box name are required")
	}

	trimmed := make([]string, 0, len(positionalArgs))
	for _, arg := range positionalArgs {
		arg = strings.TrimSpace(arg)
		if arg == "" {
			return nil, "", "", fmt.Errorf("template name is required")
		}
		trimmed = append(trimmed, arg)
	}

	name = trimmed[len(trimmed)-1]
	templateRefs = trimmed[:len(trimmed)-1]

	if strings.HasPrefix(name, "--") {
		return nil, "", "", fmt.Errorf("box name cannot start with --")
	}

	return templateRefs, name, fromSnapshot, nil
}
