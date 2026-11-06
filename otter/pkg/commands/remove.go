package commands

import (
	"context"
	"errors"
	"fmt"
	"slices"

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
	Prune          bool
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
	if !options.All && len(options.ContainerNames) == 0 {
		return nil, errors.New("please specify a container name or use --all")
	}

	listResult, err := c.listCmd.Execute(ctx, ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed while listing containers: %w", err)
	}

	otterContainersToRemove := getContainersToRemove(listResult.Containers, options.ContainerNames, options.All)

	userEnv := userenv.LoadTargetUserEnvironment(ctx)
	userHome := userEnv.Home

	containersByName := make(map[string]containermanager.Container, len(otterContainersToRemove))
	for _, container := range otterContainersToRemove {
		containersByName[container.Name] = container
	}

	// Named containers are processed even when they are absent from the
	// runtime: their exports may still be on disk from an earlier removal or
	// from a runtime that no longer knows about them. --all only targets
	// containers that actually exist.
	containerNames := make([]string, 0, len(options.ContainerNames)+len(otterContainersToRemove))
	if options.All {
		for _, container := range otterContainersToRemove {
			containerNames = append(containerNames, container.Name)
		}
	} else {
		seen := make(map[string]struct{}, len(options.ContainerNames))
		for _, name := range options.ContainerNames {
			if _, ok := seen[name]; ok {
				continue
			}
			seen[name] = struct{}{}
			containerNames = append(containerNames, name)
		}
	}

	var removedOtterContainers []containermanager.Container
	outcome := runBatch(ctx, containerNames, func(ctx context.Context, name string) (bool, error) {
		container, exists := containersByName[name]
		if !exists {
			if options.RemoveHome {
				ui.DefaultLogger.Warn(
					"refusing to remove home: container not found, home is unknown",
					"name", name,
				)
			}
			if err := CleanupContainerExports(
				ctx,
				c.containerManager.Name(),
				name,
				options.Root,
				options.Prune,
			); err != nil {
				ui.DefaultLogger.Error("failed to clean exports", "name", name, "err", err)
				return false, err
			}
			if err := c.generateEntryCmd.Execute(
				ctx,
				&GenerateEntryOptions{
					ContainerNames: []string{name},
					Delete:         true,
					Root:           options.Root,
				},
			); err != nil {
				ui.DefaultLogger.Warn("failed to remove desktop entry", "name", name, "err", err)
			}
			ui.DefaultLogger.Info("container not found, cleaned exports", "name", name)
			return false, nil
		}

		if !options.BypassLock && isLocked(ctx, c.containerManager, name) {
			if options.Prune {
				if err := CleanupContainerExports(
					ctx,
					c.containerManager.Name(),
					name,
					options.Root,
					true,
				); err != nil {
					ui.DefaultLogger.Warn("failed to prune exports", "name", name, "err", err)
				}
			}
			ui.DefaultLogger.Warn(fmt.Sprintf("locked, run 'otter unlock %s' first, skipping", name), "name", name)
			return true, nil
		}
		if err := c.removeContainer(ctx, container, options, userHome); err != nil {
			if options.Prune {
				if cleanupErr := CleanupContainerExports(
					ctx,
					c.containerManager.Name(),
					name,
					options.Root,
					true,
				); cleanupErr != nil {
					ui.DefaultLogger.Warn("failed to prune exports", "name", name, "err", cleanupErr)
				}
			}
			ui.DefaultLogger.Error("failed to remove", "name", name, "err", err)
			return false, err
		}
		removedOtterContainers = append(removedOtterContainers, container)
		ui.DefaultLogger.Info("removed", "name", name)
		return false, nil
	})

	if options.All && len(otterContainersToRemove) == 0 {
		ui.DefaultLogger.Warn("no containers found to remove")
	}

	if len(containerNames) == 0 {
		return &RmResult{Containers: removedOtterContainers}, nil
	}

	err = summarizeBatch(outcome, batchSummaryConfig{
		PastVerb:          "removed",
		BaseVerb:          "remove",
		AllSkippedMessage: "all requested containers are locked, run 'otter unlock' first",
		AllSkippedIsError: true,
	})

	return &RmResult{Containers: removedOtterContainers}, err
}

func (c *RmCommand) removeContainer(
	ctx context.Context,
	container containermanager.Container,
	options RmOptions,
	userHome string,
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

	removeHome := options.RemoveHome
	// Hard safety guard: never allow removing home if it is empty,
	// equals the host home, or is a known dangerous path.
	dangerousPaths := []string{"/", "/home", "/root", "/usr"}
	if homeToRemove == "" ||
		homeToRemove == userHome ||
		slices.Contains(dangerousPaths, homeToRemove) {
		if removeHome {
			ui.DefaultLogger.Warn("refusing to remove home, unsafe path", "path", homeToRemove)
		}
		removeHome = false
	}

	cmOptions := containermanager.RmOptions{
		Force:         options.Force,
		RemoveHome:    removeHome,
		ContainerHome: homeToRemove,
	}

	err = c.containerManager.Remove(ctx, container.Name, cmOptions)
	if err != nil {
		return fmt.Errorf("failed to remove container: %w", err)
	}

	c.cleanup(ctx, container.Name, options.Root, options.Prune)

	return nil
}

func (c *RmCommand) cleanup(ctx context.Context, containerName string, root, prune bool) {
	if err := CleanupContainerExports(
		ctx,
		c.containerManager.Name(),
		containerName,
		root,
		prune,
	); err != nil {
		ui.DefaultLogger.Warn("failed to clean exports", "container", containerName, "err", err)
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
		ui.DefaultLogger.Warn("failed to remove desktop entry", "container", containerName, "err", err)
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
