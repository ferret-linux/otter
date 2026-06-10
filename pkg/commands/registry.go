package commands

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/ferret-linux/otter/pkg/containermanager"
	"github.com/ferret-linux/otter/pkg/registry"
	"github.com/ferret-linux/otter/pkg/ui"
)

// RegistryList renders a table of available images from props.
// If all is false, disabled images are omitted.
func RegistryList(props *registry.ImagesProperties, all bool) {
	t := ui.NewTable(os.Stdout, "NAME", "ARCH", "STATUS", "IMAGE")
	for _, entry := range props.Images {
		if !all && !entry.Enabled {
			continue
		}

		status := "enabled"
		statusColor := ui.Green
		imageRef := entry.OfficialImage

		if !props.ImagesAvailable {
			status = "offline"
			statusColor = ui.Red
			imageRef = entry.FallbackVendorImage
		} else if !entry.Enabled {
			status = "disabled"
			statusColor = ui.Yellow
			imageRef = entry.FallbackVendorImage
		}

		arch := strings.Join(entry.Architecture, ", ")

		t.AddRow(
			[]string{entry.Name, arch, status, imageRef},
			[]func(string) string{ui.Teal, ui.Dim, statusColor, ui.Dim},
		)
	}
	t.Render()
}

// RegistryPull pulls the given image names using the container manager.
// Names may be comma-separated and are split before resolution.
// If all is true, all enabled images not yet locally present are pulled.
// force pulls even if already present.
func RegistryPull(
	ctx context.Context,
	cm containermanager.ContainerManager,
	props *registry.ImagesProperties,
	names []string,
	all bool,
	force bool,
	progress *ui.Progress,
) error {
	targets, err := resolvePullTargets(ctx, cm, props, names, all, force)
	if err != nil {
		return fmt.Errorf("failed to resolve pull targets: %w", err)
	}
	for _, ref := range targets {
		if err := registry.Pull(ctx, cm, ref, "", force, progress); err != nil {
			return fmt.Errorf("failed to pull image '%s': %w", ref, err)
		}
	}
	return nil
}

// RegistryRemove removes the given image names from the local container manager.
// Names may be comma-separated and are split before resolution.
// If all is true, all locally present otter images are removed.
// force removes even if the image is in use.
func RegistryRemove(
	ctx context.Context,
	cm containermanager.ContainerManager,
	props *registry.ImagesProperties,
	names []string,
	all bool,
	force bool,
) error {
	targets, err := resolveRemoveTargets(ctx, cm, props, names, all)
	if err != nil {
		return fmt.Errorf("failed to resolve remove targets: %w", err)
	}
	for _, ref := range targets {
		if !cm.ImageExists(ctx, ref) {
			ui.DefaultLogger.Warn("image '%s' not found locally, skipping", ref)
			continue
		}
		if err := cm.RemoveImage(ctx, ref, force); err != nil {
			return fmt.Errorf("failed to remove image '%s': %w", ref, err)
		}
		ui.DefaultLogger.Info("removed '%s'", ref)
	}
	return nil
}

// resolvePullTargets returns the list of image refs to pull.
func resolvePullTargets(
	ctx context.Context,
	cm containermanager.ContainerManager,
	props *registry.ImagesProperties,
	names []string,
	all bool,
	force bool,
) ([]string, error) {
	if all {
		var refs []string
		for _, entry := range props.Images {
			if !entry.Enabled {
				continue
			}
			ref, err := registry.Resolve(props, entry.Name)
			if err != nil {
				continue
			}
			if !force && cm.ImageExists(ctx, ref) {
				continue
			}
			refs = append(refs, ref)
		}
		return refs, nil
	}

	split := splitNames(names)
	if len(split) == 0 {
		return nil, errors.New("specify at least one image name with --name or use --all")
	}

	refs := make([]string, 0, len(split))
	for _, name := range split {
		ref, err := registry.Resolve(props, name)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve image '%s': %w", name, err)
		}
		refs = append(refs, ref)
	}
	return refs, nil
}

// resolveRemoveTargets returns the list of image refs to remove.
func resolveRemoveTargets(
	ctx context.Context,
	cm containermanager.ContainerManager,
	props *registry.ImagesProperties,
	names []string,
	all bool,
) ([]string, error) {
	if all {
		var refs []string
		for _, entry := range props.Images {
			ref, err := registry.Resolve(props, entry.Name)
			if err != nil {
				continue
			}
			if cm.ImageExists(ctx, ref) {
				refs = append(refs, ref)
			}
		}
		return refs, nil
	}

	split := splitNames(names)
	if len(split) == 0 {
		return nil, errors.New("specify at least one image name with --name or use --all")
	}

	refs := make([]string, 0, len(split))
	for _, name := range split {
		ref, err := registry.Resolve(props, name)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve image '%s': %w", name, err)
		}
		refs = append(refs, ref)
	}
	return refs, nil
}

// splitNames splits a slice of potentially comma-separated name strings
// into individual trimmed name tokens.
func splitNames(names []string) []string {
	var out []string
	for _, n := range names {
		for _, part := range strings.Split(n, ",") {
			part = strings.TrimSpace(part)
			if part != "" {
				out = append(out, part)
			}
		}
	}
	return out
}
