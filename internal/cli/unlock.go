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

	names, err := splitNames(cmd.Args().Slice())
	if err != nil {
		return err
	}
	if err := commands.NewUnlockCommand(cm).Execute(ctx, commands.UnlockOptions{
		ContainerNames: names,
		All:            cmd.Bool("all"),
	}); err != nil {
		return fmt.Errorf("failed to unlock container: %w", err)
	}
	return nil
}
