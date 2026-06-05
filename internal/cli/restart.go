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

func newRestartCommand(cfg *config.Values) *cli.Command {
	return &cli.Command{
		Name:    "restart",
		Aliases: []string{"reboot"},
		Flags: []cli.Flag{
			&cli.StringSliceFlag{
				Name:    "name",
				Aliases: []string{"n"},
			},
			&cli.BoolFlag{
				Name:    "all",
				Aliases: []string{"a"},
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return restartAction(ctx, cmd)
		},
	}
}

func restartAction(ctx context.Context, cmd *cli.Command) error {
	containerManager, ok := ctx.Value(containerManagerKey).(containermanager.ContainerManager)
	if !ok {
		return errors.New("container manager not found in context")
	}

	options := &commands.RestartOptions{
		ContainerNames: cmd.StringSlice("name"),
		All:            cmd.Bool("all"),
	}

	err := commands.NewRestartCommand(containerManager).Execute(ctx, options)
	if errors.Is(err, commands.ErrEmptyContainerList) {
		ui.DefaultLogger.Warn("No containers found.")
		return nil
	}
	return err
}
