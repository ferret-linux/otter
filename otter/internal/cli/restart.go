//nolint:goconst,dupl // CLI flag/command names are declarative; restart/stop intentionally mirror each other
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
		Usage:   "Restart containers",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "all",
				Aliases: []string{"a"},
				Usage:   "Use all containers",
			},
			&cli.BoolFlag{
				Name:    "force",
				Aliases: []string{"f"},
				Usage:   "Force the operation",
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
