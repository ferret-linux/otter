package cli

import (
	"context"
	"errors"
	"strings"

	"github.com/urfave/cli/v3"

	"github.com/ferret-linux/otter/pkg/commands"
	"github.com/ferret-linux/otter/pkg/config"
	"github.com/ferret-linux/otter/pkg/containermanager"
)

func newInspectCommand(cfg *config.Values) *cli.Command {
	return &cli.Command{
		Name:    "inspect",
		Aliases: []string{"info"},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "name",
				Aliases: []string{"n"},
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return inspectAction(ctx, cmd, cfg)
		},
	}
}

func inspectAction(ctx context.Context, cmd *cli.Command, cfg *config.Values) error {
	cm, ok := ctx.Value(containerManagerKey).(containermanager.ContainerManager)
	if !ok {
		return errors.New("container manager not found in context")
	}

	if strings.Contains(cmd.String("name"), ",") {
		return errors.New("inspect only accepts a single container name")
	}

	inspectCmd := commands.NewInspectCommand(cfg, cm)
	return inspectCmd.Execute(ctx, commands.InspectOptions{
		ContainerName: cmd.String("name"),
		Manager:       cm.Name(),
	})
}
