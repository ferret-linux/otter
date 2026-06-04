package cli

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"

	"github.com/ferret-linux/otter/pkg/commands"
	"github.com/ferret-linux/otter/pkg/config"
	"github.com/ferret-linux/otter/pkg/containermanager"
)

func newUnlockCommand(cfg *config.Values) *cli.Command {
	return &cli.Command{
		Name:    "unlock",
		Aliases: []string{"ulck"},
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
			return unlockAction(ctx, cmd)
		},
	}
}

func unlockAction(ctx context.Context, cmd *cli.Command) error {
	cm := ctx.Value(containerManagerKey).(containermanager.ContainerManager)

	if err := commands.NewUnlockCommand(cm).Execute(ctx, commands.UnlockOptions{
		ContainerNames: cmd.StringSlice("name"),
		All:            cmd.Bool("all"),
		Verbose:        cmd.Bool("verbose"),
		DryRun:         cmd.Bool("dry-run"),
	}); err != nil {
		return fmt.Errorf("failed to unlock container: %w", err)
	}
	return nil
}
