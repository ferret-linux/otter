//nolint:goconst // CLI flag strings are intentionally repeated per-command; they may diverge independently
package cli

import (
	"context"

	"github.com/urfave/cli/v3"

	"github.com/ferret-linux/otter/pkg/commands"
	"github.com/ferret-linux/otter/pkg/config"
	"github.com/ferret-linux/otter/pkg/containermanager"
)

func newUpgradeCommand(cfg *config.Values) *cli.Command {
	return &cli.Command{
		Name:    "upgrade",
		Aliases: []string{"syu"},
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "all",
				Aliases: []string{"a"},
			},
			&cli.BoolFlag{
				Name:    "running",
				Aliases: []string{"R"},
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return upgradeAction(ctx, cmd, cfg)
		},
	}
}

func upgradeAction(ctx context.Context, cmd *cli.Command, cfg *config.Values) error {
	return runContainerCommand(ctx, cmd, "failed to upgrade containers", func(cm containermanager.ContainerManager, names []string) error {
		return commands.NewUpgradeCommand(cfg, cm).Execute(ctx, &commands.UpgradeOptions{
			ContainerNames: names,
			All:            cmd.Bool("all"),
			Running:        cmd.Bool("running"),
		})
	})
}
