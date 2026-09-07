package registry

import (
	"context"
	"strconv"

	"github.com/ferret-linux/otter/pkg/containermanager"
)

// buildLabelKey is the OCI label baked into every official otter image,
// carrying a monotonically increasing build number (see images/*.Containerfile).
const buildLabelKey = "otter.image_build"

// StalenessState describes how a locally pulled image compares to the
// latest known upstream build for the same registry entry.
type StalenessState int

const (
	// StalenessNotOtterImage means imageRef does not match any entry's
	// official_image, so no build comparison applies (raw/custom refs,
	// --clone commit tags, fallback vendor images).
	StalenessNotOtterImage StalenessState = iota
	// StalenessUnknown means the local otter.image_build label is missing
	// or unreadable.
	StalenessUnknown
	// StalenessCurrent means the local build matches the latest remote build.
	StalenessCurrent
	// StalenessBehind means the remote build is ahead of the local build.
	StalenessBehind
	// StalenessAhead means the local build is ahead of the remote build
	// (not reachable via normal use; warn only, never auto-pull).
	StalenessAhead
)

// Staleness holds the result of comparing a local image against the latest
// known remote build for its registry entry.
type Staleness struct {
	State       StalenessState
	LocalBuild  int
	RemoteBuild int
	Diff        int // RemoteBuild - LocalBuild
}

// CheckStaleness compares imageRef's locally installed otter.image_build
// label against the remote build_number recorded in props for the matching
// official_image entry.
func CheckStaleness(
	ctx context.Context,
	cm containermanager.ContainerManager,
	props *ImagesProperties,
	imageRef string,
) Staleness {
	var entry *ImageEntry
	for i := range props.Images {
		if props.Images[i].OfficialImage == imageRef {
			entry = &props.Images[i]
			break
		}
	}
	if entry == nil {
		return Staleness{State: StalenessNotOtterImage}
	}

	localValue, ok := cm.ImageLabel(ctx, imageRef, buildLabelKey)
	if !ok {
		return Staleness{State: StalenessUnknown, RemoteBuild: entry.BuildNumber}
	}

	localBuild, err := strconv.Atoi(localValue)
	if err != nil {
		return Staleness{State: StalenessUnknown, RemoteBuild: entry.BuildNumber}
	}

	diff := entry.BuildNumber - localBuild
	state := StalenessCurrent
	switch {
	case diff > 0:
		state = StalenessBehind
	case diff < 0:
		state = StalenessAhead
	}

	return Staleness{
		State:       state,
		LocalBuild:  localBuild,
		RemoteBuild: entry.BuildNumber,
		Diff:        diff,
	}
}
