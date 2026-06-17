package cmd

import (
	"context"
	"fmt"
	"os"

	"devbox-cli/service"
)

// shared helper for cmds to open the runtime, and call the service functions using rt.function()

// mustOpenRuntime opens the runtime and panics if it fails
func mustOpenRuntime() *service.Runtime {
	rt, err := service.Open(context.Background())
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	return rt
}
