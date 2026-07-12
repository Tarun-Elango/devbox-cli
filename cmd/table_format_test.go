package cmd

import (
	"strings"
	"testing"
)

func TestFitColumnWidthsFitsTerminal(t *testing.T) {
	preferred := []int{24, 18, 10, 14, 8, 14, 16}
	min := []int{12, 8, 6, 8, 3, 8, 7}
	widths := fitColumnWidths(preferred, min, 80, len(tableColGap))
	if got := tableRowWidth(widths); got > 80 {
		t.Fatalf("row width %d exceeds terminal 80 (widths=%v)", got, widths)
	}
	for i, w := range widths {
		if w < min[i] {
			t.Fatalf("widths[%d]=%d below min %d", i, w, min[i])
		}
		if w > preferred[i] {
			t.Fatalf("widths[%d]=%d above preferred %d", i, w, preferred[i])
		}
	}
}

func TestFitColumnWidthsKeepsPreferredWhenWide(t *testing.T) {
	preferred := []int{24, 18, 10}
	min := []int{12, 8, 6}
	widths := fitColumnWidths(preferred, min, 200, 2)
	for i := range preferred {
		if widths[i] != preferred[i] {
			t.Fatalf("widths[%d]=%d, want preferred %d", i, widths[i], preferred[i])
		}
	}
}

func TestFitColumnWidthsStopsAtMin(t *testing.T) {
	preferred := []int{20, 20}
	min := []int{5, 5}
	widths := fitColumnWidths(preferred, min, 5, 2)
	if widths[0] != 5 || widths[1] != 5 {
		t.Fatalf("got %v, want mins [5 5]", widths)
	}
}

func TestFormatTableRowTruncates(t *testing.T) {
	row := formatTableRow([]string{"abcdefghij", "ok"}, []int{6, 4})
	if !strings.Contains(row, "...") {
		t.Fatalf("expected truncation, got %q", row)
	}
	if len(row) != 6+len(tableColGap)+4 {
		t.Fatalf("row length %d, want %d (%q)", len(row), 6+len(tableColGap)+4, row)
	}
}

func TestPrintBoxTableFitsWidth(t *testing.T) {
	boxes := []Box{
		{
			ID:       "i-0abcdefghijklmnopqrstuvwxyz",
			Name:     "very-long-box-name-here",
			Status:   "running",
			OSFamily: "ubuntu",
			Provider: "aws",
			Region:   "ap-southeast-2",
			PublicIP: "203.0.113.10",
		},
	}
	const width = 80
	out := captureStdout(t, func() {
		printBoxTable(boxes, width)
	})
	for i, line := range strings.Split(strings.TrimRight(out, "\n"), "\n") {
		if len(line) > width {
			t.Fatalf("line %d length %d exceeds %d: %q", i, len(line), width, line)
		}
	}
	if !strings.Contains(out, "ID") || !strings.Contains(out, "NAME") {
		t.Fatalf("missing headers: %q", out)
	}
}
