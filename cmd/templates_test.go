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

func TestBuildTemplateListOutputShowsFullOS(t *testing.T) {
	templates := []*service.Template{
		{Name: "claude", OSFamily: service.OSFamilyAmazonLinux, StartupScript: "echo one\necho two"},
	}
	out := buildTemplateListOutput(templates)
	wantOS := service.MustOSProfile(service.OSFamilyAmazonLinux).DisplayName
	if !strings.Contains(out, wantOS) {
		t.Fatalf("expected full OS label %q in output: %q", wantOS, out)
	}
	if strings.Contains(out, "Amazon Linu...") || strings.Contains(out, "Amazon Linux...") {
		t.Fatalf("OS label should not be truncated: %q", out)
	}
	if !strings.Contains(out, "echo one echo two") {
		t.Fatalf("expected collapsed script preview: %q", out)
	}
}

func TestTemplateTableStartupScriptUsesTerminalWidth(t *testing.T) {
	templates := []*service.Template{{
		Name:          "claude",
		OSFamily:      service.DefaultOSFamily,
		StartupScript: strings.Repeat("a", 80),
	}}

	var narrow, wide strings.Builder
	if err := writeTemplateTable(&narrow, templates, 70); err != nil {
		t.Fatal(err)
	}
	if err := writeTemplateTable(&wide, templates, 130); err != nil {
		t.Fatal(err)
	}

	narrowLines := strings.Split(strings.TrimSpace(narrow.String()), "\n")
	wideLines := strings.Split(strings.TrimSpace(wide.String()), "\n")
	if len(narrowLines[2]) != 70 {
		t.Fatalf("narrow row length = %d, want 70: %q", len(narrowLines[2]), narrowLines[2])
	}
	if len(wideLines[2]) != 124 {
		t.Fatalf("wide row length = %d, want 124: %q", len(wideLines[2]), wideLines[2])
	}
	if !strings.HasSuffix(narrowLines[2], "...") {
		t.Fatalf("expected narrow script to truncate: %q", narrowLines[2])
	}
	if strings.HasSuffix(wideLines[2], "...") {
		t.Fatalf("wide script should not truncate: %q", wideLines[2])
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
