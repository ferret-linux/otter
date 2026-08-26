package config

type ContainerConfig struct {
	Hostname string `toml:"hostname"`
	Name     string `toml:"name"`
}

type ImagesConfig struct {
	Default                    string `toml:"default"`
	StalenessWarnThreshold     int    `toml:"staleness-warn-threshold"`
	StalenessAutopullThreshold int    `toml:"staleness-autopull-threshold"`
}

type SettingsConfig struct {
	Shell         string `toml:"shell"`
	InitSystem    bool   `toml:"init-system"`
	Rootful       bool   `toml:"rootful"`
	UsernsNoLimit bool   `toml:"userns-nolimit"`
	ScriptsDir    string `toml:"scripts-dir"`
}

type PreferencesConfig struct {
	ContainerManager string `toml:"container-manager"`
	SudoProgram      string `toml:"sudo-program"`
	NoEntry          bool   `toml:"no-entry"`
}

// FileConfig is the raw, on-disk shape of otter.conf, exported so
// pkg/commands' settings TUI can load, edit, and write back the same
// struct the file loader itself merges — see LoadFileConfig/SaveFileConfig
// in loader.go/writer.go. This is deliberately distinct from Values: a
// Values is a DERIVED/resolved form (e.g. ScriptsDir applies a fallback in
// toValues), so editing it directly would risk baking resolved defaults
// into the user's file as if they had been explicitly set.
type FileConfig struct {
	Container   ContainerConfig   `toml:"container"`
	Images      ImagesConfig      `toml:"images"`
	Settings    SettingsConfig    `toml:"settings"`
	Preferences PreferencesConfig `toml:"preferences"`
}

type Values struct {
	ContainerManagerType       string
	SudoProgram                string
	DefaultContainerImage      string
	DefaultContainerName       string
	DefaultHostname            string
	DefaultShell               string
	DefaultInitSystem          bool
	DefaultRootful             bool
	DefaultNoEntry             bool
	DefaultUsernsNoLimit       bool
	ScriptsDir                 string
	StalenessWarnThreshold     int
	StalenessAutopullThreshold int
}
