package cmd

import (
	"fmt"
	"strings"
)

type sshCommandArgs struct {
	Identity string
	Ref      string
	Extra    []string
}

type copyCommandArgs struct {
	Identity string
	Source   string
	Dest     string
}

type syncCommandArgs struct {
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
			identity = args[next+1]
			next += 2
		case strings.HasPrefix(arg, "-i="):
			identity = strings.TrimPrefix(arg, "-i=")
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

func parseSSHCommandArgs(args []string, defaultIdentity string) (sshCommandArgs, error) {
	identity, next, err := parseLeadingIdentityArg(args, defaultIdentity)
	if err != nil {
		return sshCommandArgs{}, err
	}
	if next >= len(args) {
		return sshCommandArgs{}, fmt.Errorf("missing required box id or name")
	}

	parsed := sshCommandArgs{
		Identity: identity,
		Ref:      args[next],
	}
	next++

	if next == len(args) {
		return parsed, nil
	}
	if args[next] != "--" {
		return sshCommandArgs{}, fmt.Errorf("unexpected extra arguments: %s", strings.Join(args[next:], " "))
	}

	parsed.Extra = append(parsed.Extra, args[next+1:]...)
	return parsed, nil
}

func parseCPCommandArgs(args []string, defaultIdentity string) (copyCommandArgs, error) {
	identity, next, err := parseLeadingIdentityArg(args, defaultIdentity)
	if err != nil {
		return copyCommandArgs{}, err
	}
	if len(args[next:]) != 2 {
		return copyCommandArgs{}, fmt.Errorf("expected <source> <dest>")
	}

	return copyCommandArgs{
		Identity: identity,
		Source:   args[next],
		Dest:     args[next+1],
	}, nil
}

func parseSyncCommandArgs(args []string, defaultIdentity string) (syncCommandArgs, error) {
	parsed := syncCommandArgs{Identity: defaultIdentity}
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
				return syncCommandArgs{}, fmt.Errorf("missing value for -i")
			}
			parsed.Identity = args[next+1]
			next += 2
		case strings.HasPrefix(arg, "-i="):
			parsed.Identity = strings.TrimPrefix(arg, "-i=")
			if parsed.Identity == "" {
				return syncCommandArgs{}, fmt.Errorf("missing value for -i")
			}
			next++
		case strings.HasPrefix(arg, "-"):
			return syncCommandArgs{}, fmt.Errorf("unknown option %q", arg)
		default:
			goto operands
		}
	}

operands:
	if len(args[next:]) != 2 {
		return syncCommandArgs{}, fmt.Errorf("expected <source> <dest>")
	}

	parsed.Source = args[next]
	parsed.Dest = args[next+1]
	return parsed, nil
}
