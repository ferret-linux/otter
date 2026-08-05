package commands

import (
	"context"
	"errors"
	"fmt"

	"github.com/ferret-linux/otter/pkg/containermanager"
	"github.com/ferret-linux/otter/pkg/ui"
)

type PauseOptions struct {
	ContainerNames []string
	All            bool
}

type PauseCommand struct {
	containerManager containermanager.ContainerManager
	listCmd          *ListCommand
}

func NewPauseCommand(cm containermanager.ContainerManager) *PauseCommand {
	return &PauseCommand{
		containerManager: cm,
		listCmd:          NewListCommand(cm),
	}
}

func (c *PauseCommand) Execute(ctx context.Context, opts *PauseOptions) error {
	var containerNames []string
	skipNotRunning := map[string]bool{}

	switch {
	case opts.All:
		containers, err := c.listCmd.Execute(ctx, ListOptions{})
		if err != nil {
			return fmt.Errorf("failed to list containers: %w", err)
		}
		if len(containers.Containers) == 0 {
			return ErrNoContainersFound
		}
		for _, container := range containers.Containers {
			containerNames = append(containerNames, container.Name)
			if !container.IsRunning() {
				skipNotRunning[container.Name] = true
			}
		}
	case len(opts.ContainerNames) > 0:
		containerNames = opts.ContainerNames
	default:
		return errors.New("please specify a container name or use --all")
	}

	outcome := runBatch(ctx, containerNames, func(ctx context.Context, name string) (bool, error) {
		if skipNotRunning[name] {
			ui.DefaultLogger.Warn("'%s' is not running, skipping", name)
			return true, nil
		}
		if err := c.containerManager.Pause(ctx, name); err != nil {
			ui.DefaultLogger.Error("failed to pause '%s': %s", name, err)
			return false, err
		}
		ui.DefaultLogger.Ok("paused '%s'", name)
		return false, nil
	})

	return summarizeBatch(outcome, batchSummaryConfig{
		PastVerb:          "paused",
		BaseVerb:          "pause",
		AllSkippedMessage: fmt.Sprintf("all %d containers already stopped, nothing to do", len(containerNames)),
	})
}
