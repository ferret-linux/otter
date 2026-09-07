package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"runtime"
	"strings"

	"charm.land/log/v2"
	"github.com/charmbracelet/colorprofile"
	"github.com/urfave/cli/v3"

	"github.com/ferret-linux/otter/pkg/commands"
	"github.com/ferret-linux/otter/pkg/config"
	"github.com/ferret-linux/otter/pkg/containermanager"
	"github.com/ferret-linux/otter/pkg/containermanager/providers"
	"github.com/ferret-linux/otter/pkg/rootcheck"
	"github.com/ferret-linux/otter/pkg/ui"
)

type contextKey string

const containerManagerKey contextKey = "containerManager"

// commit and buildTime are injected at build time via -ldflags (see Makefile).
// They stay at these defaults for builds that don't go through `make build`.
//
//nolint:gochecknoglobals // required so `-ldflags -X` can inject values at build time
var (
	commit    = "unknown"
	buildTime = "unknown"
)

func NewRootCommand(cfg *config.Values) *cli.Command {
	//nolint:reassign // urfave/cli's documented mechanism for customizing version output
	cli.VersionPrinter = func(cmd *cli.Command) {
		root := cmd.Root()
		fmt.Fprintf(root.Writer, "%s %s\n\nInfo\n", root.Name, root.Version)
		fmt.Fprintf(root.Writer, "  → version    : %s\n", root.Version)
		fmt.Fprintf(root.Writer, "  → platform   : %s\n", runtime.GOARCH)
		fmt.Fprintf(root.Writer, "  → git commit : %s\n", commit)
		fmt.Fprintf(root.Writer, "  → go version : %s\n", runtime.Version())
		fmt.Fprintf(root.Writer, "  → build time : %s\n", buildTime)
	}
	return &cli.Command{
		Name:    "otter",
		Version: "0.0.9",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "sudo-command",
				Aliases: []string{"sc"},
				Hidden:  true,
				Value:   cfg.SudoProgram,
			},
			&cli.StringFlag{
				Name:    "container-manager",
				Aliases: []string{"cm"},
				Hidden:  true,
				Value:   cfg.ContainerManagerType,
			},
		},
		Before: func(ctx context.Context, _ *cli.Command) (context.Context, error) {
			ui.SetNoColor(ui.ShouldDisableColor(os.Stdout))
			if ui.NoColor() {
				ui.DefaultLogger.SetColorProfile(colorprofile.Ascii)
			}
			return ctx, nil
		},
		Commands: subcommands(cfg),
		ExitErrHandler: func(_ context.Context, _ *cli.Command, err error) {
			if err == nil {
				return
			}
			ui.DefaultLogger.Error(err)
		},
	}
}

func printMissingContainerManager(l *log.Logger) {
	l.Error("Missing dependency: we need a container manager.")
	l.Info("Please install one of podman, nerdctl, or docker.\nRun `otter documentation` for more details.")
}

func printInvalidContainerManager(l *log.Logger, containerManagerType string) {
	l.Error("Invalid input", "value", containerManagerType)
	l.Warn("The available choices are: 'autodetect', 'podman', 'nerdctl', 'docker'")
}

func subcommands(cfg *config.Values) []*cli.Command {
	cc := &CommandComposer[config.Values]{cfg: cfg}

	stdOpts := []func(*config.Values, *cli.Command) *cli.Command{
		withUsageErrorHandler,
		withRoot,
		withContainerManager,
	}
	noRootOpts := []func(*config.Values, *cli.Command) *cli.Command{
		withUsageErrorHandler,
		withContainerManager,
	}
	documentationOpts := []func(*config.Values, *cli.Command) *cli.Command{
		withUsageErrorHandler,
	}

	// Order matches the alphabetical listing urfave/cli renders in --help.
	specs := []struct {
		factory func(*config.Values) *cli.Command
		opts    []func(*config.Values, *cli.Command) *cli.Command
	}{
		{newAssembleCommand, noRootOpts},
		{newCreateCommand, stdOpts},
		{newDocumentationCommand, documentationOpts},
		{newEnterCommand, stdOpts},
		{newGenerateEntryCommand, stdOpts},
		{newInspectCommand, stdOpts},
		{newJournalCommand, stdOpts},
		{newListCommand, stdOpts},
		{newLockCommand, stdOpts},
		{newPauseCommand, stdOpts},
		{newRegistryCommand, nil},
		{newRmCommand, stdOpts},
		{newRestartCommand, stdOpts},
		{newSettingsCommand, documentationOpts},
		{newStartCommand, stdOpts},
		{newStopCommand, stdOpts},
		{newUnlockCommand, stdOpts},
		{newUpgradeCommand, stdOpts},
	}

	commands := make([]*cli.Command, len(specs))
	for i, spec := range specs {
		commands[i] = cc.apply(spec.factory, spec.opts...)
	}
	return commands
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

// runContainerCommand implements the wiring shared by every container batch
// action: pulling the container manager out of ctx (guaranteed present by
// withContainerManager, but checked defensively in case a command is ever
// wired without it), parsing container names from the CLI args, running
// execute, and translating commands.ErrNoContainersFound into a warning
// rather than a hard error. failMsg prefixes any other error from execute.
func runContainerCommand(
	ctx context.Context,
	cmd *cli.Command,
	failMsg string,
	execute func(cm containermanager.ContainerManager, names []string) error,
) error {
	containerManager, ok := ctx.Value(containerManagerKey).(containermanager.ContainerManager)
	if !ok {
		return errors.New("container manager not found in context")
	}

	names, err := splitNames(cmd.Args().Slice())
	if err != nil {
		return err
	}

	err = execute(containerManager, names)
	if errors.Is(err, commands.ErrNoContainersFound) {
		ui.DefaultLogger.Warn("no containers found")
		return nil
	}
	if err != nil {
		return fmt.Errorf("%s: %w", failMsg, err)
	}
	return nil
}

func withUsageErrorHandler(_ *config.Values, cmd *cli.Command) *cli.Command {
	cmd.OnUsageError = func(_ context.Context, _ *cli.Command, err error, _ bool) error {
		ui.DefaultLogger.Error(err)
		os.Exit(1)
		return nil
	}
	return cmd
}

// withRoot declares the --root flag on a command, resolves its effective
// value against cfg.DefaultRootful, and persists the resolved value back
// onto the flag so every downstream reader of --root (this command's own
// action, and withContainerManager) sees a single consistent answer.
// When it resolves to true, it also validates that sudo is usable.
func withRoot(cfg *config.Values, cmd *cli.Command) *cli.Command {
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
		if !c.Bool("root") && cfg.DefaultRootful {
			if err := c.Set("root", "true"); err != nil {
				return nil, fmt.Errorf("failed to apply rootful default: %w", err)
			}
		}
		if c.Bool("root") {
			if _, err := rootcheck.Validate(ctx, c.String("sudo-command")); err != nil {
				return nil, fmt.Errorf("cannot run in root mode: %w", err)
			}
		}
		return ctx, nil
	}
	return cmd
}

// withContainerManager builds the container manager for a command and stores
// it in the context. It reads --container-manager and --sudo-command from the
// root command, and --root from the current command (zero value if the flag
// is not declared), so commands without withRoot always get a rootless manager.
func withContainerManager(_ *config.Values, cmd *cli.Command) *cli.Command {
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
			c.Root().String("container-manager"),
			c.Root().String("sudo-command"),
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
	ctx context.Context,
	containerManagerType string,
	sudoCommand string,
	root bool,
) (containermanager.ContainerManager, error) {
	if root && sudoCommand == "autodetect" {
		resolved, err := rootcheck.Validate(ctx, sudoCommand)
		if err != nil {
			return nil, fmt.Errorf("failed to auto-detect sudo program: %w", err)
		}
		sudoCommand = resolved
	}

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
				printMissingContainerManager(ui.DefaultLogger)
				os.Exit(1)
			}
			return nil, fmt.Errorf("failed to auto-detect container manager: %w", err)
		}
		return cm, nil
	default:
		printInvalidContainerManager(ui.DefaultLogger, containerManagerType)
		os.Exit(1)
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
