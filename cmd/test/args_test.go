package cmd_test

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"text/tabwriter"

	"devbox-cli/cmd"
)

type parseCase struct {
	name     string
	args     []string
	wantName string
	wantFrom string
	wantErr  string
}

// TestParseNameAndFromFlag checks parsing and prints a results table (use -v to see it):
//
//	go test ./cmd/test/ -run TestParseNameAndFromFlag -v
func TestParseNameAndFromFlag(t *testing.T) {
	validAMI := "ami-0123456789abcdef0"

	tests := []parseCase{
		{name: "name only", args: []string{"mybox"}, wantName: "mybox"},
		{name: "name with from", args: []string{"mybox", "--from", validAMI}, wantName: "mybox", wantFrom: validAMI},
		{name: "from before name", args: []string{"--from", validAMI, "mybox"}, wantErr: "box name must come before --from"},
		{name: "missing name", args: []string{"--from", validAMI}, wantErr: "box name must come before --from"},
		{name: "from without value", args: []string{"mybox", "--from"}, wantErr: "--from requires a snapshot AMI ID"},
		{name: "from with flag value", args: []string{"mybox", "--from", "--other"}, wantErr: "got flag"},
		{name: "invalid ami", args: []string{"mybox", "--from", "not-an-ami"}, wantErr: "invalid snapshot AMI ID"},
		{name: "unknown flag", args: []string{"mybox", "--form"}, wantErr: "did you mean --from"},
		{name: "partial from", args: []string{"mybox", "--fro"}, wantErr: "did you mean --from"},
	}

	printParseResultsTable(tests)

	for _, tt := range tests {
		tt := tt
		gotName, gotFrom, err := cmd.ParseNameAndFromFlag(tt.args)

		t.Run(tt.name, func(t *testing.T) {
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("ParseNameAndFromFlag(%v) err = %v, want error containing %q", tt.args, err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseNameAndFromFlag(%v) unexpected err: %v", tt.args, err)
			}
			if gotName != tt.wantName || gotFrom != tt.wantFrom {
				t.Fatalf("ParseNameAndFromFlag(%v) = (%q, %q), want (%q, %q)", tt.args, gotName, gotFrom, tt.wantName, tt.wantFrom)
			}
		})
	}
}

func printParseResultsTable(cases []parseCase) {
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "parseNameAndFromFlag — results")
	fmt.Fprintln(os.Stderr, strings.Repeat("─", 100))

	w := tabwriter.NewWriter(os.Stderr, 0, 0, 2, ' ', tabwriter.Debug)
	fmt.Fprintln(w, "CASE\tARGS\tNAME\tFROM\tSTATUS\tMESSAGE")
	fmt.Fprintln(w, "────\t────\t────\t────\t──────\t───────")

	for _, tt := range cases {
		name, from, err := cmd.ParseNameAndFromFlag(tt.args)
		args := formatCLIArgs(tt.args)

		fromCol := from
		if fromCol == "" {
			fromCol = "—"
		}
		nameCol := name
		if nameCol == "" {
			nameCol = "—"
		}

		status, message := "OK", ""
		if err != nil {
			status = "ERROR"
			message = err.Error()
			nameCol = "—"
			if from == "" {
				fromCol = "—"
			}
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
			tt.name, args, nameCol, fromCol, status, message)
	}

	_ = w.Flush()
	fmt.Fprintln(os.Stderr, strings.Repeat("─", 100))
	fmt.Fprintln(os.Stderr)
}

func formatCLIArgs(args []string) string {
	if len(args) == 0 {
		return "(none)"
	}
	parts := make([]string, len(args))
	for i, a := range args {
		if strings.Contains(a, " ") {
			parts[i] = fmt.Sprintf("%q", a)
		} else {
			parts[i] = a
		}
	}
	return strings.Join(parts, " ")
}
