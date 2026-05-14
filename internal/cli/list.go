package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

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
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name: "no-color",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return listAction(ctx, cmd, cfg)
		},
	}
}

func listAction(ctx context.Context, cmd *cli.Command, cfg *config.Values) error {
	containerManager, ok := ctx.Value(containerManagerKey).(containermanager.ContainerManager)
	if !ok {
		return errors.New("container manager not found in context")
	}

	listCmd := commands.NewListCommand(cfg, containerManager)
	result, err := listCmd.Execute(ctx)
	if err != nil {
		return fmt.Errorf("failed to execute list command: %w", err)
	}

	noColor := cmd.Bool("no-color") || !isTerminal()
	printResult(result, noColor)

	return nil
}

func printResult(result *commands.ListResult, noColor bool) {
	if len(result.Containers) == 0 {
		//nolint:forbidigo // Using fmt.Println is acceptable here for CLI output
		fmt.Println("no containers found")
		return
	}

	var buf strings.Builder
	w := tabwriter.NewWriter(&buf, 0, 0, 3, ' ', 0)

	fmt.Fprintln(w, "ID\tNAME\tSTATUS\tIMAGE")
	for _, c := range result.Containers {
		status := "○ " + c.Status
		if c.IsRunning() {
			status = "● " + c.Status
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", c.ID, c.Name, status, c.Image)
	}
	w.Flush()

	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")

	//nolint:forbidigo // Using fmt.Println is acceptable here for CLI output
	fmt.Println(lines[0])
	for i, c := range result.Containers {
		line := lines[i+1]
		switch {
		case noColor:
			fmt.Println(line)
		case c.IsRunning():
			fmt.Println(ui.Green(line))
		case strings.Contains(strings.ToLower(c.Status), "exited"):
			fmt.Println(ui.Dim(line))
		default:
			fmt.Println(ui.Yellow(line))
		}
	}
}

func isTerminal() bool {
	stat, _ := os.Stdout.Stat()
	return (stat.Mode() & os.ModeCharDevice) != 0
}
