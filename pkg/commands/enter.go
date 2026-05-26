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
		if !c.containerManager.Exists(ctx, opts.ContainerName) {
			return nil, fmt.Errorf("container '%s' does not exist", opts.ContainerName)
		}

		containers, _ := c.containerManager.ListContainers(ctx)
		for _, ct := range containers {
			if ct.Name == opts.ContainerName && !ct.IsRunning() {
				return nil, fmt.Errorf("container '%s' is stopped, run 'otter start %s'", opts.ContainerName, opts.ContainerName)
			}
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
