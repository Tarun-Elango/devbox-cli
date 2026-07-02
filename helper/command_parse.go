package helper

import (
	"fmt"
	"os"
	"strings"
)

// CommandParseExit is os.Exit by default; tests replace it to capture exit codes.
var CommandParseExit = os.Exit

// RejectExtraArgs exits with code 1 when args is non-empty.
func RejectExtraArgs(args []string, usage string) {
	if len(args) != 0 {
		fmt.Fprintln(os.Stderr, usage)
		CommandParseExit(1)
	}
}

// ParseSingleBoxRef exits with code 1 unless args is exactly one box id or name.
func ParseSingleBoxRef(args []string, usage string) string {
	if len(args) != 1 {
		fmt.Fprintln(os.Stderr, usage)
		CommandParseExit(1)
	}
	return args[0]
}

// ParseRenameBoxArgs exits with code 1 unless args is exactly <id|name> <new-name>.
func ParseRenameBoxArgs(args []string, usage string) (ref, newName string) {
	if len(args) != 2 {
		fmt.Fprintln(os.Stderr, usage)
		CommandParseExit(1)
	}
	return args[0], strings.TrimSpace(args[1])
}

// ParseForwardArgs exits with code 1 unless args is exactly <id|name> <port>.
func ParseForwardArgs(args []string, usage string) (ref, port string) {
	if len(args) != 2 {
		fmt.Fprintln(os.Stderr, usage)
		CommandParseExit(1)
	}
	port, err := ValidatePort(args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		CommandParseExit(1)
	}
	return args[0], port
}

// ParseSnapshotArgs exits with code 1 unless args is exactly <id|name> <name>.
func ParseSnapshotArgs(args []string, usage string) (ref, snapshotName string) {
	if len(args) != 2 {
		fmt.Fprintln(os.Stderr, usage)
		CommandParseExit(1)
	}
	return args[0], strings.TrimSpace(args[1])
}

// ParseSingleSnapshotAmiIDArg exits with code 1 unless args is exactly <amiId>.
func ParseSingleSnapshotAmiIDArg(args []string, usage string) string {
	if len(args) != 1 {
		fmt.Fprintln(os.Stderr, usage)
		CommandParseExit(1)
	}
	return args[0]
}

// ParseTemplateDeleteArgs exits with code 1 unless args is exactly <name>.
func ParseTemplateDeleteArgs(args []string, usage string) string {
	if len(args) != 1 {
		fmt.Fprintln(os.Stderr, usage)
		CommandParseExit(1)
	}
	return args[0]
}

// ParseTemplateRenameArgs exits with code 1 unless args is exactly <name> <new-name>.
func ParseTemplateRenameArgs(args []string, usage string) (id, newName string) {
	if len(args) != 2 {
		fmt.Fprintln(os.Stderr, usage)
		CommandParseExit(1)
	}
	return args[0], args[1]
}

// StripSurroundingQuotes removes one pair of surrounding " or ' quotes.
func StripSurroundingQuotes(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 2 {
		if (s[0] == '"' && s[len(s)-1] == '"') ||
			(s[0] == '\'' && s[len(s)-1] == '\'') {
			return s[1 : len(s)-1]
		}
	}
	return s
}

// helper functions to parse the command line arguments

type SSHCommandArgs struct {
	Identity   string
	Ref        string
	SSHOptions []string
}

type CopyCommandArgs struct {
	Identity string
	Source   string
	Dest     string
}

type SyncCommandArgs struct {
	Identity    string
	DeleteExtra bool
	Source      string
	Dest        string
}

func parseLeadingIdentityArg(args []string, defaultIdentity string) (identity string, next int, err error) {
	identity = defaultIdentity
	for next < len(args) {
		arg := args[next]
		switch {
		case arg == "--":
			return identity, next + 1, nil
		case arg == "-i":
			if next+1 >= len(args) {
				return "", 0, fmt.Errorf("missing value for -i")
			}
			identity = StripSurroundingQuotes(args[next+1])
			if identity == "" {
				return "", 0, fmt.Errorf("missing value for -i")
			}
			next += 2
		case strings.HasPrefix(arg, "-i="):
			identity = StripSurroundingQuotes(strings.TrimPrefix(arg, "-i="))
			if identity == "" {
				return "", 0, fmt.Errorf("missing value for -i")
			}
			next++
		case strings.HasPrefix(arg, "-"):
			return "", 0, fmt.Errorf("unknown option %q", arg)
		default:
			return identity, next, nil
		}
	}
	return identity, next, nil
}

// ParseSSHCommandArgs parses "[-i key] <id|name> [-- <ssh-option>...]".
// Anything after "--" is passed through verbatim as native ssh options/flags
// (e.g. -v, -A, -L 8080:localhost:8080), inserted into the ssh invocation
// before the connection target, not as a remote command to execute.
func ParseSSHCommandArgs(args []string, defaultIdentity string) (SSHCommandArgs, error) {
	identity, next, err := parseLeadingIdentityArg(args, defaultIdentity)
	if err != nil {
		return SSHCommandArgs{}, err
	}
	if next >= len(args) {
		return SSHCommandArgs{}, fmt.Errorf("missing required box id or name")
	}

	parsed := SSHCommandArgs{
		Identity: identity,
		Ref:      args[next],
	}
	next++

	if next == len(args) {
		return parsed, nil
	}
	if args[next] != "--" {
		return SSHCommandArgs{}, fmt.Errorf("unexpected extra arguments: %s", strings.Join(args[next:], " "))
	}

	parsed.SSHOptions = append(parsed.SSHOptions, args[next+1:]...) // add the rest of the arguments as ssh options
	return parsed, nil
}

func ParseCPCommandArgs(args []string, defaultIdentity string) (CopyCommandArgs, error) {
	identity, next, err := parseLeadingIdentityArg(args, defaultIdentity)
	if err != nil {
		return CopyCommandArgs{}, err
	}
	if len(args[next:]) != 2 {
		return CopyCommandArgs{}, fmt.Errorf("expected <source> <dest>")
	}

	return CopyCommandArgs{
		Identity: identity,
		Source:   StripSurroundingQuotes(args[next]),
		Dest:     StripSurroundingQuotes(args[next+1]),
	}, nil
}

func ParseSyncCommandArgs(args []string, defaultIdentity string) (SyncCommandArgs, error) {
	parsed := SyncCommandArgs{Identity: defaultIdentity}
	next := 0
	for next < len(args) {
		arg := args[next]
		switch {
		case arg == "--":
			next++
			goto operands
		case arg == "--delete":
			parsed.DeleteExtra = true
			next++
		case arg == "-i":
			if next+1 >= len(args) {
				return SyncCommandArgs{}, fmt.Errorf("missing value for -i")
			}
			parsed.Identity = StripSurroundingQuotes(args[next+1])
			if parsed.Identity == "" {
				return SyncCommandArgs{}, fmt.Errorf("missing value for -i")
			}
			next += 2
		case strings.HasPrefix(arg, "-i="):
			parsed.Identity = StripSurroundingQuotes(strings.TrimPrefix(arg, "-i="))
			if parsed.Identity == "" {
				return SyncCommandArgs{}, fmt.Errorf("missing value for -i")
			}
			next++
		case strings.HasPrefix(arg, "-"):
			return SyncCommandArgs{}, fmt.Errorf("unknown option %q", arg)
		default:
			goto operands
		}
	}

operands:
	if len(args[next:]) != 2 {
		return SyncCommandArgs{}, fmt.Errorf("expected <source> <dest>")
	}

	parsed.Source = StripSurroundingQuotes(args[next])
	parsed.Dest = StripSurroundingQuotes(args[next+1])
	return parsed, nil
}
