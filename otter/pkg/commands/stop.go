package commands

import (
	"context"
	"errors"
	"fmt"

	"github.com/ferret-linux/otter/pkg/containermanager"
	"github.com/ferret-linux/otter/pkg/ui"
)

type StopCommand struct {
	containerManager containermanager.ContainerManager
	listCmd          *ListCommand
}

type StopOptions struct {
	ContainerNames []string
	All            bool
	Force          bool
}

var ErrNoContainersFound = errors.New("no containers found")

func NewStopCommand(cm containermanager.ContainerManager) *StopCommand {
	return &StopCommand{
		containerManager: cm,
		listCmd:          NewListCommand(cm),
	}
}

func (c *StopCommand) Execute(ctx context.Context, opts *StopOptions) error {
	containerNames, err := resolveContainerNames(ctx, c.listCmd, opts.ContainerNames, opts.All)
	if err != nil {
		return err
	}

	outcome := runBatch(ctx, containerNames, func(ctx context.Context, name string) (bool, error) {
		if _, err := c.containerManager.InspectContainer(ctx, name); err != nil {
			err = fmt.Errorf("container '%s' not found", name)
			ui.DefaultLogger.Error(err)
			return false, err
		}
		if err := c.containerManager.Stop(ctx, []string{name}, opts.Force); err != nil {
			err = fmt.Errorf("failed to stop '%s': %w", name, err)
			ui.DefaultLogger.Error(err)
			return false, err
		}
		ui.DefaultLogger.Info("stopped", "name", name)
		return false, nil
	})

	return summarizeBatch(outcome, batchSummaryConfig{
		PastVerb: "stopped",
		BaseVerb: "stop",
	})
}
