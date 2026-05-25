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

func newEnterCommand(cfg *config.Values) *cli.Command {
	return &cli.Command{
		Name: "enter",
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
				Name:    "yes",
				Aliases: []string{"y"},
			},
			&cli.BoolFlag{
				Name:    "no-tty",
				Aliases: []string{"T", "H"},
			},
			&cli.BoolFlag{
				Name:    "no-workdir",
				Aliases: []string{"nw"},
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return enterAction(ctx, cmd, cfg)
		},
	}
}

func enterAction(ctx context.Context, cmd *cli.Command, cfg *config.Values) error {
	containerManager, ok := ctx.Value(containerManagerKey).(containermanager.ContainerManager)
	if !ok {
		return errors.New("container manager not found in context")
	}

	// Container name: --name flag takes priority, otherwise first positional arg.
	// Everything after the container name (or after --) is the custom command.
	containerName := cmd.String("name")

	args := cmd.Args().Slice()
	if containerName == "" && len(args) > 0 {
		containerName = args[0]
		args = args[1:]
	}

	if containerName == "" {
		containerName = cfg.DefaultContainerName
	}

	options := commands.EnterOptions{
		ContainerName:   containerName,
		AdditionalFlags: cmd.String("additional-flags"),
		CustomCommand:   args,
		DryRun:          cmd.Bool("dry-run"),
		NoTTY:           cmd.Bool("no-tty"),
		CleanPath:       cmd.Bool("clean-path"),
		Verbose:         cmd.Bool("verbose"),
	}

	enterCmd := commands.NewEnterCommand(cfg, containerManager)
	_, err := enterCmd.Execute(ctx, options)
	if err != nil {
		return fmt.Errorf("failed to execute enter command: %w", err)
	}

	return nil
}
