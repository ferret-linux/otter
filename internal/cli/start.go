package cli

import (
	"context"
	"errors"

	"github.com/urfave/cli/v3"

	"github.com/ferret-linux/otter/pkg/commands"
	"github.com/ferret-linux/otter/pkg/config"
	"github.com/ferret-linux/otter/pkg/containermanager"
)

func newStartCommand(cfg *config.Values) *cli.Command {
	return &cli.Command{
		Name: "start",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "name",
				Aliases: []string{"n"},
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return startAction(ctx, cmd, cfg)
		},
	}
}

func startAction(ctx context.Context, cmd *cli.Command, cfg *config.Values) error {
	containerManager, ok := ctx.Value(containerManagerKey).(containermanager.ContainerManager)
	if !ok {
		return errors.New("container manager not found in context")
	}

	containerName := cmd.String("name")
	if containerName == "" && cmd.Args().Len() > 0 {
		containerName = cmd.Args().First()
	}

	startCmd := commands.NewStartCommand(cfg, containerManager)
	if err := startCmd.Execute(ctx, &commands.StartOptions{
		ContainerName: containerName,
		DryRun:        cmd.Bool("dry-run"),
		Verbose:       cmd.Bool("verbose"),
	}); err != nil {
		return err
	}

	return nil
}
