package registry

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

)

const propertiesURL = "https://raw.githubusercontent.com/ferret-linux/otter/stable/images/images-properties.json"

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
func Fetch(ctx context.Context) (*ImagesProperties, error) {
	client := &http.Client{Timeout: 10 * time.Second}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, propertiesURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	resp, err := client.Do(req)
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
