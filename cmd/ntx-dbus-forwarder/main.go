// Package main is the entry point for the ntx-dbus-forwarder tool.
package main

import (
	"context"
	"fmt"
	"ntx/internal/dbusforwarder"
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

	cfg := dbusforwarder.LoadConfig()

	if err := dbusforwarder.Run(ctx, cfg); err != nil {
		fmt.Fprintln(os.Stderr, "ntx-dbus-forwarder:", err)

		return 1
	}

	return 0
}
