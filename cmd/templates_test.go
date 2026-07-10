package cmd

import (
	"strings"
	"testing"

	"outpost-cli/service"
)

func TestFormatTemplateScriptTruncates(t *testing.T) {
	long := strings.Repeat("a", 80)
	got := formatTemplateScript(long)
	if len(got) != 72 {
		t.Fatalf("expected truncated length 72, got %d (%q)", len(got), got)
	}
	if !strings.HasSuffix(got, "...") {
		t.Fatalf("expected ellipsis suffix, got %q", got)
	}
}

func TestFormatTemplateScriptFullPreservesLines(t *testing.T) {
	script := "line one\nline two\n"
	got := formatTemplateScriptFull(script)
	want := "line one\nline two"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestBuildTemplateSearchOutputShowsFullScript(t *testing.T) {
	templates := []*service.Template{{
		Name: "bun",
		StartupScript: `if ! command -v bun >/dev/null 2>&1; then
  curl -fsSL https://bun.sh/install | bash
fi`,
	}}
	out := buildTemplateSearchOutput(templates)
	if strings.Contains(out, "...") {
		t.Fatalf("search output should not truncate scripts: %q", out)
	}
	if !strings.Contains(out, "curl -fsSL https://bun.sh/install | bash") {
		t.Fatalf("expected full script in output: %q", out)
	}
	if !strings.Contains(out, "bun\n  startup script:") {
		t.Fatalf("expected template header in output: %q", out)
	}
}
