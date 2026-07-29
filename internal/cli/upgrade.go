//nolint:goconst // CLI flag strings are intentionally repeated per-command; they may diverge independently
package cli

import (
	"context"
	"errors"
	"fmt"

	"github.com/urfave/cli/v3"

	"github.com/ferret-linux/otter/pkg/commands"
	"github.com/ferret-linux/otter/pkg/config"
	"github.com/ferret-linux/otter/pkg/containermanager"
	"github.com/ferret-linux/otter/pkg/ui"
)

func newUpgradeCommand(cfg *config.Values) *cli.Command {
	return &cli.Command{
		Name:    "upgrade",
		Aliases: []string{"syu"},
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "all",
				Aliases: []string{"a"},
			},
			&cli.BoolFlag{
				Name:    "running",
				Aliases: []string{"R"},
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

	names, err := splitNames(cmd.Args().Slice())
	if err != nil {
		return err
	}
	options := &commands.UpgradeOptions{
		ContainerNames: names,
		All:            cmd.Bool("all"),
		Running:        cmd.Bool("running"),
	}

	err = commands.NewUpgradeCommand(cfg, containerManager).Execute(ctx, options)
	if errors.Is(err, commands.ErrEmptyContainerList) {
		ui.DefaultLogger.Warn("No containers found.")
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to upgrade containers: %w", err)
	}
	return nil
}
