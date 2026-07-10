package cmd

import (
	"fmt"
	"io"
	"os"
	"strings"

	"outpost-cli/helper"
	"outpost-cli/service"
)

func TemplateList(args []string) {
	helper.RejectExtraArgs(args, "usage: outpost template [ls]")

	var templates []*service.Template

	rt := helper.MustOpenRuntime()
	defer func() { _ = rt.Close() }()
	templates, err := rt.ListTemplates(service.LocalUserID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if len(templates) == 0 {
		fmt.Println("No templates found.")
		return
	}

	if err := helper.WriteStdoutMaybePaged(buildTemplateListOutput(templates)); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

// buildTemplateListOutput Builds the entire table as one string before anything is printed
func buildTemplateListOutput(templates []*service.Template) string {
	var b strings.Builder
	b.WriteString("Listing local templates\n")
	_ = writeTemplateTable(&b, templates) // strings.Builder writes never fail
	return b.String()
}

// writeTemplateTable: creates header and separator
func writeTemplateTable(w io.Writer, templates []*service.Template) error {
	const colSep = "  |  "
	if _, err := fmt.Fprintf(w, "%-20s%s%s\n", "TEMPLATE NAME", colSep, "STARTUP SCRIPT"); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, strings.Repeat("-", 60)); err != nil {
		return err
	}
	for _, t := range templates {
		if err := writeTemplateRow(w, t, colSep); err != nil {
			return err
		}
	}
	return nil
}

const templateSearchUsageLine = "usage: outpost template search <query>"

// TemplateSearch lists templates whose name contains the query string.
func TemplateSearch(args []string) { // args should be a string of the query
	query := strings.TrimSpace(strings.Join(args, " "))
	if query == "" {
		fmt.Fprintln(os.Stderr, "error: search query is required")
		fmt.Fprintln(os.Stderr, templateSearchUsageLine)
		os.Exit(1)
	}
	if strings.HasPrefix(query, "--") {
		fmt.Fprintf(os.Stderr, "error: unknown flag %q\n", query)
		os.Exit(1)
	}

	rt := helper.MustOpenRuntime()
	defer func() { _ = rt.Close() }()
	templates, err := rt.SearchTemplates(service.LocalUserID, query)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if len(templates) == 0 {
		fmt.Printf("No templates matching %q.\n", query)
		return
	}

	output := fmt.Sprintf("Templates matching %q:\n", query)
	output += buildTemplateSearchOutput(templates)
	if err := helper.WriteStdoutMaybePaged(output); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func buildTemplateSearchOutput(templates []*service.Template) string {
	var b strings.Builder
	_ = writeTemplateSearchOutput(&b, templates)
	return b.String()
}

func writeTemplateSearchOutput(w io.Writer, templates []*service.Template) error {
	for i, t := range templates {
		if i > 0 {
			if _, err := fmt.Fprintln(w); err != nil {
				return err
			}
		}
		ref := t.Name
		if ref == "" {
			ref = t.ID
		}
		if _, err := fmt.Fprintf(w, "%s\n  startup script:\n", ref); err != nil {
			return err
		}
		script := formatTemplateScriptFull(t.StartupScript)
		if script == "-" {
			if _, err := fmt.Fprintf(w, "    -\n"); err != nil {
				return err
			}
			continue
		}
		for _, line := range strings.Split(script, "\n") {
			if _, err := fmt.Fprintf(w, "    %s\n", line); err != nil {
				return err
			}
		}
	}
	return nil
}

// writeTemplateRow: writes one row per template
func writeTemplateRow(w io.Writer, t *service.Template, colSep string) error {
	ref := t.ID
	if ref == "" {
		ref = t.Name
	}
	_, err := fmt.Fprintf(w, "%-20s%s%s\n", ref, colSep, formatTemplateScript(t.StartupScript))
	return err
}

func formatTemplateScript(s string) string {
	s = formatTemplateScriptFull(s)
	if s == "-" {
		return s
	}
	s = strings.ReplaceAll(s, "\n", " ")
	const maxLen = 72
	if len(s) > maxLen {
		s = s[:maxLen-3] + "..."
	}
	return s
}

func formatTemplateScriptFull(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "-"
	}
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	return strings.TrimRight(s, "\n")
}

// createFromTemplates creates a new box applying one or more templates' startup scripts.
// name and templateRefs are already validated by helper.ParseCreateArgs.
func createFromTemplates(name string, templateRefs []string, fromSnapshot string) {
	pubKey, err := readPublicKey()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	volumeSizeGB := service.DefaultVolumeSizeGB
	instanceType, err := helper.SelectInstanceType(service.AllInstanceTypes())
	if err != nil {
		fmt.Fprintf(os.Stderr, "error selecting instance type: %v\n", err)
		os.Exit(1)
	}
	if err := service.ValidateInstanceType(instanceType); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if fromSnapshot == "" {
		selectedVolume, err := helper.SelectVolumeSizeGB()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error selecting volume size: %v\n", err)
			os.Exit(1)
		}
		if err := service.ValidateVolumeSizeGB(selectedVolume); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		volumeSizeGB = selectedVolume
	}

	if fromSnapshot != "" {
		fmt.Printf("Creating box %q from templates %s (snapshot %s)...\n", name, strings.Join(templateRefs, ", "), fromSnapshot)
	} else {
		fmt.Printf("Creating box %q from templates %s...\n", name, strings.Join(templateRefs, ", "))
	}
	fmt.Println("Note: older startup scripts may not fully install — after SSH, verify your tools/libraries are present.")

	var b Box

	rt := helper.MustOpenRuntime()
	defer func() { _ = rt.Close() }()
	if fromSnapshot != "" {
		snapshotTarget, err := helper.ResolveSnapshotTarget(rt, fromSnapshot) // resolve the snapshot target from the ami id or name
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		fromSnapshot = snapshotTarget.AmiID
	}
	inst, err := rt.CreateBoxFromTemplates(name, templateRefs, pubKey, fromSnapshot, service.LocalUserID, instanceType, volumeSizeGB)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	b = instancesToBoxes([]*service.Instance{inst})[0]

	fmt.Printf("Box created.\n")
	fmt.Printf("  ID:        %s\n", b.ID)
	fmt.Printf("  Name:      %s\n", b.Name)
	fmt.Printf("  Status:    %s\n", b.Status)
	if b.InstanceType != "" {
		fmt.Printf("  Type:      %s\n", b.InstanceType)
	}
	if fromSnapshot == "" {
		fmt.Printf("  Storage:   %d GB\n", volumeSizeGB)
	}
	addSSHHostOrWarn(b.Name, inst)
	if b.PublicIP != "" {
		fmt.Printf("  Public IP: %s\n", b.PublicIP)
		fmt.Printf("\n  Connect:   outpost ssh %s\n", b.Name)
	} else {
		fmt.Printf("\n  Provisioning — check status: outpost status %s\n", b.Name)
	}
}
