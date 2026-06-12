package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/ferret-linux/otter/pkg/containermanager"
	"github.com/ferret-linux/otter/pkg/ui"
)

const containerIDDisplayLength = 12

type ListOptions struct{}

type ListResult struct {
	Containers []containermanager.Container
}

type ListCommand struct {
	containerManager containermanager.ContainerManager
}

func NewListCommand(cm containermanager.ContainerManager) *ListCommand {
	return &ListCommand{
		containerManager: cm,
	}
}

func (c *ListCommand) Execute(ctx context.Context, _ ListOptions) (*ListResult, error) {
	containers, err := c.containerManager.ListContainers(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed while listing contaiers: %w", err)
	}

	var otterContainers []containermanager.Container
	for _, container := range containers {
		if container.IsOtterContainer() {
			otterContainers = append(otterContainers, container)
		}
	}

	// Sort by container name to keep `otter list` output stable for downstream UIs; see https://github.com/89luca89/distrobox/issues/2071.
	slices.SortFunc(otterContainers, func(a, b containermanager.Container) int {
		return strings.Compare(a.Name, b.Name)
	})

	result := &ListResult{Containers: otterContainers}

	return result, nil
}

func PrintListJSON(result *ListResult) error {
	type containerJSON struct {
		ID     string `json:"id"`
		Name   string `json:"name"`
		Status string `json:"status"`
		Image  string `json:"image"`
	}

	out := make([]containerJSON, 0, len(result.Containers))
	for _, c := range result.Containers {
		out = append(out, containerJSON{
			ID:     c.ID,
			Name:   c.Name,
			Status: c.Status,
			Image:  c.Image,
		})
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(out); err != nil {
		return fmt.Errorf("failed to encode list output as JSON: %w", err)
	}
	return nil
}

func PrintList(result *ListResult) {
	if len(result.Containers) == 0 {
		ui.DefaultLogger.Warn("No containers found.")
		return
	}

	t := ui.NewTable(os.Stdout, "ID", "NAME", "STATUS", "IMAGE")
	for _, c := range result.Containers {
		id := c.ID
		if len(id) > containerIDDisplayLength {
			id = id[:containerIDDisplayLength]
		}
		status := "○ " + c.Status
		statusColor := ui.Yellow
		if c.IsRunning() {
			status = "● " + c.Status
			statusColor = ui.Green
		} else if strings.Contains(strings.ToLower(c.Status), "exited") {
			statusColor = ui.Red
		}
		t.AddRow(
			[]string{id, c.Name, status, ui.TrimImageRef(c.Image)},
			[]func(string) string{ui.Dim, ui.Teal, statusColor, ui.Dim},
		)
	}
	t.Render()
}
