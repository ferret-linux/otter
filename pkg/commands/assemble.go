package commands

import (
	"context"
	"errors"
	"fmt"
	"os"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/ferret-linux/otter/pkg/config"
	"github.com/ferret-linux/otter/pkg/containermanager"
	"github.com/ferret-linux/otter/pkg/manifest"
	"github.com/ferret-linux/otter/pkg/rootcheck"
	"github.com/ferret-linux/otter/pkg/ui"
)

const assembleCleanupTimeout = 30 * time.Second

type AssembleOptions struct {
	// ManifestPath is the path to the manifest file (from --file flag). Required.
	ManifestPath string
	// SudoCommand is the sudo program to use for root validation.
	SudoCommand string
	// Boxname is the name of the box to assemble.
	// If specified, the Assemble command will only assemble the given box.
	// If empty, the command will assemble all boxes defined in the manifest.
	Boxname string
	// Delete indicates whether to delete the existing box before assembling.
	// true=delete, false=create or update.
	Delete  bool
	Replace bool
}

type AssembleCommand struct {
	createCmd     *CreateCommand
	rmCmd         *RmCommand
	lockCmd       *LockCommand
	startCmd      *StartCommand
	enterCmd      *EnterCommand
	createCmdRoot *CreateCommand
	rmCmdRoot     *RmCommand
	lockCmdRoot   *LockCommand
	startCmdRoot  *StartCommand
	enterCmdRoot  *EnterCommand
	progress      *ui.Progress
}

func NewAssembleCommand(
	cfg *config.Values,
	cm containermanager.ContainerManager,
) *AssembleCommand {
	cmRoot := cm.CloneAsRoot()
	return &AssembleCommand{
		createCmd:     NewCreateCommand(cfg, cm, ui.NewDevNullProgress()),
		rmCmd:         NewRmCommand(cm),
		lockCmd:       NewLockCommand(cm),
		startCmd:      NewStartCommand(cm),
		enterCmd:      NewEnterCommand(cm),
		createCmdRoot: NewCreateCommand(cfg, cmRoot, ui.NewDevNullProgress()),
		rmCmdRoot:     NewRmCommand(cmRoot),
		lockCmdRoot:   NewLockCommand(cmRoot),
		startCmdRoot:  NewStartCommand(cmRoot),
		enterCmdRoot:  NewEnterCommand(cmRoot),
		progress:      ui.NewProgress(os.Stderr),
	}
}

//nolint:gocognit // ignore cognitive complexity here, the function orchestrates multi-step manifest assembly
func (ac *AssembleCommand) Execute(ctx context.Context, opts AssembleOptions) error {
	if opts.ManifestPath == "" {
		return errors.New("manifest path is required, use --file to specify it")
	}

	items, err := manifest.Parse(ctx, opts.ManifestPath)
	if err != nil {
		return fmt.Errorf("failed to parse manifest file: %w", err)
	}

	for _, item := range items {
		if item.Settings.Rootful {
			if err := rootcheck.Validate(ctx, opts.SudoCommand); err != nil {
				return fmt.Errorf("cannot run in root mode: %w", err)
			}
			break
		}
	}

	filteredItems := items
	if opts.Boxname != "" {
		idx := slices.IndexFunc(items, func(i manifest.Item) bool {
			return i.Name == opts.Boxname
		})
		if idx == -1 {
			return fmt.Errorf("box '%s' not found in manifest", opts.Boxname)
		}
		filteredItems = []manifest.Item{items[idx]}
	}

	for _, item := range filteredItems {
		switch {
		case opts.Delete:
			if err := ac.deleteItem(ctx, item); err != nil {
				return fmt.Errorf("failed to delete item '%s': %w", item.Name, err)
			}
		case opts.Replace:
			if err := ac.replaceItem(ctx, item); err != nil {
				return fmt.Errorf("failed to replace item '%s': %w", item.Name, err)
			}
		default:
			if err := ac.createItem(ctx, item); err != nil {
				return fmt.Errorf("failed to create item '%s': %w", item.Name, err)
			}
		}
	}

	return nil
}

func (ac *AssembleCommand) deleteItem(ctx context.Context, item manifest.Item) error {
	ac.progress.Next("Deleting %s...", item.Name)
	opts := RmOptions{
		Force:          true,
		All:            false,
		ContainerNames: []string{item.Name},
	}

	rmCmd := ac.rmCmd
	if item.Settings.Rootful {
		rmCmd = ac.rmCmdRoot
	}

	_, err := rmCmd.Execute(ctx, opts)
	if err != nil {
		ac.progress.Fail()
		return fmt.Errorf("failed to execute delete item '%s': %w", item.Name, err)
	}
	ac.progress.Done()
	return nil
}

func (ac *AssembleCommand) replaceItem(ctx context.Context, item manifest.Item) error {
	err := ac.deleteItem(ctx, item)
	if err != nil {
		return err
	}

	return ac.createItem(ctx, item)
}

func (ac *AssembleCommand) createItem(ctx context.Context, item manifest.Item) error {
	ac.progress.Next("Creating %s...", item.Name)
	opts := CreateOptions{
		ContainerClone:          item.Clone,
		ContainerName:           item.Name,
		ContainerImage:          item.Image,
		ContainerHostname:       item.Settings.Hostname,
		UnshareNetNs:            item.Isolation.Netns || item.Isolation.All,
		UnshareDevsys:           item.Isolation.Devsys || item.Isolation.All,
		UnshareGroups:           item.Isolation.Groups || item.Isolation.All || item.Settings.InitSystem,
		UnshareIpc:              item.Isolation.IPC || item.Isolation.All,
		UnshareProcess:          item.Isolation.Process || item.Isolation.All || item.Settings.InitSystem,
		AdditionalFlags:         item.Additional.Flags,
		AdditionalVolumes:       item.Additional.Volumes,
		AdditionalPackages:      item.Additional.Packages,
		ContainerPreInitHook:    ac.joinHooks(item.Hooks.PreInit),
		ContainerInitHook:       ac.joinHooks(item.Hooks.PostInit),
		ContainerUserCustomHome: "",
		Init:                    item.Settings.InitSystem,
		Nvidia:                  item.Hardware.Nvidia,
		Memory:                  item.Hardware.Memory,
		CPUThreads:              item.Hardware.CPU,
		GenerateEntry:           item.Settings.Entry,
		Rootful:                 item.Settings.Rootful,
		ContainerAlwaysPull:     item.ForcePull,
		ContainerShell:          item.Settings.Shell,
	}

	createCmd := ac.createCmd
	if item.Settings.Rootful {
		createCmd = ac.createCmdRoot
	}
	_, err := createCmd.Execute(ctx, opts)
	if err != nil {
		ac.progress.Fail()
		return err
	}

	success := false
	defer func() {
		if success {
			return
		}
		cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), assembleCleanupTimeout)
		defer cancel()
		rmCmd := ac.rmCmd
		if item.Settings.Rootful {
			rmCmd = ac.rmCmdRoot
		}
		if _, rmErr := rmCmd.Execute(cleanupCtx, RmOptions{
			Force:          true,
			Root:           item.Settings.Rootful,
			ContainerNames: []string{item.Name},
		}); rmErr != nil {
			ui.DefaultLogger.Warn("%s: %s", item.Name, rmErr)
		}
	}()

	err = ac.setupBox(ctx, item)
	if err != nil {
		ac.progress.Fail()
		return err
	}

	if item.Settings.Lock {
		lockCmd := ac.lockCmd
		if item.Settings.Rootful {
			lockCmd = ac.lockCmdRoot
		}
		if err := lockCmd.Execute(ctx, LockOptions{ContainerNames: []string{item.Name}}); err != nil {
			ac.progress.Fail()
			return err
		}
	}

	ac.progress.Done()
	success = true
	return nil
}

func (ac *AssembleCommand) joinHooks(hooks []string) string {
	sb := strings.Builder{}

	for i, hook := range hooks {
		sb.WriteString(hook)

		if i < len(hooks)-1 {
			semicolonRegex := regexp.MustCompile(`;[[:space:]]{0,1}$`)
			andAndRegex := regexp.MustCompile(`&&[[:space:]]{0,1}$`)

			separator := "  " // two spaces just because v1 does that, so it's comparable in regression tests
			if !semicolonRegex.MatchString(hook) && !andAndRegex.MatchString(hook) {
				separator = " && "
			}

			sb.WriteString(separator)
		}
	}

	return sb.String()
}

func (ac *AssembleCommand) setupBox(ctx context.Context, item manifest.Item) error {
	startCmd := ac.startCmd
	enterCmd := ac.enterCmd
	if item.Settings.Rootful {
		startCmd = ac.startCmdRoot
		enterCmd = ac.enterCmdRoot
	}

	if err := startCmd.Execute(ctx, &StartOptions{ContainerNames: []string{item.Name}}); err != nil {
		return err
	}
	if item.StartNow {
		_, err := enterCmd.Execute(ctx, EnterOptions{
			ContainerName: item.Name,
			NoTTY:         true,
			CustomCommand: []string{"true"}, // we just want to run the init hooks, so we can skip the shell
		})
		if err != nil {
			return fmt.Errorf("failed to execute init hooks for item '%s': %w", item.Name, err)
		}
	}

	// validate app name to prevent command injection, since it's used in a custom command
	var validAppName = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._+\-]*$`)
	for _, app := range item.Exported.Apps {
		if !validAppName.MatchString(app) {
			return fmt.Errorf("invalid app name '%s' for item '%s': must be alphanumeric (with dots, underscores, hyphens)", app, item.Name)
		}
	}
	for _, app := range item.Exported.Apps {
		_, err := enterCmd.Execute(ctx, EnterOptions{
			ContainerName: item.Name,
			NoTTY:         true,
			CustomCommand: []string{"otter-export", "--app", app},
		})
		if err != nil {
			return fmt.Errorf("failed to export app '%s' for item '%s': %w", app, item.Name, err)
		}
	}

	// validate bin path to prevent command injection, since it's used in a custom command
	var validBinPath = regexp.MustCompile(`^/[a-zA-Z0-9._+\-/]+$`)
	if len(item.Exported.Bins) > 0 && !validBinPath.MatchString(item.Exported.Path) {
		return fmt.Errorf("invalid exported bins path '%s' for item '%s': must be an absolute path with alphanumeric characters, dots, underscores, or hyphens", item.Exported.Path, item.Name)
	}
	for _, bin := range item.Exported.Bins {
		if !validBinPath.MatchString(bin) {
			return fmt.Errorf("invalid bin path '%s' for item '%s': must be an absolute path with alphanumeric characters, dots, underscores, or hyphens", bin, item.Name)
		}
	}
	for _, bin := range item.Exported.Bins {
		_, err := enterCmd.Execute(ctx, EnterOptions{
			ContainerName: item.Name,
			NoTTY:         true,
			CustomCommand: []string{"otter-export", "--bin", bin, "--export-path", item.Exported.Path},
		})
		if err != nil {
			return fmt.Errorf("failed to export bin '%s' for item '%s': %w", bin, item.Name, err)
		}
	}

	return nil
}
