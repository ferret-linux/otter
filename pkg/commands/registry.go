package commands

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ferret-linux/otter/pkg/containermanager"
	"github.com/ferret-linux/otter/pkg/registry"
	"github.com/ferret-linux/otter/pkg/ui"
)

// blockingContainers returns the names of otter-managed containers (running
// or stopped) whose image resolves to the same local image ID as imageRef.
// Comparison uses ContainerImageID, each container's frozen image binding
// from when it was created, rather than resolving Container.Image (a tag
// string) through ImageID — a tag can move to a different build after the
// container was created (e.g. a later registry pull refresh), and
// re-resolving it would silently report whatever the tag points to now,
// not what the container is actually running on.
func blockingContainers(
	ctx context.Context,
	cm containermanager.ContainerManager,
	containers []containermanager.Container,
	imageRef string,
) ([]string, error) {
	targetID, ok := cm.ImageID(ctx, imageRef)
	if !ok {
		return nil, fmt.Errorf("failed to resolve image ID for '%s'", imageRef)
	}

	var names []string
	for _, c := range containers {
		if !c.IsOtterContainer() {
			continue
		}
		containerImageID, ok := cm.ContainerImageID(ctx, c.Name)
		if !ok {
			continue
		}
		if containerImageID == targetID {
			names = append(names, c.Name)
		}
	}
	return names, nil
}

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
	All  bool
	JSON bool
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
	if opts.JSON {
		return registryListJSON(ctx, c.containerManager, props, opts.All)
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
		if err := registry.Pull(ctx, c.containerManager, ref, "", c.progress); err != nil {
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

	if opts.Force {
		return c.removeTargets(ctx, targets, opts.Force)
	}

	containers, err := c.containerManager.ListContainers(ctx)
	if err != nil {
		return fmt.Errorf("failed to list containers: %w", err)
	}

	removable := make([]string, 0, len(targets))
	blocked := make(map[string][]string)
	for _, ref := range targets {
		names, err := blockingContainers(ctx, c.containerManager, containers, ref)
		if err != nil {
			return fmt.Errorf("failed to check image usage for '%s': %w", ref, err)
		}
		if len(names) > 0 {
			blocked[ref] = names
			continue
		}
		removable = append(removable, ref)
	}

	if len(blocked) == len(targets) && len(targets) > 0 {
		var b strings.Builder
		b.WriteString("all target images are in use, nothing removed:")
		for _, ref := range targets {
			fmt.Fprintf(&b, "\n  %s (in use by: %s)", ref, strings.Join(blocked[ref], ", "))
		}
		return errors.New(b.String())
	}

	for _, ref := range targets {
		if names, ok := blocked[ref]; ok {
			ui.DefaultLogger.Warn("image in use, skipping", "image", ref, "containers", strings.Join(names, ", "))
		}
	}

	return c.removeTargets(ctx, removable, opts.Force)
}

// removeTargets attempts to remove every ref in targets, continuing past
// individual failures so every target is attempted before reporting. If
// anything was skipped-as-missing or failed, a non-nil error is returned
// after all targets have been tried.
func (c *RegistryRemoveCommand) removeTargets(ctx context.Context, targets []string, force bool) error {
	var failed []string
	for _, ref := range targets {
		if !c.containerManager.ImageExists(ctx, ref) {
			ui.DefaultLogger.Warn("image not found locally, skipping", "image", ref)
			continue
		}
		if err := c.containerManager.RemoveImage(ctx, ref, force); err != nil {
			ui.DefaultLogger.Warn("failed to remove image", "image", ref, "error", err)
			failed = append(failed, ref)
			continue
		}
		ui.DefaultLogger.Info("removed", "image", ref)
	}
	if len(failed) > 0 {
		return fmt.Errorf("failed to remove: %s", strings.Join(failed, ", "))
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

// registryListJSON prints props's entries as JSON, with a structured
// pulled/staleness/behind_count breakdown instead of registryList's
// single human-readable LOCAL string, so scripts can branch on state
// without parsing prose. built_at is also left as the raw RFC3339 value
// rather than relativeTime's phrasing, for the same reason.
func registryListJSON(
	ctx context.Context,
	cm containermanager.ContainerManager,
	props *registry.ImagesProperties,
	all bool,
) error {
	type registryEntryJSON struct {
		Name         string   `json:"name"`
		Architecture []string `json:"architecture"`
		Enabled      bool     `json:"enabled"`
		BuiltAt      string   `json:"built_at"`
		Pulled       bool     `json:"pulled"`
		Staleness    string   `json:"staleness,omitempty"`
		BehindCount  int      `json:"behind_count,omitempty"`
		Image        string   `json:"image"`
	}

	out := make([]registryEntryJSON, 0, len(props.Images))
	for _, entry := range props.Images {
		if !all && !entry.Enabled {
			continue
		}

		imageRef := entry.OfficialImage
		if !entry.Enabled {
			imageRef = entry.FallbackVendorImage
		}

		row := registryEntryJSON{
			Name:         entry.Name,
			Architecture: entry.Architecture,
			Enabled:      entry.Enabled,
			BuiltAt:      entry.BuiltAt,
			Image:        ui.TrimImageRef(imageRef),
		}

		if entry.Enabled {
			row.Pulled = cm.ImageExists(ctx, imageRef)
			if !row.Pulled {
				row.Staleness = "not_pulled"
			} else {
				st := registry.CheckStaleness(ctx, cm, props, imageRef)
				switch st.State {
				case registry.StalenessCurrent:
					row.Staleness = "current"
				case registry.StalenessBehind:
					row.Staleness = "behind"
					row.BehindCount = st.Diff
				case registry.StalenessAhead:
					row.Staleness = "ahead"
				default:
					row.Staleness = "unknown"
				}
			}
		}

		out = append(out, row)
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(out); err != nil {
		return fmt.Errorf("failed to encode registry list output as JSON: %w", err)
	}
	return nil
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
				ui.DefaultLogger.Info("skipping, already up to date", "image", ref)
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
