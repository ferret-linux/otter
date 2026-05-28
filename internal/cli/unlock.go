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
		Name:      "unlock",
		Aliases:   []string{"ulck"},
		ArgsUsage: "CONTAINER",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return unlockAction(ctx, cmd, cfg)
		},
	}
}

func unlockAction(ctx context.Context, cmd *cli.Command, cfg *config.Values) error {
	cm := ctx.Value(containerManagerKey).(containermanager.ContainerManager)

	if cmd.NArg() == 0 {
		return fmt.Errorf("please specify the name of the container")
	}

	unlockCmd := commands.NewUnlockCommand(cfg, cm)
	if err := unlockCmd.Execute(ctx, commands.UnlockOptions{
		ContainerName: cmd.Args().First(),
		Verbose:       cmd.Bool("verbose"),
		DryRun:        cmd.Bool("dry-run"),
	}); err != nil {
		return fmt.Errorf("failed to unlock container: %w", err)
	}

	return nil
}
