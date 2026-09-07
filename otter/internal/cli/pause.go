//nolint:goconst // CLI flag strings are intentionally repeated per-command; they may diverge independently
package cli

import (
	"context"

	"github.com/urfave/cli/v3"

	"github.com/ferret-linux/otter/pkg/commands"
	"github.com/ferret-linux/otter/pkg/config"
	"github.com/ferret-linux/otter/pkg/containermanager"
)

func newPauseCommand(_ *config.Values) *cli.Command {
	return &cli.Command{
		Name:    "pause",
		Aliases: []string{"zz"},
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "all",
				Aliases: []string{"a"},
			},
		},
		Action: pauseAction,
	}
}

func pauseAction(ctx context.Context, cmd *cli.Command) error {
	return runContainerCommand(ctx, cmd, "failed to pause container", func(cm containermanager.ContainerManager, names []string) error {
		return commands.NewPauseCommand(cm).Execute(ctx, &commands.PauseOptions{
			ContainerNames: names,
			All:            cmd.Bool("all"),
		})
	})
}
