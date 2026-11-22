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
		os.Exit(1)
	}
}

func run() error {
	if os.Getuid() == 0 {
		ui.DefaultLogger.Error("running as root is highly unsupported; use the per-command --root flag on supported commands")
	}

	cfg, err := config.LoadValues()
	if err != nil {
		// The CLI's exit handler is not installed yet at this point, so
		// report the failure here or it would be completely silent.
		ui.DefaultLogger.Error("failed to load configuration", "error", err)
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
