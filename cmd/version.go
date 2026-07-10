package cmd

import (
	"fmt"

	"outpost-cli/helper"
	"outpost-cli/internal/version"
)

func Version(args []string) {
	helper.RejectExtraArgs(args, "usage: outpost version")
	fmt.Println(version.String())
}
