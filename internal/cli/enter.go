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

func newEnterCommand(cfg *config.Values) *cli.Command {
	return &cli.Command{
		Name:    "enter",
		Aliases: []string{"sh"},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "name",
				Aliases: []string{"n"},
			},
			&cli.BoolFlag{
				Name:    "clean-path",
				Aliases: []string{"c"},
			},
			&cli.StringFlag{
				Name:    "additional-flags",
				Aliases: []string{"a"},
			},
			&cli.BoolFlag{
				Name:    "no-tty",
				Aliases: []string{"T", "H"},
			},
			&cli.BoolFlag{
				Name:    "no-workdir",
				Aliases: []string{"nw"},
			},
			&cli.StringSliceFlag{
				Name: "add-env",
			},
			&cli.BoolFlag{
				Name: "empty-env",
			},
			&cli.BoolFlag{
				Name:    "auto-start",
				Aliases: []string{"S"},
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return enterAction(ctx, cmd)
		},
	}
}

func enterAction(ctx context.Context, cmd *cli.Command) error {
	containerManager, ok := ctx.Value(containerManagerKey).(containermanager.ContainerManager)
	if !ok {
		return errors.New("container manager not found in context")
	}

	if strings.Contains(cmd.String("name"), ",") {
		return errors.New("enter only accepts a single container name")
	}

	options := commands.EnterOptions{
		ContainerName:   cmd.String("name"),
		AdditionalFlags: cmd.String("additional-flags"),
		CustomCommand:   cmd.Args().Slice(),
		AddEnv:          cmd.StringSlice("add-env"),
		DryRun:          cmd.Bool("dry-run"),
		NoTTY:           cmd.Bool("no-tty"),
		CleanPath:       cmd.Bool("clean-path"),
		EmptyEnv:        cmd.Bool("empty-env"),
		AutoStart:       cmd.Bool("auto-start"),
		Verbose:         cmd.Bool("verbose"),
	}

	_, err := commands.NewEnterCommand(containerManager).Execute(ctx, options)
	if err != nil {
		return fmt.Errorf("failed to execute enter command: %w", err)
	}
	return nil
}
