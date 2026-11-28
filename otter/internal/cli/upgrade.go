//nolint:goconst // CLI flag/command names are declarative and intentionally repeated with their aliases/usage
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
		Usage:   "Upgrade packages in containers",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "all",
				Aliases: []string{"a"},
				Usage:   "Use all containers",
			},
			&cli.BoolFlag{
				Name:    "running",
				Aliases: []string{"R"},
				Usage:   "Only running containers",
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
