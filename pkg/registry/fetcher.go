package registry

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/ferret-linux/otter/pkg/netcheck"
)

const propertiesURL = "https://raw.githubusercontent.com/ferret-linux/otter/main/images/images-properties.json"

// ImageSize holds per-architecture size data for an image.
type ImageSize struct {
	CompressedSize int64 `json:"compressed_size"`
	DiskSize       int64 `json:"disk_size"`
}

// ImageEntry represents a single image entry from images-properties.json.
type ImageEntry struct {
	Name                string               `json:"name"`
	OfficialImage       string               `json:"official_image"`
	FallbackVendorImage string               `json:"fallback_vendor_image"`
	Architecture        []string             `json:"architecture"`
	Enabled             bool                 `json:"enabled"`
	BuiltAt             string               `json:"built_at"`
	Sizes               map[string]ImageSize `json:"sizes"`
}

// ImagesProperties is the top-level structure of images-properties.json.
type ImagesProperties struct {
	ImagesAvailable bool         `json:"images_available"`
	Images          []ImageEntry `json:"images"`
}

// Fetch retrieves and parses images-properties.json from the upstream repository.
// It calls netcheck before making any network request.
func Fetch() (*ImagesProperties, error) {
	if err := netcheck.Check(); err != nil {
		return nil, fmt.Errorf("registry unavailable: %w", err)
	}

	client := &http.Client{Timeout: 10 * time.Second}

	resp, err := client.Get(propertiesURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch image properties: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status fetching image properties: %d", resp.StatusCode)
	}

	var props ImagesProperties
	if err := json.NewDecoder(resp.Body).Decode(&props); err != nil {
		return nil, fmt.Errorf("failed to parse image properties: %w", err)
	}

	return &props, nil
}
