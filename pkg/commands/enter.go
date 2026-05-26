package commands

import (
	"context"
	"fmt"

	"github.com/ferret-linux/otter/pkg/config"
	"github.com/ferret-linux/otter/pkg/containermanager"
)

type EnterResult struct{}

type EnterOptions struct {
	ContainerName   string
	AdditionalFlags string
	CustomCommand   []string
	DryRun          bool
	NoTTY           bool
	Verbose         bool
	CleanPath       bool
}

type EnterCommand struct {
	cfg              *config.Values
	containerManager containermanager.ContainerManager
}

func NewEnterCommand(
	cfg *config.Values,
	cm containermanager.ContainerManager,
) *EnterCommand {
	return &EnterCommand{
		cfg:              cfg,
		containerManager: cm,
	}
}

func (c *EnterCommand) Execute(ctx context.Context, opts EnterOptions) (*EnterResult, error) {
	if !opts.DryRun {
		inspectResult, err := c.containerManager.InspectContainer(ctx, opts.ContainerName)
		if err != nil {
			return nil, fmt.Errorf("container '%s' not found", opts.ContainerName)
		}
		if inspectResult.ContainerStatus != containermanager.RunningStatus {
			return nil, fmt.Errorf("container '%s' is stopped, run 'otter start %s'", opts.ContainerName, opts.ContainerName)
		}
		if !c.containerManager.IsSetupDone(ctx, opts.ContainerName) {
			return nil, fmt.Errorf("container '%s' is initializing, please wait", opts.ContainerName)
		}
	}

	cmdOpts := containermanager.EnterOptions{
		ContainerName:   opts.ContainerName,
		AdditionalFlags: opts.AdditionalFlags,
		CustomCommand:   opts.CustomCommand,
		DryRun:          opts.DryRun,
		NoTTY:           opts.NoTTY,
		Verbose:         opts.Verbose,
		CleanPath:       opts.CleanPath,
	}

	err := c.containerManager.Enter(ctx, cmdOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to enter the container: %w", err)
	}

	return &EnterResult{}, nil
}
