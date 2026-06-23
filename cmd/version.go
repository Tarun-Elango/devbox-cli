package cmd

import (
	"fmt"

	"devbox-cli/internal/version"
)

func Version(args []string) {
	fmt.Println(version.String())
}
