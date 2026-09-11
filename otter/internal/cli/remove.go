//nolint:goconst // CLI flag strings are intentionally repeated per-command; they may diverge independently
package cli

import (
	"context"
	"errors"
	"fmt"

	"github.com/urfave/cli/v3"

	"github.com/ferret-linux/otter/pkg/commands"
	"github.com/ferret-linux/otter/pkg/config"
	"github.com/ferret-linux/otter/pkg/containermanager"
)

func newRmCommand(_ *config.Values) *cli.Command {
	return &cli.Command{
		Name:    "remove",
		Aliases: []string{"rm"},
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "all",
				Aliases: []string{"a"},
				Usage:   "Delete all otter containers",
			},
			&cli.BoolFlag{
				Name:    "force",
				Aliases: []string{"f"},
				Usage:   "Force deletion of problematic containers",
			},
			&cli.BoolFlag{
				Name:    "rm-home",
				Aliases: []string{"H"},
				Usage:   "Remove container's custom home directory",
			},
			&cli.BoolFlag{
				Name:    "bypass-lock",
				Aliases: []string{"B"},
				Usage:   "Remove container even if it is locked",
			},
		},

		Action: rmAction,
	}
}

func rmAction(ctx context.Context, cmd *cli.Command) error {
	containerManager, ok := ctx.Value(containerManagerKey).(containermanager.ContainerManager)
	if !ok {
		return errors.New("container manager not found in context")
	}

	names, err := splitNames(cmd.Args().Slice())
	if err != nil {
		return err
	}
	options := commands.RmOptions{
		Force:          cmd.Bool("force"),
		BypassLock:     cmd.Bool("bypass-lock"),
		All:            cmd.Bool("all"),
		RemoveHome:     cmd.Bool("rm-home"),
		Root:           cmd.Bool("root"),
		ContainerNames: names,
	}

	if _, err := commands.NewRmCommand(containerManager).Execute(ctx, options); err != nil {
		return fmt.Errorf("failed to execute rm command: %w", err)
	}
	return nil
}
