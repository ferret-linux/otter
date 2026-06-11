package commands

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ferret-linux/otter/pkg/containermanager"
	"github.com/ferret-linux/otter/pkg/registry"
	"github.com/ferret-linux/otter/pkg/ui"
)

// relativeTime returns a human-readable relative time string for an RFC3339 timestamp.
func relativeTime(s string) string {
	if s == "" {
		return "unknown"
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return "unknown"
	}
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		m := int(d.Minutes())
		if m == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", m)
	case d < 24*time.Hour:
		h := int(d.Hours())
		if h == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", h)
	case d < 7*24*time.Hour:
		day := int(d.Hours() / 24)
		if day == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", day)
	case d < 30*24*time.Hour:
		w := int(d.Hours() / 24 / 7)
		if w == 1 {
			return "1 week ago"
		}
		return fmt.Sprintf("%d weeks ago", w)
	case d < 365*24*time.Hour:
		mo := int(d.Hours() / 24 / 30)
		if mo == 1 {
			return "1 month ago"
		}
		return fmt.Sprintf("%d months ago", mo)
	default:
		y := int(d.Hours() / 24 / 365)
		if y == 1 {
			return "1 year ago"
		}
		return fmt.Sprintf("%d years ago", y)
	}
}

// RegistryList renders a table of available images from props.
// If all is false, disabled images are omitted and STATUS/BUILT columns are hidden.
func RegistryList(props *registry.ImagesProperties, all bool) {
	var t *ui.Table
	if all {
		t = ui.NewTable(os.Stdout, "NAME", "ARCH", "STATUS", "BUILT", "IMAGE")
	} else {
		t = ui.NewTable(os.Stdout, "NAME", "ARCH", "IMAGE")
	}

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
		imageRef = ui.TrimImageRef(imageRef)

		if all {
			t.AddRow(
				[]string{entry.Name, arch, status, relativeTime(entry.BuiltAt), imageRef},
				[]func(string) string{ui.Teal, ui.Dim, statusColor, ui.Dim, ui.Dim},
			)
		} else {
			t.AddRow(
				[]string{entry.Name, arch, imageRef},
				[]func(string) string{ui.Teal, ui.Dim, ui.Dim},
			)
		}
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
