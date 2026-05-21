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

func newStopCommand(cfg *config.Values) *cli.Command {
	return &cli.Command{
		Name: "stop",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "all",
				Aliases: []string{"a"},
			},
			&cli.BoolFlag{
				Name:    "yes",
				Aliases: []string{"Y"},
			},
			&cli.BoolFlag{
				Name:    "dry-run",
				Aliases: []string{"d"},
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return stopAction(ctx, cmd, cfg)
		},
	}
}

func stopAction(ctx context.Context, cmd *cli.Command, cfg *config.Values) error {
	containerManager, ok := ctx.Value(containerManagerKey).(containermanager.ContainerManager)
	if !ok {
		return errors.New("container manager not found in context")
	}

	all := cmd.Bool("all")
	nonInteractive := cmd.Bool("yes")
	containerNames := cmd.Args().Slice()

	options := &commands.StopOptions{
		ContainerNames: containerNames,
		NonInteractive: nonInteractive,
		All:            all,
		DryRun:         cmd.Bool("dry-run"),
	}

	prompter := ui.NewPrompter(*bufio.NewReader(os.Stdin), os.Stdout)

	stopCmd := commands.NewStopCommand(cfg, containerManager, prompter)

	err := stopCmd.Execute(ctx, options)

	if errors.Is(err, commands.ErrStopAbortedByUserError) {
		ui.DefaultLogger.Warn("Aborted.")
		return nil
	}

	if errors.Is(err, commands.ErrEmptyContainerList) {
		ui.DefaultLogger.Warn("No containers found.")
		return nil
	}

	if err != nil {
		return fmt.Errorf("failed to stop containers: %w", err)
	}

	return nil
}
