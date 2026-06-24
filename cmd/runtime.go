package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"devbox-cli/service"
)

const defaultCommandTimeout = 5 * time.Minute

// CommandContext returns a context bounded to the lifetime of a CLI command.
// timer for the whole command
func CommandContext() context.Context {
	ctx, _ := context.WithTimeout(context.Background(), defaultCommandTimeout)
	return ctx
}

// shared helper for cmds to open the runtime, and call the service functions using rt.function()

// mustOpenRuntime opens the runtime and panics if it fails
func mustOpenRuntime() *service.Runtime {
	rt, err := service.Open(CommandContext())
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	return rt
}
