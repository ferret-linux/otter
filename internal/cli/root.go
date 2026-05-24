package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/urfave/cli/v3"

	"github.com/ferret-linux/otter/internal/rootful"
	"github.com/ferret-linux/otter/pkg/config"
	"github.com/ferret-linux/otter/pkg/containermanager"
	"github.com/ferret-linux/otter/pkg/containermanager/providers"
	"github.com/ferret-linux/otter/pkg/ui"
	"github.com/ferret-linux/otter/pkg/version"
)

type contextKey string

const containerManagerKey contextKey = "containerManager"

func NewRootCommand(cfg *config.Values) *cli.Command {
	return &cli.Command{
		Name:    "otter",
		Version: version.Version,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "verbose",
				Aliases: []string{"v"},
				Value:   cfg.Verbose,
			},
			&cli.StringFlag{
				Name:   "sudo-command",
				Hidden: true,
				Value:  cfg.SudoProgram,
			},
		},
		Commands: subcommands(cfg),
		ExitErrHandler: func(ctx context.Context, cmd *cli.Command, err error) {
			if err == nil {
				return
			}
			if cmd.Bool("verbose") {
				ui.DefaultLogger.Error("%s", err)
			} else {
				parts := strings.Split(err.Error(), ": ")
				ui.DefaultLogger.Error("%s", parts[len(parts)-1])
			}
		},
	}
}

func printMissingContainerManager(l *ui.Logger) {
	l.Error("Missing dependency: we need a container manager.")
	l.Info("Please install one of podman, nerdctl, or docker.\nYou can follow the documentation on:\n\tman otter-compatibility\nor:\n\thttps://github.com/89luca89/distrobox/blob/main/docs/compatibility.md")
}

func printInvalidContainerManager(l *ui.Logger, containerManagerType string) {
	l.Error("Invalid input %s.", containerManagerType)
	l.Warn("The available choices are: 'autodetect', 'podman', 'nerdctl', 'docker'")
}

func subcommands(cfg *config.Values) []*cli.Command {
	cc := &CommandComposer[config.Values]{cfg: cfg}

	list := cc.apply(
		newListCommand,
		withUsageErrorHandler,
		withRoot,
		withContainerManager,
	)

	generateEntry := cc.apply(
		newGenerateEntryCommand,
		withUsageErrorHandler,
		withRoot,
		withContainerManager,
	)

	create := cc.apply(
		newCreateCommand,
		withUsageErrorHandler,
		withRoot,
		withContainerManager,
	)

	enter := cc.apply(
		newEnterCommand,
		withUsageErrorHandler,
		withRoot,
		withContainerManager,
	)

	assemble := cc.apply(
		newAssembleCommand,
		withUsageErrorHandler,
		withContainerManager,
	)

	remove := cc.apply(
		newRmCommand,
		withUsageErrorHandler,
		withRoot,
		withContainerManager,
	)

	stop := cc.apply(
		newStopCommand,
		withUsageErrorHandler,
		withRoot,
		withContainerManager,
	)

	ephemeral := cc.apply(
		newEphemeralCommand,
		withUsageErrorHandler,
		withRoot,
		withContainerManager,
	)

	upgrade := cc.apply(
		newUpgradeCommand,
		withUsageErrorHandler,
		withRoot,
		withContainerManager,
	)

	lock := cc.apply(
		newLockCommand,
		withUsageErrorHandler,
		withRoot,
		withContainerManager,
	)

	unlock := cc.apply(
		newUnlockCommand,
		withUsageErrorHandler,
		withRoot,
		withContainerManager,
	)

	return []*cli.Command{
		assemble,
		create,
		enter,
		ephemeral,
		generateEntry,
		list,
		lock,
		remove,
		stop,
		unlock,
		upgrade,
	}
}

func withUsageErrorHandler(_ *config.Values, cmd *cli.Command) *cli.Command {
	cmd.OnUsageError = func(_ context.Context, _ *cli.Command, err error, _ bool) error {
		ui.DefaultLogger.Error("%s", err)
		return cli.Exit("", 1)
	}
	return cmd
}

// withRoot declares the --root flag on a command and, when it is set,
// validates that sudo is usable. The actual root-mode container manager is
// built by withContainerManager, which reads the same flag.
func withRoot(_ *config.Values, cmd *cli.Command) *cli.Command {
	cmd.Flags = append(cmd.Flags, &cli.BoolFlag{
		Name:    "root",
		Aliases: []string{"r"},
	})

	prev := cmd.Before
	cmd.Before = func(ctx context.Context, c *cli.Command) (context.Context, error) {
		if prev != nil {
			var err error
			ctx, err = prev(ctx, c)
			if err != nil {
				return nil, err
			}
		}
		if c.Bool("root") {
			if err := rootful.Validate(ctx, c.String("sudo-command")); err != nil {
				return nil, fmt.Errorf("cannot run in root mode: %w", err)
			}
		}
		return ctx, nil
	}
	return cmd
}

// withContainerManager builds the container manager for a command and stores
// it in the context. It reads --root from the command (zero value if the flag
// is not declared), so commands without withRootSupport always get a rootless
// manager.
func withContainerManager(cfg *config.Values, cmd *cli.Command) *cli.Command {
	cmd.Flags = append(cmd.Flags, &cli.StringFlag{
		Name:   "container-manager",
		Hidden: true,
		Value:  cfg.ContainerManagerType,
	})

	prev := cmd.Before
	cmd.Before = func(ctx context.Context, c *cli.Command) (context.Context, error) {
		if prev != nil {
			var err error
			ctx, err = prev(ctx, c)
			if err != nil {
				return nil, err
			}
		}
		cm, err := buildContainerManager(
			ctx,
			c.String("container-manager"),
			c.String("sudo-command"),
			c.Bool("verbose"),
			c.Bool("root"),
		)
		if err != nil {
			return nil, err
		}
		return context.WithValue(ctx, containerManagerKey, cm), nil
	}
	return cmd
}

func buildContainerManager(
	_ context.Context,
	containerManagerType string,
	sudoCommand string,
	verbose bool,
	root bool,
) (containermanager.ContainerManager, error) {
	errLogger := ui.NewLogger(os.Stderr)

	switch containerManagerType {
	case "docker":
		return providers.NewDocker(root, sudoCommand, verbose), nil
	case "podman":
		return providers.NewPodman(root, sudoCommand, verbose), nil
	case "nerdctl":
		return providers.NewNerdctl(root, sudoCommand, verbose), nil
	case "autodetect", "":
		cm, err := providers.NewAutoDetect(root, sudoCommand, verbose)
		if err != nil {
			if errors.Is(err, providers.ErrNoContainerManager) {
				printMissingContainerManager(errLogger)
			}
			return nil, fmt.Errorf("failed to auto-detect container manager: %w", err)
		}
		return cm, nil
	default:
		printInvalidContainerManager(errLogger, containerManagerType)
		return nil, fmt.Errorf("invalid input %s", containerManagerType)
	}
}

// CommandComposer is a helper for building commands with options.
// It holds the config, so that options don't need to receive it as an argument.
type CommandComposer[CFG any] struct {
	cfg *CFG
}

// apply builds a command factory by applying options left-to-right.
// Order matches Before execution: the first option's work runs first.
func (cc *CommandComposer[CFG]) apply(factory func(*CFG) *cli.Command, options ...func(*CFG, *cli.Command) *cli.Command) *cli.Command {
	cmd := factory(cc.cfg)
	for _, option := range options {
		cmd = option(cc.cfg, cmd)
	}
	return cmd
}
