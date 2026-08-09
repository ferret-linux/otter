package commands

import (
	"context"
	"fmt"

	"github.com/ferret-linux/otter/pkg/containermanager"
	"github.com/ferret-linux/otter/pkg/ui"
)

type RestartOptions struct {
	ContainerNames []string
	All            bool
	Force          bool
}

type RestartCommand struct {
	containerManager containermanager.ContainerManager
	listCmd          *ListCommand
}

func NewRestartCommand(cm containermanager.ContainerManager) *RestartCommand {
	return &RestartCommand{
		containerManager: cm,
		listCmd:          NewListCommand(cm),
	}
}

func (c *RestartCommand) Execute(ctx context.Context, opts *RestartOptions) error {
	containerNames, err := resolveContainerNames(ctx, c.listCmd, opts.ContainerNames, opts.All)
	if err != nil {
		return err
	}

	outcome := runBatch(ctx, containerNames, func(ctx context.Context, name string) (bool, error) {
		if err := c.containerManager.Stop(ctx, []string{name}, opts.Force); err != nil {
			err = fmt.Errorf("failed to restart (stop) '%s': %w", name, err)
			ui.DefaultLogger.Error(err)
			return false, err
		}
		if err := c.containerManager.Start(ctx, name); err != nil {
			err = fmt.Errorf("failed to restart (start) '%s': %w", name, err)
			ui.DefaultLogger.Error(err)
			return false, err
		}
		ui.DefaultLogger.Info("restarted", "name", name)
		return false, nil
	})

	return summarizeBatch(outcome, batchSummaryConfig{
		PastVerb: "restarted",
		BaseVerb: "restart",
	})
}
