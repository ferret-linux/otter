package cli

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/urfave/cli/v3"

	"github.com/ferret-linux/otter/pkg/commands"
	"github.com/ferret-linux/otter/pkg/config"
	"github.com/ferret-linux/otter/pkg/containermanager"
	"github.com/ferret-linux/otter/pkg/ui"
)

func newUpgradeCommand(cfg *config.Values) *cli.Command {
	return &cli.Command{
		Name: "upgrade",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "all",
				Aliases: []string{"a"},
			},
			&cli.BoolFlag{
				Name: "running",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return upgradeAction(ctx, cmd, cfg)
		},
	}
}

func upgradeAction(ctx context.Context, cmd *cli.Command, cfg *config.Values) error {
	containerManager, ok := ctx.Value(containerManagerKey).(containermanager.ContainerManager)
	if !ok {
		return errors.New("container manager not found in context")
	}

	options := &commands.UpgradeOptions{
		ContainerNames: cmd.Args().Slice(),
		All:            cmd.Bool("all"),
		Running:        cmd.Bool("running"),
		DryRun:         cmd.Bool("dry-run"),
		Verbose:        cmd.Bool("verbose"),
	}

	progress := ui.NewProgress(os.Stderr)

	upgradeCmd := commands.NewUpgradeCommand(cfg, containerManager, progress)

	err := upgradeCmd.Execute(ctx, options)

	if errors.Is(err, commands.ErrEmptyContainerList) {
		ui.DefaultLogger.Warn("No containers found.")
		return nil
	}

	if errors.Is(err, commands.ErrUpgradeNoContainerSpecified) {
		ui.DefaultLogger.Warn("Please specify the name of the container.")
		//nolint:wrapcheck // sentinel returned as-is so caller exits non-zero
		return err
	}

	if err != nil {
		return fmt.Errorf("failed to upgrade containers: %w", err)
	}

	return nil
}
