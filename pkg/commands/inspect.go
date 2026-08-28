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
		Locked         bool   `json:"locked"`
		Rootful        bool   `json:"rootful"`
		Manager        string `json:"manager"`
		Memory         string `json:"memory"`
		CPUThreads     int    `json:"cpu_threads"`
		Init           bool   `json:"init"`
		GPU            string `json:"gpu"`
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
		Home:           inspectHomeValue(result),
		Locked:         locked,
		Rootful:        result.Rootful,
		Manager:        opts.Manager,
		Memory:         result.Memory,
		CPUThreads:     result.CPUThreads,
		Init:           result.Init,
		GPU:            result.GPU,
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

	type inspectRow struct {
		section string
		key     string
		value   string
	}
	//nolint:goconst // A new constant is useless here
	rows := []inspectRow{
		{"General", "Name", opts.ContainerName},
		{"General", "ID", id},
		{"General", "Created", created},
		{"General", "Status", result.ContainerStatus},
		{"General", "Image", ui.TrimImageRef(result.ContainerImage)},
		{"General", "Platform", result.ContainerPlatform},
		{"General", "Hostname", result.ContainerHostname},
		{"General", "Shell", result.ContainerShell},
		{"General", "Home", inspectHomeValue(result)},
		{"General", "Locked", strconv.FormatBool(locked)},
		{"General", "Rootful", strconv.FormatBool(result.Rootful)},
		{"General", "Manager", opts.Manager},
		{"Resources", "Memory", memory},
		{"Resources", "CPU", cpuThreads},
		{"Features", "Init", boolToEnabledStr(result.Init)},
		{"Features", "GPU", result.GPU},
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
