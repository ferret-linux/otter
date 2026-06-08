package cli

import (
	"context"
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

func newListCommand(cfg *config.Values) *cli.Command {
	return &cli.Command{
		Name:    "list",
		Aliases: []string{"ls"},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return listAction(ctx, cmd)
		},
	}
}

func listAction(ctx context.Context, _ *cli.Command) error {
	containerManager, ok := ctx.Value(containerManagerKey).(containermanager.ContainerManager)
	if !ok {
		return errors.New("container manager not found in context")
	}

	result, err := commands.NewListCommand(containerManager).Execute(ctx)
	if err != nil {
		return fmt.Errorf("failed to execute list command: %w", err)
	}

	printResult(result)

	return nil
}

func printResult(result *commands.ListResult) {
	if len(result.Containers) == 0 {
		//nolint:forbidigo // Using fmt.Println is acceptable here for CLI output
		fmt.Println("no containers found")
		return
	}

	t := ui.NewTable(os.Stdout, "ID", "NAME", "STATUS", "IMAGE")
	for _, c := range result.Containers {
		status := "○ " + c.Status
		statusColor := ui.Yellow
		if c.IsRunning() {
			status = "● " + c.Status
			statusColor = ui.Green
		} else if strings.Contains(strings.ToLower(c.Status), "exited") {
			statusColor = ui.Red
		}
		t.AddRow(
			[]string{c.ID, c.Name, status, c.Image},
			[]func(string) string{ui.Dim, ui.Teal, statusColor, ui.Dim},
		)
	}
	t.Render()
}
