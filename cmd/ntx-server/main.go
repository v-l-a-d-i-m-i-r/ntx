// Package main is the entry point for the ntx-server service.
package main

import (
	"context"
	"fmt"
	"ntx/internal/server"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	os.Exit(run())
}

func run() int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg := server.LoadConfig()

	if err := server.Run(ctx, cfg); err != nil {
		fmt.Fprintln(os.Stderr, "ntx-server:", err)

		return 1
	}

	return 0
}
