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
		Aliases: []string{"sync"},
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
			return upgradeAction(ctx, cmd)
		},
	}
}

func upgradeAction(ctx context.Context, cmd *cli.Command) error {
	containerManager, ok := ctx.Value(containerManagerKey).(containermanager.ContainerManager)
	if !ok {
		return errors.New("container manager not found in context")
	}

	options := &commands.UpgradeOptions{
		ContainerNames: splitNames(cmd.Args().Slice()),
		All:            cmd.Bool("all"),
		Running:        cmd.Bool("running"),
	}

	err := commands.NewUpgradeCommand(containerManager).Execute(ctx, options)
	if errors.Is(err, commands.ErrEmptyContainerList) {
		ui.DefaultLogger.Warn("No containers found.")
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to upgrade containers: %w", err)
	}
	return nil
}
