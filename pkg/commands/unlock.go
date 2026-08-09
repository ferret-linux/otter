package commands

import (
	"context"
	"errors"
	"fmt"

	"github.com/ferret-linux/otter/pkg/containermanager"
	"github.com/ferret-linux/otter/pkg/ui"
)

var ErrNotLocked = errors.New("container is not locked")

type UnlockOptions struct {
	ContainerNames []string
	All            bool
}

type UnlockCommand struct {
	containerManager containermanager.ContainerManager
	listCmd          *ListCommand
}

func NewUnlockCommand(cm containermanager.ContainerManager) *UnlockCommand {
	return &UnlockCommand{
		containerManager: cm,
		listCmd:          NewListCommand(cm),
	}
}

//nolint:dupl // structurally mirrors LockCommand.Execute; the lock/unlock semantics genuinely differ
func (c *UnlockCommand) Execute(ctx context.Context, opts UnlockOptions) error {
	containerNames, err := resolveContainerNames(ctx, c.listCmd, opts.ContainerNames, opts.All)
	if err != nil {
		return err
	}

	outcome := runBatch(ctx, containerNames, func(ctx context.Context, name string) (bool, error) {
		if err := c.unlockOne(ctx, name); err != nil {
			if errors.Is(err, ErrNotLocked) {
				ui.DefaultLogger.Warn("already unlocked, skipping", "name", name)
				return true, nil
			}
			ui.DefaultLogger.Error("failed to unlock", "name", name, "err", err)
			return false, err
		}
		return false, nil
	})

	return summarizeBatch(outcome, batchSummaryConfig{
		PastVerb: "unlocked",
		BaseVerb: "unlock",
	})
}

func (c *UnlockCommand) unlockOne(ctx context.Context, name string) error {
	if !c.containerManager.Exists(ctx, name) {
		return fmt.Errorf("container '%s' not found", name)
	}

	if !isLocked(ctx, c.containerManager, name) {
		return fmt.Errorf("'%s' %w", name, ErrNotLocked)
	}

	if err := c.containerManager.DeleteFromContainer(ctx, name, lockFilePath); err != nil {
		return fmt.Errorf("failed to remove lock file from '%s': %w", name, err)
	}

	ui.DefaultLogger.Info("unlocked", "name", name)
	return nil
}
