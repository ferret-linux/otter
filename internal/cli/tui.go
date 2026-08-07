package cli

import (
	"context"
	"errors"
	"fmt"

	"github.com/urfave/cli/v3"

	"github.com/ferret-linux/otter/internal/tui"
	"github.com/ferret-linux/otter/pkg/config"
	"github.com/ferret-linux/otter/pkg/containermanager"
)

func newTUICommand(_ *config.Values) *cli.Command {
	return &cli.Command{
		Name:   "tui",
		Action: tuiAction,
	}
}

func tuiAction(ctx context.Context, _ *cli.Command) error {
	containerManager, ok := ctx.Value(containerManagerKey).(containermanager.ContainerManager)
	if !ok {
		return errors.New("container manager not found in context")
	}

	if err := tui.Run(ctx, containerManager); err != nil {
		return fmt.Errorf("failed to run tui: %w", err)
	}
	return nil
}
