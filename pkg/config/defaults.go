package config

func defaults() fileConfig {
	return fileConfig{
		Container: containerConfig{
			Image: "ghcr.io/ferret-linux/ubuntu-otr:lts",
			Name:  "my-container",
		},
		Preferences: preferencesConfig{
			ContainerManager: "autodetect",
			SudoProgram:      "autodetect",
		},
	}
}

func DefaultValues() *Values {
	return toValues(defaults())
}

func toValues(cfg fileConfig) *Values {
	return &Values{
		ContainerManagerType:  cfg.Preferences.ContainerManager,
		SudoProgram:           cfg.Preferences.SudoProgram,
		DefaultContainerImage: cfg.Container.Image,
		DefaultContainerName:  cfg.Container.Name,
		DefaultHostname:       cfg.Container.Hostname,
		DefaultShell:          cfg.Settings.Shell,
		DefaultInitSystem:     cfg.Settings.InitSystem,
		DefaultRootful:        cfg.Settings.Rootful,
		DefaultNoEntry:        cfg.Preferences.NoEntry,
	}
}
