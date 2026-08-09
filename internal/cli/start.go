//nolint:goconst // CLI flag strings are intentionally repeated per-command; they may diverge independently
package cli

import (
	"context"

	"github.com/urfave/cli/v3"

	"github.com/ferret-linux/otter/pkg/commands"
	"github.com/ferret-linux/otter/pkg/config"
	"github.com/ferret-linux/otter/pkg/containermanager"
)

func newStartCommand(_ *config.Values) *cli.Command {
	return &cli.Command{
		Name:    "start",
		Aliases: []string{"up"},
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "all",
				Aliases: []string{"a"},
			},
		},
		Action: startAction,
	}
}

func startAction(ctx context.Context, cmd *cli.Command) error {
	return runContainerCommand(ctx, cmd, "failed to start containers", func(cm containermanager.ContainerManager, names []string) error {
		return commands.NewStartCommand(cm).Execute(ctx, &commands.StartOptions{
			ContainerNames: names,
			All:            cmd.Bool("all"),
		})
	})
}
