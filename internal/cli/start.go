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
	"github.com/ferret-linux/otter/pkg/ui"
)

func newStartCommand(_ *config.Values) *cli.Command {
	return &cli.Command{
		Name:    "start",
		Aliases: []string{"up"},
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "all",
				Aliases: []string{"a"},
			},
		},
		Action: startAction,
	}
}

func startAction(ctx context.Context, cmd *cli.Command) error {
	containerManager, ok := ctx.Value(containerManagerKey).(containermanager.ContainerManager)
	if !ok {
		return errors.New("container manager not found in context")
	}

	names, err := splitNames(cmd.Args().Slice())
	if err != nil {
		return err
	}
	err = commands.NewStartCommand(containerManager).Execute(ctx, &commands.StartOptions{
		ContainerNames: names,
		All:            cmd.Bool("all"),
	})
	if errors.Is(err, commands.ErrNoContainersFound) {
		ui.DefaultLogger.Warn("No containers found.")
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to start containers: %w", err)
	}
	return nil
}
