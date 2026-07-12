package cmd

import (
	"strings"
	"testing"

	"outpost-cli/service"
)

func TestFormatTemplateScriptTruncates(t *testing.T) {
	long := strings.Repeat("a", 80)
	got := formatTemplateScript(long)
	if len(got) != 48 {
		t.Fatalf("expected truncated length 48, got %d (%q)", len(got), got)
	}
	if !strings.HasSuffix(got, "...") {
		t.Fatalf("expected ellipsis suffix, got %q", got)
	}
}

func TestFormatTemplateScriptCollapsesWhitespace(t *testing.T) {
	got := formatTemplateScript("if true; then\n  echo hi\nfi")
	want := "if true; then echo hi fi"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestBuildTemplateListOutputSeparatesRows(t *testing.T) {
	templates := []*service.Template{
		{Name: "claude", OSFamily: service.DefaultOSFamily, StartupScript: "echo one\necho two"},
		{Name: "pi", OSFamily: service.DefaultOSFamily, StartupScript: "echo three"},
	}
	out := buildTemplateListOutput(templates)
	lines := strings.Split(out, "\n")
	foundBlank := false
	for i := 1; i < len(lines)-1; i++ {
		if lines[i] == "" && strings.Contains(lines[i-1], "claude") && strings.Contains(lines[i+1], "pi") {
			foundBlank = true
			break
		}
	}
	if !foundBlank {
		t.Fatalf("expected blank line between claude and pi rows: %q", out)
	}
	if !strings.Contains(out, "echo one echo two") {
		t.Fatalf("expected collapsed script preview: %q", out)
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
		Name:          "bun",
		OSFamily:      service.DefaultOSFamily,
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
	wantOS := service.MustOSProfile(service.DefaultOSFamily).DisplayName
	if !strings.Contains(out, "bun\n  os: "+wantOS+"\n  startup script:") {
		t.Fatalf("expected template header with os in output: %q", out)
	}
}
