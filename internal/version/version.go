package version

import (
	_ "embed"
	"strings"
)

//go:embed VERSION
var raw string // holds the version string from neighboring VERSION file

// String returns the application version baked in at build time.
func String() string {
	return strings.TrimSpace(raw)
}
