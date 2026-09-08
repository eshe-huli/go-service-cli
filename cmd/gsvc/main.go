// Command gsvc scaffolds and checks convention-driven Go services.
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"gsvc.local/cli/internal/cli"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	os.Exit(cli.Run(ctx, os.Args[1:], os.Stdout, os.Stderr))
}
