package manifest

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"

	"github.com/BurntSushi/toml"

	"github.com/ferret-linux/otter/internal/userenv"
)

// Additional holds additional container configuration.
type Additional struct {
	Packages []string `toml:"packages"`
	Volumes  []string `toml:"volumes"`
	Flags    []string `toml:"flags"`
}

// Exported holds exported apps and bins configuration.
type Exported struct {
	Apps []string `toml:"apps"`
	Bins []string `toml:"bins"`
	Path string   `toml:"path"`
}

// Hooks holds pre and post init hooks.
type Hooks struct {
	PreInit  []string `toml:"pre-init"`
	PostInit []string `toml:"post-init"`
}

// Settings holds container behaviour settings.
type Settings struct {
	Lock       bool   `toml:"lock"`
	Entry      bool   `toml:"entry"`
	Shell      string `toml:"shell"`
	Rootful    bool   `toml:"rootful"`
	InitSystem bool   `toml:"init-system"`
	Hostname   string `toml:"hostname"`
}

// Hardware holds resource and hardware settings.
type Hardware struct {
	Memory string `toml:"memory"`
	Nvidia bool   `toml:"nvidia"`
	CPU    int    `toml:"cpu"`
}

// Isolation holds namespace isolation settings.
type Isolation struct {
	Netns   bool `toml:"netns"`
	IPC     bool `toml:"ipc"`
	Process bool `toml:"process"`
	Devsys  bool `toml:"devsys"`
	Groups  bool `toml:"groups"`
	All     bool `toml:"all"`
}

// Item represents a single [[container]] entry in the manifest file.
type Item struct {
	Name       string     `toml:"name"`
	Image      string     `toml:"image"`
	Clone      string     `toml:"clone"`
	StartNow   bool       `toml:"start-now"`
	ForcePull  bool       `toml:"force-pull"`
	Include    string     `toml:"include"`
	Additional Additional `toml:"additional"`
	Exported   Exported   `toml:"exported"`
	Hooks      Hooks      `toml:"hooks"`
	Settings   Settings   `toml:"settings"`
	Hardware   Hardware   `toml:"hardware"`
	Isolation  Isolation  `toml:"isolation"`
}

// manifest is the top-level TOML structure.
type manifest struct {
	Containers []Item `toml:"container"`
}

// Parse reads and parses a TOML manifest file from the given filepath or URL.
// It supports 'include' fields to inherit fields from another container in the same file.
// Returns a slice of Item structs representing each [[container]] in the manifest.
func Parse(ctx context.Context, filepath string) ([]Item, error) {
	data, err := readData(ctx, filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest: %w", err)
	}

	var m manifest
	if err := toml.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("failed to parse TOML: %w", err)
	}

	if err := resolveIncludes(m.Containers); err != nil {
		return nil, fmt.Errorf("failed to resolve includes: %w", err)
	}

	env := userenv.LoadUserEnvironment(ctx)
	for i := range m.Containers {
		if m.Containers[i].Exported.Path == "" {
			m.Containers[i].Exported.Path = env.Home + "/.local/bin"
		}
	}

	return m.Containers, nil
}

// resolveIncludes merges included container fields into each item that declares include.
// The including item's own non-zero fields take priority over the base.
func resolveIncludes(items []Item) error {
	index := make(map[string]int, len(items))
	for i, item := range items {
		if item.Name == "" {
			return fmt.Errorf("container at index %d has no name", i)
		}
		if _, exists := index[item.Name]; exists {
			return fmt.Errorf("duplicate container name '%s'", item.Name)
		}
		index[item.Name] = i
	}

	processing := make(map[string]bool)
	resolved := make(map[string]bool)

	for i := range items {
		if err := resolveOne(items, index, items[i].Name, processing, resolved); err != nil {
			return err
		}
	}

	return nil
}

func resolveOne(items []Item, index map[string]int, name string, processing, resolved map[string]bool) error {
	if resolved[name] {
		return nil
	}
	if processing[name] {
		return fmt.Errorf("circular include detected: '%s'", name)
	}

	i, ok := index[name]
	if !ok {
		return fmt.Errorf("container '%s' not found", name)
	}

	includeName := items[i].Include
	if includeName == "" {
		resolved[name] = true
		return nil
	}

	if _, ok := index[includeName]; !ok {
		return fmt.Errorf("container '%s' includes '%s' which does not exist", name, includeName)
	}

	processing[name] = true

	if err := resolveOne(items, index, includeName, processing, resolved); err != nil {
		return err
	}

	base := items[index[includeName]]
	items[i] = mergeItems(base, items[i])
	items[i].Include = ""

	processing[name] = false
	resolved[name] = true
	return nil
}

// mergeItems merges base into item, with item's own non-zero values taking priority.
func mergeItems(base, item Item) Item {
	if item.Image == "" {
		item.Image = base.Image
	}
	if item.Clone == "" {
		item.Clone = base.Clone
	}
	if !item.StartNow {
		item.StartNow = base.StartNow
	}
	if !item.ForcePull {
		item.ForcePull = base.ForcePull
	}

	// Additional — append base first, item appended on top
	item.Additional.Packages = mergeSlices(base.Additional.Packages, item.Additional.Packages)
	item.Additional.Volumes = mergeSlices(base.Additional.Volumes, item.Additional.Volumes)
	item.Additional.Flags = mergeSlices(base.Additional.Flags, item.Additional.Flags)

	// Exported
	item.Exported.Apps = mergeSlices(base.Exported.Apps, item.Exported.Apps)
	item.Exported.Bins = mergeSlices(base.Exported.Bins, item.Exported.Bins)
	if item.Exported.Path == "" {
		item.Exported.Path = base.Exported.Path
	}

	// Hooks — append base first
	item.Hooks.PreInit = mergeSlices(base.Hooks.PreInit, item.Hooks.PreInit)
	item.Hooks.PostInit = mergeSlices(base.Hooks.PostInit, item.Hooks.PostInit)

	// Settings — item wins on non-zero
	if !item.Settings.Lock {
		item.Settings.Lock = base.Settings.Lock
	}
	if !item.Settings.Entry {
		item.Settings.Entry = base.Settings.Entry
	}
	if item.Settings.Shell == "" {
		item.Settings.Shell = base.Settings.Shell
	}
	if !item.Settings.Rootful {
		item.Settings.Rootful = base.Settings.Rootful
	}
	if !item.Settings.InitSystem {
		item.Settings.InitSystem = base.Settings.InitSystem
	}
	if item.Settings.Hostname == "" {
		item.Settings.Hostname = base.Settings.Hostname
	}

	// Hardware — item wins on non-zero
	if item.Hardware.Memory == "" {
		item.Hardware.Memory = base.Hardware.Memory
	}
	if !item.Hardware.Nvidia {
		item.Hardware.Nvidia = base.Hardware.Nvidia
	}
	if item.Hardware.CPU == 0 {
		item.Hardware.CPU = base.Hardware.CPU
	}

	// Isolation — item wins on non-zero
	if !item.Isolation.Netns {
		item.Isolation.Netns = base.Isolation.Netns
	}
	if !item.Isolation.IPC {
		item.Isolation.IPC = base.Isolation.IPC
	}
	if !item.Isolation.Process {
		item.Isolation.Process = base.Isolation.Process
	}
	if !item.Isolation.Devsys {
		item.Isolation.Devsys = base.Isolation.Devsys
	}
	if !item.Isolation.Groups {
		item.Isolation.Groups = base.Isolation.Groups
	}
	if !item.Isolation.All {
		item.Isolation.All = base.Isolation.All
	}

	return item
}

// mergeSlices returns base followed by item, deduplicating entries already in base.
func mergeSlices(base, item []string) []string {
	if len(base) == 0 {
		return item
	}
	if len(item) == 0 {
		return base
	}
	seen := make(map[string]bool, len(base))
	for _, v := range base {
		seen[v] = true
	}
	result := make([]string, len(base), len(base)+len(item))
	copy(result, base)
	for _, v := range item {
		if !seen[v] {
			result = append(result, v)
		}
	}
	return result
}

// readData reads data from a local file or a URL.
func readData(ctx context.Context, path string) ([]byte, error) {
	if u, err := url.Parse(path); err == nil && u.Scheme != "" && u.Host != "" {
		return fetchURL(ctx, path)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return data, nil
}

// fetchURL retrieves data from the specified URL.
func fetchURL(ctx context.Context, u string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	var buf bytes.Buffer
	_, err = io.Copy(&buf, resp.Body)

	return buf.Bytes(), err
}
