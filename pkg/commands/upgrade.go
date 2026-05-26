package commands

import (
	"context"
	"errors"
	"fmt"

	insideContainer "github.com/ferret-linux/otter/internal/inside-container"
	"github.com/ferret-linux/otter/pkg/config"
	"github.com/ferret-linux/otter/pkg/containermanager"
	"github.com/ferret-linux/otter/pkg/ui"
)

//nolint:lll // upgrade command mirrors the shell version
const upgradeScript = `command -v su-exec 2>/dev/null && su-exec root /usr/bin/entrypoint --upgrade || command -v doas 2>/dev/null && doas /usr/bin/entrypoint --upgrade || sudo -S /usr/bin/entrypoint --upgrade`

type UpgradeOptions struct {
	ContainerNames []string
	All            bool
	Running        bool
	DryRun         bool
	Verbose        bool
}

type UpgradeCommand struct {
	cfg              *config.Values
	containerManager containermanager.ContainerManager
	listCmd          *ListCommand
	startCmd         *StartCommand
	enterCmd         *EnterCommand
}

var ErrUpgradeNoContainerSpecified = errors.New("please specify the name of the container")

func NewUpgradeCommand(
	cfg *config.Values,
	cm containermanager.ContainerManager,
	progress *ui.Progress,
) *UpgradeCommand {
	return &UpgradeCommand{
		cfg:              cfg,
		containerManager: cm,
		listCmd:          NewListCommand(cfg, cm),
		startCmd:         NewStartCommand(cfg, cm),
		enterCmd:         NewEnterCommand(cfg, cm),
	}
}

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
		return ErrUpgradeNoContainerSpecified
	}

	var lastErr error

	for _, name := range containerNames {
		if opts.Verbose {
			ui.DefaultLogger.Info("upgrading '%s'", name)
		}
		if isLocked(ctx, c.containerManager, name) {
			if opts.All || opts.Running {
				ui.DefaultLogger.Warn("'%s' is locked, skipping", name)
				continue
			}
			return fmt.Errorf("'%s' is locked, run 'otter unlock %s' first", name, name)
		}
		if err := c.upgradeContainer(ctx, name, opts.DryRun); err != nil {
			lastErr = fmt.Errorf("failed while upgrading %s: %w", name, err)
			continue
		}
	}

	return lastErr
}

func (c *UpgradeCommand) upgradeContainer(ctx context.Context, name string, dryRun bool) error {
	if !dryRun {
		if _, updated, err := insideContainer.ProvisionScripts(); err != nil {
			ui.DefaultLogger.Warn("failed to provision scripts: %s", err)
		} else if updated {
			ui.DefaultLogger.Info("otter scripts updated")
		} else {
			ui.DefaultLogger.Info("otter scripts already up to date")
		}
	}

	if err := c.startCmd.Execute(ctx, &StartOptions{
		ContainerName: name,
		DryRun:        dryRun,
	}); err != nil {
		return err
	}

	enterOpts := EnterOptions{
		ContainerName: name,
		CustomCommand: []string{"sh", "-c", upgradeScript},
		DryRun:        dryRun,
	}

	if _, err := c.enterCmd.Execute(ctx, enterOpts); err != nil {
		return fmt.Errorf("failed to upgrade container %s: %w", name, err)
	}

	return nil
}
