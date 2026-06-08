package registry

import (
	"errors"
	"fmt"
	"strings"
)

// ErrUnknownImage is returned when a name cannot be found in images-properties.json.
var ErrUnknownImage = errors.New("unknown image")

// Resolve returns the correct image ref for the given name.
//
// If the name already contains '/' or ':', it is a full registry path and is
// returned as-is. Otherwise the name is looked up in props:
//   - images_available false  → fallback_vendor_image
//   - entry enabled false     → fallback_vendor_image
//   - otherwise               → official_image
//
// If the name is not found, ErrUnknownImage is returned along with a message
// listing all currently valid (enabled) image names.
func Resolve(props *ImagesProperties, name string) (string, error) {
	if strings.ContainsAny(name, "/:") {
		return name, nil
	}

	lower := strings.ToLower(name)

	for _, entry := range props.Images {
		if entry.Name != lower {
			continue
		}
		if !props.ImagesAvailable || !entry.Enabled {
			return entry.FallbackVendorImage, nil
		}
		return entry.OfficialImage, nil
	}

	return "", fmt.Errorf("%w %q\n\nvalid images:\n  %s",
		ErrUnknownImage, name, validNames(props))
}

// validNames returns a comma-separated list of enabled image names from props.
func validNames(props *ImagesProperties) string {
	names := make([]string, 0, len(props.Images))
	for _, entry := range props.Images {
		if entry.Enabled {
			names = append(names, entry.Name)
		}
	}
	return strings.Join(names, ", ")
}
