package cli

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/urfave/cli/v3"

	"github.com/ferret-linux/otter/pkg/commands"
	"github.com/ferret-linux/otter/pkg/config"
	"github.com/ferret-linux/otter/pkg/containermanager"
	"github.com/ferret-linux/otter/pkg/ui"
)

func newRmCommand(cfg *config.Values) *cli.Command {
	return &cli.Command{
		Name:    "remove",
		Aliases: []string{"rm"},
		Flags: []cli.Flag{
			&cli.StringSliceFlag{
				Name:    "name",
				Aliases: []string{"n"},
			},
			&cli.BoolFlag{
				Name:    "all",
				Aliases: []string{"a"},
			},
			&cli.BoolFlag{
				Name:    "force",
				Aliases: []string{"f"},
			},
			&cli.BoolFlag{
				Name:    "yes",
				Aliases: []string{"y"},
			},
			&cli.BoolFlag{
				Name: "rm-home",
			},
			&cli.BoolFlag{
				Name: "bypass-lock",
			},
		},

		Action: func(ctx context.Context, cmd *cli.Command) error {
			return rmAction(ctx, cmd)
		},
	}
}

func rmAction(ctx context.Context, cmd *cli.Command) error {
	containerManager, ok := ctx.Value(containerManagerKey).(containermanager.ContainerManager)
	if !ok {
		return errors.New("container manager not found in context")
	}

	options := commands.RmOptions{
		NoTTY:          cmd.Bool("yes"),
		Force:          cmd.Bool("force"),
		BypassLock:     cmd.Bool("bypass-lock"),
		All:            cmd.Bool("all"),
		RemoveHome:     cmd.Bool("rm-home"),
		Root:           cmd.Bool("root"),
		ContainerNames: cmd.StringSlice("name"),
	}

	prompter := ui.NewPrompter(*bufio.NewReader(os.Stdin), os.Stdout)

	if _, err := commands.NewRmCommand(containerManager, prompter).Execute(ctx, options); err != nil {
		return fmt.Errorf("failed to execute rm command: %w", err)
	}
	return nil
}
