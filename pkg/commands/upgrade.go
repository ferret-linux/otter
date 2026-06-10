package commands

import (
	"context"
	"errors"
	"fmt"

	"github.com/ferret-linux/otter/internal/insidecontainer"
	"github.com/ferret-linux/otter/pkg/containermanager"
	"github.com/ferret-linux/otter/pkg/netcheck"
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
	containerManager containermanager.ContainerManager
	listCmd          *ListCommand
	startCmd         *StartCommand
	enterCmd         *EnterCommand
}

func NewUpgradeCommand(
	cm containermanager.ContainerManager,
) *UpgradeCommand {
	return &UpgradeCommand{
		containerManager: cm,
		listCmd:          NewListCommand(cm),
		startCmd:         NewStartCommand(cm),
		enterCmd:         NewEnterCommand(cm),
	}
}

//nolint:gocognit // ignore cognitive complexity here, the function orchestrates multi-step container upgrade
func (c *UpgradeCommand) Execute(ctx context.Context, opts *UpgradeOptions) error {
	var containerNames []string

	switch {
	case opts.All, opts.Running:
		containers, err := c.listCmd.Execute(ctx)
		if err != nil {
			return fmt.Errorf("failed to list containers: %w", err)
		}

		if len(containers.Containers) == 0 {
			return ErrEmptyContainerList
		}

		containerNames = make([]string, 0, len(containers.Containers))
		for _, container := range containers.Containers {
			if opts.Running && !container.IsRunning() {
				continue
			}

			containerNames = append(containerNames, container.Name)
		}

		if len(containerNames) == 0 {
			return ErrEmptyContainerList
		}
	case len(opts.ContainerNames) > 0:
		containerNames = opts.ContainerNames
	default:
		return errors.New("please specify a container name with --name/-n")
	}

	var lastErr error

	for _, name := range containerNames {
		if isLocked(ctx, c.containerManager, name) {
			if opts.All || opts.Running {
				ui.DefaultLogger.Warn("'%s' is locked, skipping", name)
				continue
			}
			return fmt.Errorf("'%s' is locked, run 'otter unlock %s' first", name, name)
		}
		if err := c.upgradeContainer(ctx, name); err != nil {
			lastErr = fmt.Errorf("failed while upgrading %s: %w", name, err)
			continue
		}
	}

	return lastErr
}

func (c *UpgradeCommand) upgradeContainer(ctx context.Context, name string) error {
	if err := netcheck.Check(ctx); err != nil {
		ui.DefaultLogger.Warn("%s", err)
		return nil
	}
	if _, updated, err := insidecontainer.ProvisionScripts(); err != nil {
		ui.DefaultLogger.Warn("failed to provision scripts: %s", err)
	} else if updated {
		ui.DefaultLogger.Info("otter scripts updated")
	} else {
		ui.DefaultLogger.Info("otter scripts already up to date")
	}

	if err := c.startCmd.Execute(ctx, &StartOptions{
		ContainerNames: []string{name},
	}); err != nil {
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
