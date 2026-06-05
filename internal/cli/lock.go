package cli

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"

	"github.com/ferret-linux/otter/pkg/commands"
	"github.com/ferret-linux/otter/pkg/config"
	"github.com/ferret-linux/otter/pkg/containermanager"
)

func newLockCommand(cfg *config.Values) *cli.Command {
	return &cli.Command{
		Name:    "lock",
		Aliases: []string{"lck"},
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
			return lockAction(ctx, cmd)
		},
	}
}

func lockAction(ctx context.Context, cmd *cli.Command) error {
	cm := ctx.Value(containerManagerKey).(containermanager.ContainerManager)

	if err := commands.NewLockCommand(cm).Execute(ctx, commands.LockOptions{
		ContainerNames: cmd.StringSlice("name"),
		All:            cmd.Bool("all"),
	}); err != nil {
		return fmt.Errorf("failed to lock container: %w", err)
	}
	return nil
}
