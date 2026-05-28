package cli

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/urfave/cli/v3"

	"github.com/ferret-linux/otter/pkg/commands"
	"github.com/ferret-linux/otter/pkg/config"
	"github.com/ferret-linux/otter/pkg/containermanager"
)

func newGenerateEntryCommand(cfg *config.Values) *cli.Command {
	return &cli.Command{
		Name:    "generate-entry",
		Aliases: []string{"gety"},
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "delete",
				Aliases: []string{"del"},
			},
			&cli.StringFlag{
				Name:    "icon",
				Aliases: []string{"i"},
				Value:   "auto",
			},
			&cli.BoolFlag{
				Name:    "all",
				Aliases: []string{"a"},
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return generateEntryAction(ctx, cmd, cfg)
		},
	}
}

func generateEntryAction(ctx context.Context, cmd *cli.Command, cfg *config.Values) error {
	// The current executable is used as otter path
	otterPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get otter executable path: %w", err)
	}

	containerManager, ok := ctx.Value(containerManagerKey).(containermanager.ContainerManager)
	if !ok {
		return errors.New("container manager not found in context")
	}

	opts := &commands.GenerateEntryOptions{
		Verbose:   cmd.Bool("verbose"),
		DryRun:    cmd.Bool("dry-run"),
		Delete:    cmd.Bool("delete"),
		Root:      cmd.Bool("root"),
		OtterPath: otterPath,
	}
	if cmd.Bool("all") {
		opts.All = true
	} else {
		opts.ContainerName = cmd.Args().First()
		opts.Icon = cmd.String("icon")
	}

	genEntryCmd := commands.NewGenerateEntryCommand(cfg, commands.NewListCommand(cfg, containerManager), containerManager)

	err = genEntryCmd.Execute(ctx, opts)
	if err != nil {
		return err
	}

	return nil
}
