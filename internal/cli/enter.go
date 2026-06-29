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

func newEnterCommand(_ *config.Values) *cli.Command {
	return &cli.Command{
		Name:    "enter",
		Aliases: []string{"sh"},
		Flags: []cli.Flag{
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
				Aliases: []string{"T"},
			},
			&cli.BoolFlag{
				Name:    "no-workdir",
				Aliases: []string{"nw"},
			},
			&cli.StringSliceFlag{
				Name:    "add-env",
				Aliases: []string{"e"},
			},
			&cli.BoolFlag{
				Name:    "empty-env",
				Aliases: []string{"E"},
			},
			&cli.BoolFlag{
				Name:    "auto-start",
				Aliases: []string{"S"},
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

	options := commands.EnterOptions{
		ContainerName:   firstName(cmd.Args().Slice()),
		AdditionalFlags: cmd.String("additional-flags"),
		CustomCommand:   cmd.Args().Tail(),
		AddEnv:          cmd.StringSlice("add-env"),
		NoTTY:           cmd.Bool("no-tty"),
		CleanPath:       cmd.Bool("clean-path"),
		EmptyEnv:        cmd.Bool("empty-env"),
		AutoStart:       cmd.Bool("auto-start"),
		NoWorkDir:       cmd.Bool("no-workdir"),
	}

	_, err := commands.NewEnterCommand(containerManager).Execute(ctx, options)
	if err != nil {
		return fmt.Errorf("failed to execute enter command: %w", err)
	}
	return nil
}
