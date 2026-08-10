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

// inspectHomeValue returns the single home path a container is actually
// using: the real host directory backing a custom home, if one is set,
// otherwise the container's normal home.
func inspectHomeValue(result *containermanager.InspectResult) string {
	if result.ContainerCustomHomeSource != "" {
		return result.ContainerCustomHomeSource
	}
	return result.ContainerHome
}

// InspectResult is the data `otter inspect` surfaces for a single
// container, returned by Execute so both the CLI and the webui can render
// it without Execute writing to stdout itself.
type InspectResult struct {
	ID             string
	Name           string
	Created        string
	Status         string
	Image          string
	Platform       string
	Hostname       string
	Shell          string
	Home           string
	Locked         bool
	Rootful        bool
	Manager        string
	Memory         string
	CPUThreads     int
	Init           bool
	Nvidia         bool
	UnshareIPC     bool
	UnshareNetNS   bool
	UnshareProcess bool
	UnshareDevsys  bool
	UnshareGroups  bool
	UsernsNoLimit  bool
}

func (c *InspectCommand) Execute(ctx context.Context, opts InspectOptions) (*InspectResult, error) {
	if opts.ContainerName == "" {
		return nil, errors.New("please specify a container name")
	}

	if strings.Contains(opts.ContainerName, ",") {
		return nil, errors.New("inspect only accepts a single container name")
	}

	if !c.containerManager.Exists(ctx, opts.ContainerName) {
		return nil, fmt.Errorf("container '%s' not found", opts.ContainerName)
	}

	result, err := c.containerManager.InspectContainer(ctx, opts.ContainerName)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect container: %w", err)
	}

	locked := isLocked(ctx, c.containerManager, opts.ContainerName)

	return &InspectResult{
		ID:             result.ContainerID,
		Name:           opts.ContainerName,
		Created:        result.ContainerCreated,
		Status:         result.ContainerStatus,
		Image:          result.ContainerImage,
		Platform:       result.ContainerPlatform,
		Hostname:       result.ContainerHostname,
		Shell:          result.ContainerShell,
		Home:           inspectHomeValue(result),
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
	}, nil
}

// PrintInspect renders an InspectResult to stdout, either as JSON or as the
// CLI's sectioned table, matching `otter inspect` / `otter inspect --json`.
func PrintInspect(result *InspectResult, jsonMode bool) error {
	if jsonMode {
		return printInspectJSON(result)
	}

	// Trim Created timestamp to readable format
	created := result.Created
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

	id := result.ID
	if len(id) > containerIDDisplayLength {
		id = id[:containerIDDisplayLength]
	}

	type inspectRow struct {
		section string
		key     string
		value   string
	}
	//nolint:goconst // A new constant is useless here
	rows := []inspectRow{
		{"General", "Name", result.Name},
		{"General", "ID", id},
		{"General", "Created", created},
		{"General", "Status", result.Status},
		{"General", "Image", ui.TrimImageRef(result.Image)},
		{"General", "Platform", result.Platform},
		{"General", "Hostname", result.Hostname},
		{"General", "Shell", result.Shell},
		{"General", "Home", result.Home},
		{"General", "Locked", strconv.FormatBool(result.Locked)},
		{"General", "Rootful", strconv.FormatBool(result.Rootful)},
		{"General", "Manager", result.Manager},
		{"Resources", "Memory", memory},
		{"Resources", "CPU", cpuThreads},
		{"Features", "Init", boolToEnabledStr(result.Init)},
		{"Features", "Nvidia", boolToEnabledStr(result.Nvidia)},
		{"Isolation", "IPC", boolToSharedStr(result.UnshareIPC)},
		{"Isolation", "Network", boolToSharedStr(result.UnshareNetNS)},
		{"Isolation", "Process", boolToSharedStr(result.UnshareProcess)},
		{"Isolation", "Devices", boolToSharedStr(result.UnshareDevsys)},
		{"Isolation", "Groups", boolToSharedStr(result.UnshareGroups)},
		{"Isolation", "Userns No Limit", boolToEnabledStr(result.UsernsNoLimit)},
	}

	t := ui.NewTable(os.Stdout, "SECTION", "KEY", "VALUE")
	lastSection := ""
	for i, r := range rows {
		section := r.section
		if section == lastSection {
			section = ""
		} else {
			if i > 0 {
				t.AddSeparator()
			}
			lastSection = section
		}
		t.AddRow(
			[]string{section, r.key, r.value},
			[]func(string) string{ui.Yellow, ui.Teal, ui.Dim},
		)
	}
	t.Render()

	return nil
}

func printInspectJSON(result *InspectResult) error {
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
		ID:             result.ID,
		Name:           result.Name,
		Created:        result.Created,
		Status:         result.Status,
		Image:          result.Image,
		Platform:       result.Platform,
		Hostname:       result.Hostname,
		Shell:          result.Shell,
		Home:           result.Home,
		Locked:         result.Locked,
		Rootful:        result.Rootful,
		Manager:        result.Manager,
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
