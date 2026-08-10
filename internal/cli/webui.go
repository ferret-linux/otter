package cli

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"net"

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
			&cli.BoolFlag{
				Name:  "allow-remote",
				Usage: "allow binding the webui to a non-loopback address (can also be set via settings.allow-remote in config)",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return webuiAction(ctx, cmd, cfg)
		},
	}
}

func webuiAction(ctx context.Context, cmd *cli.Command, cfg *config.Values) error {
	containerManager, ok := ctx.Value(containerManagerKey).(containermanager.ContainerManager)
	if !ok {
		return errors.New("container manager not found in context")
	}

	bind := cmd.String("bind")
	allowRemote := cmd.Bool("allow-remote") || cfg.DefaultWebUIAllowRemote
	if !isLoopbackHost(bind) && !allowRemote {
		return fmt.Errorf(
			"refusing to bind webui to non-loopback address %q without --allow-remote (or settings.allow-remote in config): "+
				"the webui is protected only by its access token, and exposing it beyond localhost lets anyone who can reach the port manage your containers",
			bind,
		)
	}

	token, err := resolveWebUIToken(cfg)
	if err != nil {
		return fmt.Errorf("failed to prepare webui: %w", err)
	}

	addr := fmt.Sprintf("%s:%d", bind, cmd.Int("port"))
	if err := webui.Serve(ctx, containerManager, cfg, cmd.Bool("root"), addr, token); err != nil {
		return fmt.Errorf("failed to run webui: %w", err)
	}
	return nil
}

// isLoopbackHost reports whether host — as used in an http.Server bind
// address, i.e. without a port — refers only to the local machine. An empty
// host means "all interfaces" to net/http, so that is treated as non-loopback.
func isLoopbackHost(host string) bool {
	if host == "" {
		return false
	}
	if host == "localhost" {
		return true
	}
	if ip := net.ParseIP(host); ip != nil {
		return ip.IsLoopback()
	}
	addrs, err := net.LookupHost(host)
	if err != nil || len(addrs) == 0 {
		return false
	}
	for _, a := range addrs {
		ip := net.ParseIP(a)
		if ip == nil || !ip.IsLoopback() {
			return false
		}
	}
	return true
}

// resolveWebUIToken returns the configured webui access token, or generates
// a random one if none is configured.
func resolveWebUIToken(cfg *config.Values) (string, error) {
	if cfg.DefaultWebUIToken != "" {
		return cfg.DefaultWebUIToken, nil
	}
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("failed to generate webui token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}
