//nolint:goconst // CLI flag strings are intentionally repeated per-command; they may diverge independently
package cli

import (
	"context"

	"github.com/urfave/cli/v3"

	"github.com/ferret-linux/otter/pkg/commands"
	"github.com/ferret-linux/otter/pkg/config"
	"github.com/ferret-linux/otter/pkg/containermanager"
)

func newLockCommand(_ *config.Values) *cli.Command {
	return &cli.Command{
		Name:    "lock",
		Aliases: []string{"lk"},
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "all",
				Aliases: []string{"a"},
			},
		},
		Action: lockAction,
	}
}

func lockAction(ctx context.Context, cmd *cli.Command) error {
	return runContainerCommand(ctx, cmd, "failed to lock container", func(cm containermanager.ContainerManager, names []string) error {
		return commands.NewLockCommand(cm).Execute(ctx, commands.LockOptions{
			ContainerNames: names,
			All:            cmd.Bool("all"),
		})
	})
}
