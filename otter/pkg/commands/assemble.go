package commands

import (
	"context"
	"errors"
	"fmt"
	"os"
	"regexp"
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
	// BoxNames are the names of the boxes to assemble.
	// If specified, the Assemble command will only assemble the given boxes.
	// If empty, the command will assemble all boxes defined in the manifest.
	BoxNames []string
	// Delete indicates whether to delete the existing box before assembling.
	// true=delete, false=create or update.
	Delete  bool
	Replace bool
}

type AssembleCommand struct {
	createCmd            *CreateCommand
	rmCmd                *RmCommand
	lockCmd              *LockCommand
	containerManager     containermanager.ContainerManager
	enterCmd             *EnterCommand
	createCmdRoot        *CreateCommand
	rmCmdRoot            *RmCommand
	lockCmdRoot          *LockCommand
	containerManagerRoot containermanager.ContainerManager
	enterCmdRoot         *EnterCommand
	progress             *ui.Progress
	cfg                  *config.Values
}

func NewAssembleCommand(
	cfg *config.Values,
	cm containermanager.ContainerManager,
) *AssembleCommand {
	cmRoot := cm.CloneAsRoot()
	return &AssembleCommand{
		createCmd:            NewCreateCommand(cfg, cm, ui.NewDevNullProgress()),
		rmCmd:                NewRmCommand(cm),
		lockCmd:              NewLockCommand(cm),
		containerManager:     cm,
		enterCmd:             NewEnterCommand(cm),
		createCmdRoot:        NewCreateCommand(cfg, cmRoot, ui.NewDevNullProgress()),
		rmCmdRoot:            NewRmCommand(cmRoot),
		lockCmdRoot:          NewLockCommand(cmRoot),
		containerManagerRoot: cmRoot,
		enterCmdRoot:         NewEnterCommand(cmRoot),
		progress:             ui.NewProgress(os.Stderr),
		cfg:                  cfg,
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
		if *item.Settings.Rootful || ac.cfg.DefaultRootful {
			if _, err := rootcheck.Validate(ctx, opts.SudoCommand); err != nil {
				return fmt.Errorf("cannot run in root mode: %w", err)
			}
			break
		}
	}

	itemsByName := make(map[string]manifest.Item, len(items))
	for _, item := range items {
		itemsByName[item.Name] = item
	}

	boxNames := opts.BoxNames
	if len(boxNames) == 0 {
		boxNames = make([]string, len(items))
		for i, item := range items {
			boxNames[i] = item.Name
		}
	} else {
		for _, name := range boxNames {
			if _, ok := itemsByName[name]; !ok {
				return fmt.Errorf("box '%s' not found in manifest", name)
			}
		}
	}

	pastVerb, baseVerb := "created", "create"
	switch {
	case opts.Delete:
		pastVerb, baseVerb = "removed", "remove"
	case opts.Replace:
		pastVerb, baseVerb = "replaced", "replace"
	}

	outcome := runBatch(ctx, boxNames, func(ctx context.Context, name string) (bool, error) {
		item := itemsByName[name]
		var err error
		switch {
		case opts.Delete:
			err = ac.deleteItem(ctx, item)
		case opts.Replace:
			err = ac.replaceItem(ctx, item)
		default:
			err = ac.createItem(ctx, item)
		}
		if err != nil {
			ui.DefaultLogger.Error(fmt.Sprintf("failed to %s item", baseVerb), "name", name, "err", err)
			return false, err
		}
		return false, nil
	})

	return summarizeBatch(outcome, batchSummaryConfig{
		PastVerb: pastVerb,
		BaseVerb: baseVerb,
	})
}

func (ac *AssembleCommand) deleteItem(ctx context.Context, item manifest.Item) error {
	ac.progress.Next("deleting %s...", item.Name)
	opts := RmOptions{
		Force:          true,
		All:            false,
		ContainerNames: []string{item.Name},
	}

	rmCmd := ac.rmCmd
	if *item.Settings.Rootful || ac.cfg.DefaultRootful {
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
	ac.progress.Next("creating %s...", item.Name)
	rootful := *item.Settings.Rootful || ac.cfg.DefaultRootful
	initSystem := *item.Settings.InitSystem || ac.cfg.DefaultInitSystem

	opts := CreateOptions{
		ContainerClone:          item.Clone,
		ContainerName:           item.Name,
		ContainerImage:          item.Image,
		ContainerHostname:       item.Settings.Hostname,
		UnshareNetNs:            *item.Isolation.Netns || *item.Isolation.All,
		UnshareDevsys:           *item.Isolation.Devsys || *item.Isolation.All,
		UnshareGroups:           *item.Isolation.Groups || *item.Isolation.All || initSystem,
		UnshareIpc:              *item.Isolation.IPC || *item.Isolation.All,
		UnshareProcess:          *item.Isolation.Process || *item.Isolation.All || initSystem,
		AdditionalFlags:         item.Additional.Flags,
		AdditionalVolumes:       item.Additional.Volumes,
		AdditionalPackages:      item.Additional.Packages,
		ContainerPreInitHook:    ac.joinHooks(item.Hooks.PreInit),
		ContainerInitHook:       ac.joinHooks(item.Hooks.PostInit),
		ContainerUserCustomHome: item.Home,
		Init:                    initSystem,
		GPU:                     *item.Hardware.GPU,
		NoUsernsLimit:           *item.Isolation.UsernsNoLimit || ac.cfg.DefaultUsernsNoLimit,
		Memory:                  item.Hardware.Memory,
		CPUThreads:              item.Hardware.CPU,
		GenerateEntry:           *item.Settings.Entry && !ac.cfg.DefaultNoEntry,
		Rootful:                 rootful,
		ContainerAlwaysPull:     *item.ForcePull,
		ContainerShell:          item.Settings.Shell,
	}

	createCmd := ac.createCmd
	if rootful {
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
		if rootful {
			rmCmd = ac.rmCmdRoot
		}
		if _, rmErr := rmCmd.Execute(cleanupCtx, RmOptions{
			Force:          true,
			Root:           rootful,
			ContainerNames: []string{item.Name},
		}); rmErr != nil {
			ui.DefaultLogger.Warn("failed to remove item", "name", item.Name, "err", rmErr)
		}
	}()

	err = ac.setupBox(ctx, item)
	if err != nil {
		ac.progress.Fail()
		return err
	}

	if *item.Settings.Lock {
		lockCmd := ac.lockCmd
		if rootful {
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
	// A hook that already ends in its own terminator (`;` or `&&`) keeps
	// it; otherwise insert ` && ` so consecutive hooks don't run together.
	selfTerminated := regexp.MustCompile(`(;|&&)[[:space:]]?$`)

	sb := strings.Builder{}

	for i, hook := range hooks {
		sb.WriteString(hook)
		if i == len(hooks)-1 {
			continue
		}
		if selfTerminated.MatchString(hook) {
			sb.WriteString(" ")
		} else {
			sb.WriteString(" && ")
		}
	}
	return sb.String()
}

func (ac *AssembleCommand) setupBox(ctx context.Context, item manifest.Item) error {
	cm := ac.containerManager
	enterCmd := ac.enterCmd
	if *item.Settings.Rootful || ac.cfg.DefaultRootful {
		cm = ac.containerManagerRoot
		enterCmd = ac.enterCmdRoot
	}

	hasExports := len(item.Exported.Apps) > 0 || len(item.Exported.Bins) > 0

	if *item.StartNow || hasExports {
		if err := cm.Start(ctx, item.Name); err != nil {
			return fmt.Errorf("failed to start container '%s': %w", item.Name, err)
		}
	}
	if *item.StartNow {
		_, err := enterCmd.Execute(ctx, EnterOptions{
			ContainerName: item.Name,
			NoTTY:         true,
			CustomCommand: []string{"true"}, // we just want to run the init hooks, so we can skip the shell
		})
		if err != nil {
			return fmt.Errorf("failed to execute init hooks for item '%s': %w", item.Name, err)
		}
	}

	if err := ac.exportApps(ctx, enterCmd, item); err != nil {
		return err
	}

	if err := ac.exportBins(ctx, enterCmd, item); err != nil {
		return err
	}

	if hasExports && !*item.StartNow {
		if err := cm.Stop(ctx, []string{item.Name}, false); err != nil {
			return fmt.Errorf("failed to stop container '%s' after exporting: %w", item.Name, err)
		}
	}

	return nil
}

// exportApps validates and exports every app declared in item.Exported.Apps.
func (ac *AssembleCommand) exportApps(ctx context.Context, enterCmd *EnterCommand, item manifest.Item) error {
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
	return nil
}

// exportBins validates and exports every bin declared in item.Exported.Bins.
func (ac *AssembleCommand) exportBins(ctx context.Context, enterCmd *EnterCommand, item manifest.Item) error {
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
