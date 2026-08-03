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
		return "unknown" //nolint:goconst // trivial literal, not worth a shared constant
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

type RegistryListOptions struct {
	All bool
}

type RegistryListCommand struct {
	containerManager containermanager.ContainerManager
}

func NewRegistryListCommand(cm containermanager.ContainerManager) *RegistryListCommand {
	return &RegistryListCommand{containerManager: cm}
}

func (c *RegistryListCommand) Execute(ctx context.Context, opts RegistryListOptions) error {
	props, err := registry.Fetch(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch registry: %w", err)
	}
	registryList(ctx, c.containerManager, props, opts.All)
	return nil
}

type RegistryPullOptions struct {
	Names []string
	All   bool
	Force bool
}

type RegistryPullCommand struct {
	containerManager containermanager.ContainerManager
	progress         *ui.Progress
}

func NewRegistryPullCommand(cm containermanager.ContainerManager) *RegistryPullCommand {
	return &RegistryPullCommand{
		containerManager: cm,
		progress:         ui.NewProgress(os.Stderr),
	}
}

func (c *RegistryPullCommand) Execute(ctx context.Context, opts RegistryPullOptions) error {
	props, err := registry.Fetch(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch registry: %w", err)
	}
	targets, err := resolvePullTargets(ctx, c.containerManager, props, opts.Names, opts.All, opts.Force)
	if err != nil {
		return fmt.Errorf("failed to resolve pull targets: %w", err)
	}
	for _, ref := range targets {
		if err := registry.Pull(ctx, c.containerManager, ref, "", opts.Force, c.progress); err != nil {
			return fmt.Errorf("failed to pull image '%s': %w", ref, err)
		}
	}
	return nil
}

type RegistryRemoveOptions struct {
	Names []string
	All   bool
	Force bool
}

type RegistryRemoveCommand struct {
	containerManager containermanager.ContainerManager
}

func NewRegistryRemoveCommand(cm containermanager.ContainerManager) *RegistryRemoveCommand {
	return &RegistryRemoveCommand{
		containerManager: cm,
	}
}

func (c *RegistryRemoveCommand) Execute(ctx context.Context, opts RegistryRemoveOptions) error {
	props, err := registry.Fetch(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch registry: %w", err)
	}
	targets, err := resolveRemoveTargets(ctx, c.containerManager, props, opts.Names, opts.All)
	if err != nil {
		return fmt.Errorf("failed to resolve remove targets: %w", err)
	}
	for _, ref := range targets {
		if !c.containerManager.ImageExists(ctx, ref) {
			ui.DefaultLogger.Warn("image '%s' not found locally, skipping", ref)
			continue
		}
		if err := c.containerManager.RemoveImage(ctx, ref, opts.Force); err != nil {
			return fmt.Errorf("failed to remove image '%s': %w", ref, err)
		}
		ui.DefaultLogger.Info("removed '%s'", ref)
	}
	return nil
}

// registryList renders a table of available images from props.
// If all is false, disabled images are omitted and STATUS/BUILT columns are hidden.
// A LOCAL column is shown for enabled entries, reflecting whether the image
// is pulled and, if so, how it compares to the latest known remote build.
func registryList(ctx context.Context, cm containermanager.ContainerManager, props *registry.ImagesProperties, all bool) {
	var t *ui.Table
	if all {
		t = ui.NewTable(os.Stdout, "NAME", "ARCH", "STATUS", "BUILT", "LOCAL", "IMAGE")
	} else {
		t = ui.NewTable(os.Stdout, "NAME", "ARCH", "LOCAL", "IMAGE")
	}

	for _, entry := range props.Images {
		if !all && !entry.Enabled {
			continue
		}

		status := "enabled"
		statusColor := ui.Green
		imageRef := entry.OfficialImage

		if !entry.Enabled {
			status = "disabled"
			statusColor = ui.Yellow
			imageRef = entry.FallbackVendorImage
		}

		local, localColor := "", ui.Dim
		if entry.Enabled {
			local, localColor = localStatus(ctx, cm, props, imageRef)
		}

		arch := strings.Join(entry.Architecture, ", ")
		imageRef = ui.TrimImageRef(imageRef)

		if all {
			t.AddRow(
				[]string{entry.Name, arch, status, relativeTime(entry.BuiltAt), local, imageRef},
				[]func(string) string{ui.Teal, ui.Dim, statusColor, ui.Dim, localColor, ui.Dim},
			)
		} else {
			t.AddRow(
				[]string{entry.Name, arch, local, imageRef},
				[]func(string) string{ui.Teal, ui.Dim, localColor, ui.Dim},
			)
		}
	}
	t.Render()
}

// localStatus returns a human-readable LOCAL column value and its color for
// an enabled registry entry's official image ref.
func localStatus(
	ctx context.Context,
	cm containermanager.ContainerManager,
	props *registry.ImagesProperties,
	imageRef string,
) (string, func(string) string) {
	if !cm.ImageExists(ctx, imageRef) {
		return "not pulled", ui.Dim
	}

	st := registry.CheckStaleness(ctx, cm, props, imageRef)
	switch st.State {
	case registry.StalenessCurrent:
		return "up to date", ui.Green
	case registry.StalenessBehind:
		return fmt.Sprintf("%d behind", st.Diff), ui.Yellow
	case registry.StalenessAhead:
		return "ahead", ui.Yellow
	case registry.StalenessUnknown, registry.StalenessNotOtterImage:
		return "unknown", ui.Dim
	default:
		return "unknown", ui.Dim
	}
}

// resolvePullCandidates returns the set of image refs eligible for pulling,
// before any local-existence or staleness filtering is applied: either all
// enabled registry entries, or the explicitly named ones.
func resolvePullCandidates(props *registry.ImagesProperties, names []string, all bool) ([]string, error) {
	if all {
		var candidates []string
		for _, entry := range props.Images {
			if !entry.Enabled {
				continue
			}
			ref, err := registry.Resolve(props, entry.Name)
			if err != nil {
				continue
			}
			candidates = append(candidates, ref)
		}
		return candidates, nil
	}

	if len(names) == 0 {
		return nil, errors.New("specify at least one image name or use --all")
	}
	candidates := make([]string, 0, len(names))
	for _, name := range names {
		ref, err := registry.Resolve(props, name)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve image '%s': %w", name, err)
		}
		candidates = append(candidates, ref)
	}
	return candidates, nil
}

// resolvePullTargets returns the list of image refs to pull. A ref is
// skipped only if it already exists locally and is not stale; --force
// bypasses this check entirely. This applies uniformly whether targets come
// from --all or from explicitly named images.
func resolvePullTargets(
	ctx context.Context,
	cm containermanager.ContainerManager,
	props *registry.ImagesProperties,
	names []string,
	all bool,
	force bool,
) ([]string, error) {
	candidates, err := resolvePullCandidates(props, names, all)
	if err != nil {
		return nil, err
	}

	refs := make([]string, 0, len(candidates))
	for _, ref := range candidates {
		if !force && cm.ImageExists(ctx, ref) {
			st := registry.CheckStaleness(ctx, cm, props, ref)
			if st.State != registry.StalenessBehind && st.State != registry.StalenessUnknown {
				ui.DefaultLogger.Info("skipping '%s', already up to date", ref)
				continue
			}
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
			if !entry.Enabled {
				continue
			}
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

	if len(names) == 0 {
		return nil, errors.New("specify at least one image name or use --all")
	}

	refs := make([]string, 0, len(names))
	for _, name := range names {
		ref, err := registry.Resolve(props, name)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve image '%s': %w", name, err)
		}
		refs = append(refs, ref)
	}
	return refs, nil
}
