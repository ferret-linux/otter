package commands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/ferret-linux/otter/internal/userenv"
	"github.com/ferret-linux/otter/pkg/containermanager"
	"github.com/ferret-linux/otter/pkg/ui"
)

type RmResult struct {
	Containers []containermanager.Container
}

type RmCommand struct {
	containerManager containermanager.ContainerManager
	listCmd          *ListCommand
	generateEntryCmd *GenerateEntryCommand
}

type RmOptions struct {
	Force          bool
	BypassLock     bool
	All            bool
	RemoveHome     bool
	Root           bool
	ContainerNames []string
}

func NewRmCommand(cm containermanager.ContainerManager) *RmCommand {
	listCmd := NewListCommand(cm)
	generateEntryCmd := NewGenerateEntryCommand(listCmd, cm)
	return &RmCommand{
		containerManager: cm,
		listCmd:          listCmd,
		generateEntryCmd: generateEntryCmd,
	}
}

func (c *RmCommand) Execute(ctx context.Context, options RmOptions) (*RmResult, error) {
	listResult, err := c.listCmd.Execute(ctx, ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed while listing containers: %w", err)
	}

	otterContainersToRemove := getContainersToRemove(listResult.Containers, options.ContainerNames, options.All)

	userEnv := userenv.LoadUserEnvironment(ctx)
	userHome := userEnv.Home

	var removedOtterContainers []containermanager.Container
	for _, currentOtterContainer := range otterContainersToRemove {
		if !options.BypassLock && isLocked(ctx, c.containerManager, currentOtterContainer.Name) {
			if options.All {
				ui.DefaultLogger.Warn("'%s' is locked, skipping", currentOtterContainer.Name)
				continue
			}
			ui.DefaultLogger.Error("'%s' is locked, run 'otter unlock %s' first", currentOtterContainer.Name, currentOtterContainer.Name)
			continue
		}
		err := c.removeContainer(ctx, currentOtterContainer, options.Force, options.RemoveHome, userHome, options.Root)
		if err != nil {
			ui.DefaultLogger.Error("failed deleting %s: %s", currentOtterContainer.Name, err)
		} else {
			removedOtterContainers = append(removedOtterContainers, currentOtterContainer)
		}
	}

	if len(otterContainersToRemove) == 0 && options.All {
		ui.DefaultLogger.Warn("no containers found to remove")
	}

	for _, name := range options.ContainerNames {
		if !slices.ContainsFunc(otterContainersToRemove, func(c containermanager.Container) bool {
			return c.Name == name
		}) {
			ui.DefaultLogger.Warn("container '%s' not found", name)
		}
	}

	if len(removedOtterContainers) > 0 {
		names := make([]string, len(removedOtterContainers))
		for i, c := range removedOtterContainers {
			names[i] = c.Name
		}
		ui.DefaultLogger.Ok("removed, %s", strings.Join(names, " "))
	}

	return &RmResult{Containers: removedOtterContainers}, nil
}

func (c *RmCommand) removeContainer(
	ctx context.Context,
	container containermanager.Container,
	force bool,
	removeHome bool,
	userHome string,
	root bool,
) error {
	inspectOutput, err := c.containerManager.InspectContainer(ctx, container.Name)
	if err != nil {
		return fmt.Errorf("error inspecting the container: %w", err)
	}

	// For custom-home containers, ContainerHome is now the canonical in-container
	// path (e.g. /home/ferret), not a real host path, so the actual host directory
	// to remove is ContainerCustomHomeSource instead, when present.
	homeToRemove := inspectOutput.ContainerHome
	if inspectOutput.ContainerCustomHomeSource != "" {
		homeToRemove = inspectOutput.ContainerCustomHomeSource
	}

	// Hard safety guard: never allow removing home if it is empty,
	// equals the host home, or is a known dangerous path.
	dangerousPaths := []string{"/", "/home", "/root", "/usr"}
	if homeToRemove == "" ||
		homeToRemove == userHome ||
		slices.Contains(dangerousPaths, homeToRemove) {
		if removeHome {
			ui.DefaultLogger.Warn("refusing to remove home '%s': unsafe path", homeToRemove)
		}
		removeHome = false
	}

	cmOptions := containermanager.RmOptions{
		Force:         force,
		RemoveHome:    removeHome,
		ContainerHome: homeToRemove,
	}

	err = c.containerManager.Remove(ctx, container.Name, cmOptions)
	if err != nil {
		return fmt.Errorf("failed to remove container: %w", err)
	}

	c.cleanup(ctx, userHome, container.Name, root)

	return nil
}

func (c *RmCommand) cleanup(ctx context.Context, userHome, containerName string, root bool) {
	bins := findExportedBinaries(userHome, containerName, root)
	desktopApps := findExportedDesktopApps(userHome, containerName, root)

	toDelete := slices.Concat(bins, desktopApps)

	for _, path := range toDelete {
		if err := os.Remove(path); err != nil {
			ui.DefaultLogger.Warn("failed to remove file '%s': %s", path, err)
		}
	}

	err := c.generateEntryCmd.Execute(
		ctx,
		&GenerateEntryOptions{
			ContainerNames: []string{containerName},
			Delete:         true,
			Root:           root,
		},
	)
	if err != nil {
		ui.DefaultLogger.Warn("failed to remove desktop entry for '%s': %s", containerName, err)
	}
}

func getContainersToRemove(
	containers []containermanager.Container,
	names []string,
	all bool,
) []containermanager.Container {
	if all {
		return containers
	}

	var filtered []containermanager.Container
	for _, container := range containers {
		if slices.ContainsFunc(names, func(name string) bool {
			return container.Name == name
		}) {
			filtered = append(filtered, container)
		}
	}

	return filtered
}

func findExportedBinaries(userHome, containerName string, root bool) []string {
	binDir := filepath.Join(userHome, ".local", "bin")

	marker := containerName
	if root {
		marker = "rootful-" + containerName
	}

	entries, err := os.ReadDir(binDir)
	if err != nil {
		return nil
	}

	var files []string

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		path := filepath.Join(binDir, entry.Name())

		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		content := string(data)
		if strings.Contains(content, "# otter_binary") &&
			strings.Contains(content, "# name: "+marker+"\n") {
			absPath, err := filepath.Abs(path)
			if err != nil {
				continue
			}

			files = append(files, absPath)
		}
	}

	return files
}

func findExportedDesktopApps(userHome, containerName string, root bool) []string {
	prefix := containerName
	if root {
		prefix = "rootful-" + containerName
	}
	appsPattern := filepath.Join(userHome, ".local", "share", "applications", prefix+"*")

	matches, err := filepath.Glob(appsPattern)
	if err != nil {
		ui.DefaultLogger.Warn("failed to glob desktop apps: %s", err)
		return []string{}
	}

	var files []string

	for _, desktopFile := range matches {
		iconValue, ok := parseDesktopExport(desktopFile, containerName)
		if !ok {
			continue
		}

		absDesktop, err := filepath.Abs(desktopFile)
		if err != nil {
			continue
		}

		files = append(files, absDesktop)

		if iconValue != "" {
			files = append(files, findIconFiles(userHome, iconValue)...)
		}
	}

	return files
}

func parseDesktopExport(desktopFile, containerName string) (string, bool) {
	data, err := os.ReadFile(desktopFile)
	if err != nil {
		return "", false
	}

	hasExecMatch := false
	var iconValue string

	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "Exec=") && strings.Contains(line, containerName+" ") {
			hasExecMatch = true
		}

		if strings.HasPrefix(line, "Icon=") {
			iconValue = strings.TrimPrefix(line, "Icon=")
		}
	}

	return iconValue, hasExecMatch
}

func findIconFiles(userHome, iconName string) []string {
	iconsDir := filepath.Join(userHome, ".local", "share", "icons")
	iconPrefix := iconName + "."

	var files []string

	if err := filepath.WalkDir(iconsDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil //nolint:nilerr // skip unreadable directories
		}

		if d.IsDir() {
			return nil
		}

		if strings.HasPrefix(d.Name(), iconPrefix) {
			absIcon, err := filepath.Abs(path)
			if err == nil {
				files = append(files, absIcon)
			}
		}

		return nil
	}); err != nil && !os.IsNotExist(err) {
		ui.DefaultLogger.Warn("failed to walk icons dir: %v", err)
	}

	return files
}
