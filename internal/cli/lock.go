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
		Name:      "lock",
		Aliases:   []string{"lck"},
		ArgsUsage: "CONTAINER",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return lockAction(ctx, cmd, cfg)
		},
	}
}

func lockAction(ctx context.Context, cmd *cli.Command, cfg *config.Values) error {
	cm := ctx.Value(containerManagerKey).(containermanager.ContainerManager)

	if cmd.NArg() == 0 {
		return fmt.Errorf("please specify the name of the container")
	}

	lockCmd := commands.NewLockCommand(cfg, cm)
	if err := lockCmd.Execute(ctx, commands.LockOptions{
		ContainerName: cmd.Args().First(),
		Verbose:       cmd.Bool("verbose"),
		DryRun:        cmd.Bool("dry-run"),
	}); err != nil {
		return fmt.Errorf("failed to lock container: %w", err)
	}

	return nil
}
