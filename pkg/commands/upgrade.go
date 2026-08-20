package commands

import (
	"context"
	"errors"
	"fmt"

	"github.com/ferret-linux/otter/internal/insidecontainer"
	"github.com/ferret-linux/otter/pkg/config"
	"github.com/ferret-linux/otter/pkg/containermanager"
	"github.com/ferret-linux/otter/pkg/ui"
)

const upgradeScript = "" +
	"command -v su-exec 2>/dev/null && su-exec root /usr/lib/otter/scripts/otter-init --upgrade || " +
	"command -v doas 2>/dev/null && doas /usr/lib/otter/scripts/otter-init --upgrade || " +
	"sudo -S /usr/lib/otter/scripts/otter-init --upgrade"

type UpgradeOptions struct {
	ContainerNames []string
	All            bool
	Running        bool
}

type UpgradeCommand struct {
	cfg              *config.Values
	containerManager containermanager.ContainerManager
	listCmd          *ListCommand
	enterCmd         *EnterCommand
}

func NewUpgradeCommand(
	cfg *config.Values,
	cm containermanager.ContainerManager,
) *UpgradeCommand {
	return &UpgradeCommand{
		cfg:              cfg,
		containerManager: cm,
		listCmd:          NewListCommand(cm),
		enterCmd:         NewEnterCommand(cm),
	}
}

func (c *UpgradeCommand) Execute(ctx context.Context, opts *UpgradeOptions) error {
	var containerNames []string

	switch {
	case opts.All, opts.Running:
		containers, err := c.listCmd.Execute(ctx, ListOptions{})
		if err != nil {
			return fmt.Errorf("failed to list containers: %w", err)
		}

		if len(containers.Containers) == 0 {
			return ErrNoContainersFound
		}

		containerNames = make([]string, 0, len(containers.Containers))
		for _, container := range containers.Containers {
			if opts.Running && !container.IsRunning() {
				continue
			}

			containerNames = append(containerNames, container.Name)
		}

		if len(containerNames) == 0 {
			return ErrNoContainersFound
		}
	case len(opts.ContainerNames) > 0:
		containerNames = opts.ContainerNames
	default:
		return errors.New("please specify a container name or use --all")
	}

	outcome := runBatch(ctx, containerNames, func(ctx context.Context, name string) (bool, error) {
		if IsLocked(ctx, c.containerManager, name) {
			ui.DefaultLogger.Warn("locked, skipping", "name", name)
			return true, nil
		}
		if err := c.upgradeContainer(ctx, name); err != nil {
			ui.DefaultLogger.Error("failed while upgrading", "name", name, "err", err)
			return false, err
		}
		return false, nil
	})

	return summarizeBatch(outcome, batchSummaryConfig{
		PastVerb:          "upgraded",
		BaseVerb:          "upgrade",
		AllSkippedMessage: "all requested containers are locked, run 'otter unlock' first",
		AllSkippedIsError: true,
	})
}

func (c *UpgradeCommand) upgradeContainer(ctx context.Context, name string) error {
	if _, updated, err := insidecontainer.ProvisionScripts(c.cfg.ScriptsDir); err != nil {
		ui.DefaultLogger.Warn("failed to provision scripts", "err", err)
	} else if updated {
		ui.DefaultLogger.Info("otter scripts updated")
	} else {
		ui.DefaultLogger.Info("otter scripts already up to date")
	}

	if err := c.containerManager.Start(ctx, name); err != nil {
		return fmt.Errorf("failed to start container '%s' for upgrade: %w", name, err)
	}

	enterOpts := EnterOptions{
		ContainerName: name,
		CustomCommand: []string{"sh", "-c", upgradeScript},
	}

	if _, err := c.enterCmd.Execute(ctx, enterOpts); err != nil {
		return fmt.Errorf("failed to upgrade container %s: %w", name, err)
	}

	return nil
}
