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
				Aliases: []string{"Y"},
			},
			&cli.BoolFlag{
				Name: "rm-home",
			},
			&cli.BoolFlag{
				Name: "bypass-lock",
			},
		},

		Action: func(ctx context.Context, cmd *cli.Command) error {
			return rmAction(ctx, cmd, cfg)
		},
	}
}

func rmAction(ctx context.Context, cmd *cli.Command, cfg *config.Values) error {
	containerManager, ok := ctx.Value(containerManagerKey).(containermanager.ContainerManager)
	if !ok {
		return errors.New("container manager not found in context")
	}

	names := cmd.Args().Slice()
	if len(names) == 0 && !cmd.Bool("all") {
		names = []string{cfg.DefaultContainerName}
	}

	options := commands.RmOptions{
		NoTTY:          cmd.Bool("yes"),
		Force:          cmd.Bool("force"),
		BypassLock:     cmd.Bool("bypass-lock"),
		All:            cmd.Bool("all"),
		RemoveHome:     cmd.Bool("rm-home"),
		DryRun:         cmd.Bool("dry-run"),
		Verbose:        cmd.Bool("verbose"),
		Root:           cmd.Bool("root"),
		ContainerNames: names,
	}

	prompter := ui.NewPrompter(*bufio.NewReader(os.Stdin), os.Stdout)

	rmCmd := commands.NewRmCommand(cfg, containerManager, prompter)
	_, err := rmCmd.Execute(ctx, options)
	if err != nil {
		return fmt.Errorf("failed to execute rm command: %w", err)
	}

	return nil
}
