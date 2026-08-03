package commands

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/ferret-linux/otter/pkg/containermanager"
	"github.com/ferret-linux/otter/pkg/ui"
)

type InspectOptions struct {
	ContainerName string
	Manager       string
	JSON          bool
}

type InspectCommand struct {
	containerManager containermanager.ContainerManager
}

func NewInspectCommand(cm containermanager.ContainerManager) *InspectCommand {
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

func printInspectJSON(result *containermanager.InspectResult, locked bool, opts InspectOptions) error {
	out := struct {
		ID             string `json:"id"`
		Name           string `json:"name"`
		Created        string `json:"created"`
		Status         string `json:"status"`
		Image          string `json:"image"`
		Platform       string `json:"platform"`
		Hostname       string `json:"hostname"`
		Shell          string `json:"shell"`
		Home           string `json:"home"`
		HostHome       string `json:"host_home,omitempty"`
		Locked         bool   `json:"locked"`
		Rootful        bool   `json:"rootful"`
		Manager        string `json:"manager"`
		Memory         string `json:"memory"`
		CPUThreads     int    `json:"cpu_threads"`
		Init           bool   `json:"init"`
		Nvidia         bool   `json:"nvidia"`
		UnshareIPC     bool   `json:"unshare_ipc"`
		UnshareNetNS   bool   `json:"unshare_netns"`
		UnshareProcess bool   `json:"unshare_process"`
		UnshareDevsys  bool   `json:"unshare_devsys"`
		UnshareGroups  bool   `json:"unshare_groups"`
		UsernsNoLimit  bool   `json:"userns_nolimit"`
	}{
		ID:             result.ContainerID,
		Name:           opts.ContainerName,
		Created:        result.ContainerCreated,
		Status:         result.ContainerStatus,
		Image:          result.ContainerImage,
		Platform:       result.ContainerPlatform,
		Hostname:       result.ContainerHostname,
		Shell:          result.ContainerShell,
		Home:           result.ContainerHome,
		HostHome:       result.ContainerCustomHomeSource,
		Locked:         locked,
		Rootful:        result.Rootful,
		Manager:        opts.Manager,
		Memory:         result.Memory,
		CPUThreads:     result.CPUThreads,
		Init:           result.Init,
		Nvidia:         result.Nvidia,
		UnshareIPC:     result.UnshareIPC,
		UnshareNetNS:   result.UnshareNetNS,
		UnshareProcess: result.UnshareProcess,
		UnshareDevsys:  result.UnshareDevsys,
		UnshareGroups:  result.UnshareGroups,
		UsernsNoLimit:  result.UsernsNoLimit,
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(out); err != nil {
		return fmt.Errorf("failed to encode inspect output as JSON: %w", err)
	}
	return nil
}

func (c *InspectCommand) Execute(ctx context.Context, opts InspectOptions) error {
	if opts.ContainerName == "" {
		return errors.New("please specify a container name")
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

	if opts.JSON {
		return printInspectJSON(result, locked, opts)
	}

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

	id := result.ContainerID
	if len(id) > containerIDDisplayLength {
		id = id[:containerIDDisplayLength]
	}

	p := ui.NewPanel(os.Stdout)
	p.AddSection("General",
		ui.PanelRow("Name", opts.ContainerName),
		ui.PanelRow("ID", id),
		ui.PanelRow("Created", created),
		ui.PanelRow("Status", result.ContainerStatus),
		ui.PanelRow("Image", ui.TrimImageRef(result.ContainerImage)),
		ui.PanelRow("Platform", result.ContainerPlatform),
		ui.PanelRow("Hostname", result.ContainerHostname),
		ui.PanelRow("Shell", result.ContainerShell),
		ui.PanelRow("Home", result.ContainerHome),
		ui.PanelRow("Host Home", result.ContainerCustomHomeSource),
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
		ui.PanelRow("Userns No Limit", boolToEnabledStr(result.UsernsNoLimit)),
	)
	p.Render()

	return nil
}
