package cmd

import (
	"fmt"

	"devbox-cli/helper"
	"devbox-cli/internal/version"
)

func Version(args []string) {
	helper.RejectExtraArgs(args, "usage: devbox version")
	fmt.Println(version.String())
}
