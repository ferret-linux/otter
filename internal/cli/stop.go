package cli

import (
	"context"
	"errors"

	"github.com/urfave/cli/v3"

	"github.com/ferret-linux/otter/pkg/commands"
	"github.com/ferret-linux/otter/pkg/config"
	"github.com/ferret-linux/otter/pkg/containermanager"
	"github.com/ferret-linux/otter/pkg/ui"
)

func newStopCommand(cfg *config.Values) *cli.Command {
	return &cli.Command{
		Name: "stop",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "all",
				Aliases: []string{"a"},
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return stopAction(ctx, cmd, cfg)
		},
	}
}

func stopAction(ctx context.Context, cmd *cli.Command, cfg *config.Values) error {
	containerManager, ok := ctx.Value(containerManagerKey).(containermanager.ContainerManager)
	if !ok {
		return errors.New("container manager not found in context")
	}

	options := &commands.StopOptions{
		ContainerNames: cmd.Args().Slice(),
		All:            cmd.Bool("all"),
		DryRun:         cmd.Bool("dry-run"),
		Verbose:        cmd.Bool("verbose"),
	}

	stopCmd := commands.NewStopCommand(cfg, containerManager)

	err := stopCmd.Execute(ctx, options)

	if errors.Is(err, commands.ErrEmptyContainerList) {
		ui.DefaultLogger.Warn("No containers found.")
		return nil
	}

	if err != nil {
		return err
	}

	return nil
}
