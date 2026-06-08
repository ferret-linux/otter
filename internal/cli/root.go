package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/urfave/cli/v3"

	"github.com/ferret-linux/otter/pkg/config"
	"github.com/ferret-linux/otter/pkg/containermanager"
	"github.com/ferret-linux/otter/pkg/containermanager/providers"
	"github.com/ferret-linux/otter/pkg/rootful"
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
			&cli.StringFlag{
				Name:   "sudo-command",
				Hidden: true,
				Value:  cfg.SudoProgram,
			},
			&cli.BoolFlag{
				Name:    "no-color",
				Aliases: []string{"nc"},
			},
		},
		Before: func(ctx context.Context, cmd *cli.Command) (context.Context, error) {
			if cmd.Bool("no-color") {
				ui.SetNoColor(true)
			} else {
				ui.DisableIfNotTerminal()
			}
			return ctx, nil
		},
		Commands: subcommands(cfg),
		ExitErrHandler: func(ctx context.Context, cmd *cli.Command, err error) {

			if err == nil {
				return
			}
			ui.DefaultLogger.Error("%s", err)
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

	start := cc.apply(
		newStartCommand,
		withUsageErrorHandler,
		withRoot,
		withContainerManager,
	)

	restart := cc.apply(
		newRestartCommand,
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

	upgrade := cc.apply(
		newUpgradeCommand,
		withUsageErrorHandler,
		withRoot,
		withContainerManager,
	)

	journal := cc.apply(
		newJournalCommand,
		withUsageErrorHandler,
		withContainerManager,
	)

	inspect := cc.apply(
		newInspectCommand,
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

	registry := newRegistryCommand(cfg)

	return []*cli.Command{
		assemble,
		create,
		enter,
		generateEntry,
		inspect,
		journal,
		list,
		lock,
		registry,
		remove,
		restart,
		start,
		stop,
		unlock,
		upgrade,
	}
}

// firstName returns the first positional argument, or empty string if none.
func firstName(args []string) string {
	if len(args) == 0 {
		return ""
	}
	return args[0]
}

// splitNames splits the first positional argument on commas and returns the resulting slice.
// e.g. "foo,bar" -> ["foo", "bar"]
func splitNames(args []string) ([]string, error) {
	if len(args) == 0 {
		return nil, nil
	}
	if len(args) > 1 {
		return nil, fmt.Errorf("multiple arguments are not supported, use comma-separated names instead (e.g. %s)", strings.Join(args, ","))
	}
	var names []string
	for _, n := range strings.Split(args[0], ",") {
		n = strings.TrimSpace(n)
		if n != "" {
			names = append(names, n)
		}
	}
	return names, nil
}

func withUsageErrorHandler(_ *config.Values, cmd *cli.Command) *cli.Command {
	cmd.OnUsageError = func(_ context.Context, _ *cli.Command, err error, _ bool) error {
		ui.DefaultLogger.Error("%s", err)
		os.Exit(1)
		return nil
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
	root bool,
) (containermanager.ContainerManager, error) {
	errLogger := ui.NewLogger(os.Stderr)

	switch containerManagerType {
	case "docker":
		return providers.NewDocker(root, sudoCommand), nil
	case "podman":
		return providers.NewPodman(root, sudoCommand), nil
	case "nerdctl":
		return providers.NewNerdctl(root, sudoCommand), nil
	case "autodetect", "":
		cm, err := providers.NewAutoDetect(root, sudoCommand)
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
