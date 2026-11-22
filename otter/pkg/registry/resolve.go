package registry

import (
	"errors"
	"strings"
)

// ErrUnreliableUnavailable is returned when an official image is currently
// disabled and its registry entry has no reliable vendor fallback. It is
// handled once at the CLI boundary so the user sees a single clear message.
//
//nolint:revive,staticcheck // exact user-facing message is part of the CLI contract
var ErrUnreliableUnavailable = errors.New("image is currently unavailable and has no reliable fallback; please wait for the distro to return.")

// canonical variants. "generic" is runtime-only and never appears in props.
const (
	VariantGeneric  = "generic"
	VariantFree     = "free"
	VariantNonfree  = "nonfree"
	VariantNegativo = "negativo17"
	officialPrefix  = "ghcr.io/ferret-linux/"
)

// ResolveResult carries the effective image reference and the metadata needed
// for first-time setup. Props is non-nil only for images that belong to the
// Otter registry ("ours").
type ResolveResult struct {
	Image    string
	Variant  string
	Family   string
	Reliable bool
	Ours     bool
	Props    *ImagesProperties
}

// ResolveEntry resolves the effective image ref and its variant metadata.
//
// Rules:
//   - short name found in props                  -> ours, full metadata
//   - exact ghcr.io/ferret-linux official_image  -> ours, exact ref preserved
//   - everything else (other full refs, clones,
//     unknown or empty names)                    -> generic
//
// A disabled ours entry with no fallback_vendor_image is an error.
func ResolveEntry(props *ImagesProperties, name string) (ResolveResult, error) {
	generic := ResolveResult{Variant: VariantGeneric, Reliable: true}
	if name == "" {
		return generic, nil
	}
	if strings.ContainsAny(name, "/:") {
		return resolveFullRef(props, name, generic)
	}
	return resolveShortName(props, name, generic)
}

func resolveFullRef(props *ImagesProperties, name string, generic ResolveResult) (ResolveResult, error) {
	if props != nil {
		for _, entry := range props.Images {
			if entry.OfficialImage == name {
				return entryResult(entry, true, props)
			}
		}
	}
	generic.Image = name
	return generic, nil
}

func resolveShortName(props *ImagesProperties, name string, generic ResolveResult) (ResolveResult, error) {
	if props != nil {
		wanted := strings.ToLower(name)
		for _, entry := range props.Images {
			if entry.Name == wanted {
				return entryResult(entry, entry.Enabled || entry.IsReliableFallback, props)
			}
		}
	}
	// Unknown short names are valid generic image references too.
	generic.Image = name
	return generic, nil
}

func entryResult(entry ImageEntry, reliable bool, props *ImagesProperties) (ResolveResult, error) {
	image, err := imageForEntry(entry)
	if err != nil {
		return ResolveResult{}, err
	}
	return ResolveResult{
		Image:    image,
		Variant:  normalizeVariant(entry.ImageVariant),
		Family:   entry.Family,
		Reliable: reliable,
		Ours:     true,
		Props:    props,
	}, nil
}

// imageForEntry applies the enabled/fallback rule without any warning or
// degradation branch. Unreliable disabled entries have no fallback at all.
func imageForEntry(entry ImageEntry) (string, error) {
	if entry.Enabled {
		if entry.OfficialImage != "" {
			return entry.OfficialImage, nil
		}
		return officialPrefix + entry.Name + ":latest", nil
	}
	if entry.FallbackVendorImage != "" {
		return entry.FallbackVendorImage, nil
	}
	return "", ErrUnreliableUnavailable
}

func normalizeVariant(v string) string {
	switch v {
	case VariantFree, VariantNonfree, VariantNegativo:
		return v
	default:
		return VariantGeneric
	}
}

// Resolve returns only the effective image ref. It is used by registry pull
// and remove, which do not need variant metadata.
func Resolve(props *ImagesProperties, name string) (string, error) {
	res, err := ResolveEntry(props, name)
	if err != nil {
		return "", err
	}
	return res.Image, nil
}
