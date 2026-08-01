package config

type containerConfig struct {
	Hostname string `toml:"hostname"`
	Name     string `toml:"name"`
}

type imagesConfig struct {
	Default                    string `toml:"default"`
	StalenessWarnThreshold     int    `toml:"staleness-warn-threshold"`
	StalenessAutopullThreshold int    `toml:"staleness-autopull-threshold"`
}

type settingsConfig struct {
	Shell         string `toml:"shell"`
	InitSystem    bool   `toml:"init-system"`
	Rootful       bool   `toml:"rootful"`
	UsernsNoLimit bool   `toml:"userns-nolimit"`
	ScriptsDir    string `toml:"scripts-dir"`
}

type preferencesConfig struct {
	ContainerManager string `toml:"container-manager"`
	SudoProgram      string `toml:"sudo-program"`
	NoEntry          bool   `toml:"no-entry"`
}

type fileConfig struct {
	Container   containerConfig   `toml:"container"`
	Images      imagesConfig      `toml:"images"`
	Settings    settingsConfig    `toml:"settings"`
	Preferences preferencesConfig `toml:"preferences"`
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
