//nolint:goconst // CLI flag strings are intentionally repeated per-command; they may diverge independently
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

func newGenerateEntryCommand(_ *config.Values) *cli.Command {
	return &cli.Command{
		Name:    "generate-entry",
		Aliases: []string{"ge"},
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "delete",
				Aliases: []string{"d"},
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
		Action: generateEntryAction,
	}
}

func generateEntryAction(ctx context.Context, cmd *cli.Command) error {
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
		Delete:    cmd.Bool("delete"),
		Root:      cmd.Bool("root"),
		OtterPath: otterPath,
	}
	if cmd.Bool("all") {
		opts.All = true
	} else {
		names, err := splitNames(cmd.Args().Slice())
		if err != nil {
			return err
		}
		opts.ContainerNames = names
		opts.Icon = cmd.String("icon")
	}

	if err := commands.NewGenerateEntryCommand(commands.NewListCommand(containerManager), containerManager).Execute(ctx, opts); err != nil {
		return fmt.Errorf("failed to generate entry: %w", err)
	}
	return nil
}
