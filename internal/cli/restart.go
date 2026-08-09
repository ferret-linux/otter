//nolint:goconst // CLI flag strings are intentionally repeated per-command; they may diverge independently
package cli

import (
	"context"

	"github.com/urfave/cli/v3"

	"github.com/ferret-linux/otter/pkg/commands"
	"github.com/ferret-linux/otter/pkg/config"
	"github.com/ferret-linux/otter/pkg/containermanager"
)

func newRestartCommand(_ *config.Values) *cli.Command {
	return &cli.Command{
		Name:    "restart",
		Aliases: []string{"rbt"},
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "all",
				Aliases: []string{"a"},
			},
			&cli.BoolFlag{
				Name:    "force",
				Aliases: []string{"f"},
			},
		},
		Action: restartAction,
	}
}

func restartAction(ctx context.Context, cmd *cli.Command) error {
	return runContainerCommand(ctx, cmd, "failed to restart containers", func(cm containermanager.ContainerManager, names []string) error {
		return commands.NewRestartCommand(cm).Execute(ctx, &commands.RestartOptions{
			ContainerNames: names,
			All:            cmd.Bool("all"),
			Force:          cmd.Bool("force"),
		})
	})
}
