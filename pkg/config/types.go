package config

type containerConfig struct {
	Hostname string `toml:"hostname"`
	Image    string `toml:"image"`
	Name     string `toml:"name"`
}

type settingsConfig struct {
	Shell         string `toml:"shell"`
	InitSystem    bool   `toml:"init-system"`
	Rootful       bool   `toml:"rootful"`
	UsernsNoLimit bool   `toml:"userns-nolimit"`
}

type preferencesConfig struct {
	ContainerManager string `toml:"container-manager"`
	SudoProgram      string `toml:"sudo-program"`
	NoEntry          bool   `toml:"no-entry"`
}

type fileConfig struct {
	Container   containerConfig   `toml:"container"`
	Settings    settingsConfig    `toml:"settings"`
	Preferences preferencesConfig `toml:"preferences"`
}

type Values struct {
	ContainerManagerType  string
	SudoProgram           string
	DefaultContainerImage string
	DefaultContainerName  string
	DefaultHostname       string
	DefaultShell          string
	DefaultInitSystem     bool
	DefaultRootful        bool
	DefaultNoEntry        bool
	DefaultUsernsNoLimit  bool
}
