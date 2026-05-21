package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/ferret-linux/otter/internal/cli"
	"github.com/ferret-linux/otter/pkg/config"
	"github.com/ferret-linux/otter/pkg/ui"
)

func main() {
	if err := run(); err != nil {
		ui.DefaultLogger.Error("%s", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.LoadValues()
	if err != nil {
		//nolint:wrapcheck // main reports errors as-is
		return err
	}

	// SIGINT register
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cmd := cli.NewRootCommand(cfg)

	//nolint:wrapcheck // main reports errors as-is
	return cmd.Run(ctx, os.Args)
}
