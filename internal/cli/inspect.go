package cli

import (
	"context"
	"errors"
	"fmt"

	"github.com/urfave/cli/v3"

	"github.com/ferret-linux/otter/pkg/commands"
	"github.com/ferret-linux/otter/pkg/config"
	"github.com/ferret-linux/otter/pkg/containermanager"
)

func newInspectCommand(_ *config.Values) *cli.Command {
	return &cli.Command{
		Name:    "inspect",
		Aliases: []string{"info"},
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "json",
				Aliases: []string{"j"},
			},
		},
		Action: inspectAction,
	}
}

func inspectAction(ctx context.Context, cmd *cli.Command) error {
	cm, ok := ctx.Value(containerManagerKey).(containermanager.ContainerManager)
	if !ok {
		return errors.New("container manager not found in context")
	}

	inspectCmd := commands.NewInspectCommand(cm)
	result, err := inspectCmd.Execute(ctx, commands.InspectOptions{
		ContainerName: firstName(cmd.Args().Slice()),
		Manager:       cm.Name(),
		JSON:          cmd.Bool("json"),
	})
	if err != nil {
		return fmt.Errorf("failed to inspect container: %w", err)
	}
	if err := commands.PrintInspect(result, cmd.Bool("json")); err != nil {
		return fmt.Errorf("failed to inspect container: %w", err)
	}
	return nil
}
