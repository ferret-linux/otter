package manifest

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
	Netns         bool `toml:"netns"`
	IPC           bool `toml:"ipc"`
	Process       bool `toml:"process"`
	Devsys        bool `toml:"devsys"`
	Groups        bool `toml:"groups"`
	All           bool `toml:"all"`
	UsernsNoLimit bool `toml:"userns-nolimit"`
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
