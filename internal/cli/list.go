package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/urfave/cli/v3"

	"github.com/ferret-linux/otter/pkg/commands"
	"github.com/ferret-linux/otter/pkg/config"
	"github.com/ferret-linux/otter/pkg/containermanager"
	"github.com/ferret-linux/otter/pkg/ui"
)

const containerIDDisplayLength = 12

func newListCommand(_ *config.Values) *cli.Command {
	return &cli.Command{
		Name:    "list",
		Aliases: []string{"ls"},
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "json",
				Aliases: []string{"j"},
			},
		},
		Action: listAction,
	}
}

func listAction(ctx context.Context, cmd *cli.Command) error {
	containerManager, ok := ctx.Value(containerManagerKey).(containermanager.ContainerManager)
	if !ok {
		return errors.New("container manager not found in context")
	}

	result, err := commands.NewListCommand(containerManager).Execute(ctx)
	if err != nil {
		return fmt.Errorf("failed to execute list command: %w", err)
	}

	if cmd.Bool("json") {
		return printResultJSON(result)
	}

	printResult(result)

	return nil
}

func printResultJSON(result *commands.ListResult) error {
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

func printResult(result *commands.ListResult) {
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
