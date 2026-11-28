package cli

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/urfave/cli/v3"

	"github.com/ferret-linux/otter/pkg/commands"
	"github.com/ferret-linux/otter/pkg/config"
	"github.com/ferret-linux/otter/pkg/containermanager"
)

func newEnterCommand(_ *config.Values) *cli.Command {
	return &cli.Command{
		Name:                      "enter",
		Aliases:                   []string{"sh"},
		Usage:                     "Enter running containers",
		DisableSliceFlagSeparator: true,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "clean-path",
				Aliases: []string{"c"},
				Usage:   "Reset PATH to FHS standard",
			},
			&cli.StringSliceFlag{
				Name:    "additional-flags",
				Aliases: []string{"a"},
				Usage:   "Extra flags for manager",
			},
			&cli.BoolFlag{
				Name:    "background",
				Aliases: []string{"b"},
				Usage:   "Run in background without tty",
			},
			&cli.BoolFlag{
				Name:    "no-workdir",
				Aliases: []string{"nw"},
				Usage:   "Start from container home",
			},
			&cli.StringSliceFlag{
				Name:    "add-env",
				Aliases: []string{"e"},
				Usage:   "Copy NAME or set NAME=value",
			},
			&cli.BoolFlag{
				Name:    "empty-env",
				Aliases: []string{"E"},
				Usage:   "Do not copy host environment",
			},
		},
		Action: enterAction,
	}
}

func enterAction(ctx context.Context, cmd *cli.Command) error {
	containerManager, ok := ctx.Value(containerManagerKey).(containermanager.ContainerManager)
	if !ok {
		return errors.New("container manager not found in context")
	}

	args := cmd.Args().Slice()
	if len(args) == 0 || strings.TrimSpace(args[0]) == "" {
		return errors.New("container name is required")
	}
	options := commands.EnterOptions{
		ContainerName:   args[0],
		AdditionalFlags: cmd.StringSlice("additional-flags"),
		CustomCommand:   cmd.Args().Tail(),
		AddEnv:          cmd.StringSlice("add-env"),
		Background:      cmd.Bool("background"),
		CleanPath:       cmd.Bool("clean-path"),
		EmptyEnv:        cmd.Bool("empty-env"),
		NoWorkDir:       cmd.Bool("no-workdir"),
	}

	_, err := commands.NewEnterCommand(containerManager).Execute(ctx, options)
	if err != nil {
		return fmt.Errorf("failed to execute enter command: %w", err)
	}
	return nil
}
