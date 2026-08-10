package cli

import (
	"context"
	"errors"
	"fmt"

	"github.com/urfave/cli/v3"

	"github.com/ferret-linux/otter/internal/webui"
	"github.com/ferret-linux/otter/pkg/config"
	"github.com/ferret-linux/otter/pkg/containermanager"
)

func newWebUICommand(cfg *config.Values) *cli.Command {
	return &cli.Command{
		Name:  "webui",
		Usage: "start a local web dashboard for managing containers",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "bind",
				Usage: "address to bind the webui to",
				Value: cfg.DefaultWebUIBind,
			},
			&cli.IntFlag{
				Name:  "port",
				Usage: "port to bind the webui to",
				Value: cfg.DefaultWebUIPort,
			},
		},
		Action: webuiAction,
	}
}

func webuiAction(ctx context.Context, cmd *cli.Command) error {
	containerManager, ok := ctx.Value(containerManagerKey).(containermanager.ContainerManager)
	if !ok {
		return errors.New("container manager not found in context")
	}

	addr := fmt.Sprintf("%s:%d", cmd.String("bind"), cmd.Int("port"))
	if err := webui.Serve(ctx, containerManager, addr); err != nil {
		return fmt.Errorf("failed to run webui: %w", err)
	}
	return nil
}
