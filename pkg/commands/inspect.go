package commands

import (
	"context"
	"fmt"

	"github.com/ferret-linux/otter/pkg/config"
	"github.com/ferret-linux/otter/pkg/containermanager"
	"github.com/ferret-linux/otter/pkg/ui"
)

type InspectOptions struct {
	ContainerName string
	Manager       string
}

type InspectCommand struct {
	containerManager containermanager.ContainerManager
}

func NewInspectCommand(_ *config.Values, cm containermanager.ContainerManager) *InspectCommand {
	return &InspectCommand{
		containerManager: cm,
	}
}

func boolToEnabledStr(b bool) string {
	if b {
		return "enabled"
	}
	return "disabled"
}

func boolToSharedStr(b bool) string {
	if b {
		return "unshared"
	}
	return "shared"
}

func (c *InspectCommand) Execute(ctx context.Context, opts InspectOptions) error {
	if opts.ContainerName == "" {
		return fmt.Errorf("please specify a container name with --name/-n")
	}

	if !c.containerManager.Exists(ctx, opts.ContainerName) {
		return fmt.Errorf("container '%s' not found", opts.ContainerName)
	}

	result, err := c.containerManager.InspectContainer(ctx, opts.ContainerName)
	if err != nil {
		return fmt.Errorf("failed to inspect container: %w", err)
	}

	locked := isLocked(ctx, c.containerManager, opts.ContainerName)

	// Trim Created timestamp to readable format
	created := result.ContainerCreated
	if len(created) > 19 {
		created = created[:10] + " " + created[11:19]
	}

	memory := result.Memory
	if memory == "" {
		memory = "unlimited"
	}

	cpuThreads := "unlimited"
	if result.CPUThreads > 0 {
		cpuThreads = fmt.Sprintf("%d threads", result.CPUThreads)
	}

	ui.DefaultLogger.Info("Name:        %s", opts.ContainerName)
	ui.DefaultLogger.Info("ID:          %s", result.ContainerID)
	ui.DefaultLogger.Info("Created:     %s", created)
	ui.DefaultLogger.Info("Status:      %s", result.ContainerStatus)
	ui.DefaultLogger.Info("Image:       %s", result.ContainerImage)
	ui.DefaultLogger.Info("Platform:    %s", result.ContainerPlatform)
	ui.DefaultLogger.Info("Hostname:    %s", result.ContainerHostname)
	ui.DefaultLogger.Info("Shell:       %s", result.ContainerShell)
	ui.DefaultLogger.Info("Home:        %s", result.ContainerHome)
	ui.DefaultLogger.Info("Locked:      %v", locked)
	ui.DefaultLogger.Info("Rootful:     %v", result.Rootful)
	ui.DefaultLogger.Info("Manager:     %s", opts.Manager)
	ui.DefaultLogger.Info("")
	ui.DefaultLogger.Info("Resources:")
	ui.DefaultLogger.Info("  Memory:    %s", memory)
	ui.DefaultLogger.Info("  CPU:       %s", cpuThreads)
	ui.DefaultLogger.Info("")
	ui.DefaultLogger.Info("Features:")
	ui.DefaultLogger.Info("  Init:      %s", boolToEnabledStr(result.Init))
	ui.DefaultLogger.Info("  Nvidia:    %s", boolToEnabledStr(result.Nvidia))
	ui.DefaultLogger.Info("")
	ui.DefaultLogger.Info("Isolation:")
	ui.DefaultLogger.Info("  IPC:       %s", boolToSharedStr(result.UnshareIPC))
	ui.DefaultLogger.Info("  Network:   %s", boolToSharedStr(result.UnshareNetNS))
	ui.DefaultLogger.Info("  Process:   %s", boolToSharedStr(result.UnshareProcess))
	ui.DefaultLogger.Info("  Devices:   %s", boolToSharedStr(result.UnshareDevsys))
	ui.DefaultLogger.Info("  Groups:    %s", boolToSharedStr(result.UnshareGroups))

	return nil
}
