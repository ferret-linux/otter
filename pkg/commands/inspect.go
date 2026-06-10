package commands

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

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
		return errors.New("please specify a container name with --name/-n")
	}

	if strings.Contains(opts.ContainerName, ",") {
		return errors.New("inspect only accepts a single container name")
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

	p := ui.NewPanel(os.Stdout)
	p.AddSection("General",
		ui.PanelRow("Name", opts.ContainerName),
		ui.PanelRow("ID", result.ContainerID),
		ui.PanelRow("Created", created),
		ui.PanelRow("Status", result.ContainerStatus),
		ui.PanelRow("Image", result.ContainerImage),
		ui.PanelRow("Platform", result.ContainerPlatform),
		ui.PanelRow("Hostname", result.ContainerHostname),
		ui.PanelRow("Shell", result.ContainerShell),
		ui.PanelRow("Home", result.ContainerHome),
		ui.PanelRow("Locked", strconv.FormatBool(locked)),
		ui.PanelRow("Rootful", strconv.FormatBool(result.Rootful)),
		ui.PanelRow("Manager", opts.Manager),
	)
	p.AddSection("Resources",
		ui.PanelRow("Memory", memory),
		ui.PanelRow("CPU", cpuThreads),
	)
	p.AddSection("Features",
		ui.PanelRow("Init", boolToEnabledStr(result.Init)),
		ui.PanelRow("Nvidia", boolToEnabledStr(result.Nvidia)),
	)
	p.AddSection("Isolation",
		ui.PanelRow("IPC", boolToSharedStr(result.UnshareIPC)),
		ui.PanelRow("Network", boolToSharedStr(result.UnshareNetNS)),
		ui.PanelRow("Process", boolToSharedStr(result.UnshareProcess)),
		ui.PanelRow("Devices", boolToSharedStr(result.UnshareDevsys)),
		ui.PanelRow("Groups", boolToSharedStr(result.UnshareGroups)),
	)
	p.Render()

	return nil
}
