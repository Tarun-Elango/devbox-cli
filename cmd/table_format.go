package cmd

import (
	"fmt"
	"strings"
)

const tableColGap = "  "

// fitColumnWidths shrinks preferred widths so the row fits terminalWidth.
// Columns above their min shrink from the widest first; mins are never breached.
func fitColumnWidths(preferred, min []int, terminalWidth, gapLen int) []int {
	if len(preferred) != len(min) {
		panic("fitColumnWidths: preferred and min length mismatch")
	}
	widths := append([]int(nil), preferred...)
	if len(widths) == 0 {
		return widths
	}

	gaps := gapLen * (len(widths) - 1)
	sum := gaps
	for _, w := range widths {
		sum += w
	}

	for sum > terminalWidth {
		best := -1
		for i, w := range widths {
			if w <= min[i] {
				continue
			}
			if best < 0 || w > widths[best] || (w == widths[best] && i > best) {
				best = i
			}
		}
		if best < 0 {
			break
		}
		widths[best]--
		sum--
	}
	return widths
}

func formatTableRow(cells []string, widths []int) string {
	parts := make([]string, len(cells))
	for i, c := range cells {
		parts[i] = truncatePad(c, widths[i])
	}
	return strings.Join(parts, tableColGap)
}

func tableRowWidth(widths []int) int {
	if len(widths) == 0 {
		return 0
	}
	n := len(tableColGap) * (len(widths) - 1)
	for _, w := range widths {
		n += w
	}
	return n
}

func printTable(headers []string, rows [][]string, preferred, min []int, termWidth int) {
	widths := fitColumnWidths(preferred, min, termWidth, len(tableColGap))
	fmt.Println(formatTableRow(headers, widths))
	fmt.Println(strings.Repeat("-", tableRowWidth(widths)))
	for _, row := range rows {
		fmt.Println(formatTableRow(row, widths))
	}
}
