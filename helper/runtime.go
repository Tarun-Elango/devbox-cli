package helper

import (
	"context"
	"fmt"
	"os"
	"time"

	"outpost-cli/service"
)

const defaultCommandTimeout = 5 * time.Minute

// CommandContext returns a context bounded to the lifetime of a CLI command.
func CommandContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), defaultCommandTimeout)
}

// MustOpenRuntime opens the runtime and exits if it fails.
func MustOpenRuntime() *service.Runtime {
	ctx, cancel := CommandContext()
	rt, err := service.Open(ctx, cancel)
	if err != nil {
		cancel()
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	return rt
}
