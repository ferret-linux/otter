package config

import (
	"os"
	"path/filepath"
)

func defaults() fileConfig {
	return fileConfig{
		Container: containerConfig{
			Name: "my-container",
		},
		Images: imagesConfig{
			Default:                    "ghcr.io/ferret-linux/ubuntu-otr:lts",
			StalenessWarnThreshold:     5,
			StalenessAutopullThreshold: 10,
		},
		Preferences: preferencesConfig{
			ContainerManager: "autodetect",
			SudoProgram:      "autodetect",
		},
		WebUI: webUIConfig{
			Bind: "127.0.0.1",
			Port: 8080,
		},
	}
}

func DefaultValues() *Values {
	return toValues(defaults())
}

func toValues(cfg fileConfig) *Values {
	return &Values{
		ContainerManagerType:       cfg.Preferences.ContainerManager,
		SudoProgram:                cfg.Preferences.SudoProgram,
		DefaultContainerImage:      cfg.Images.Default,
		DefaultContainerName:       cfg.Container.Name,
		DefaultHostname:            cfg.Container.Hostname,
		DefaultShell:               cfg.Settings.Shell,
		DefaultInitSystem:          cfg.Settings.InitSystem,
		DefaultRootful:             cfg.Settings.Rootful,
		DefaultNoEntry:             cfg.Preferences.NoEntry,
		DefaultUsernsNoLimit:       cfg.Settings.UsernsNoLimit,
		ScriptsDir:                 resolveScriptsDir(cfg.Settings.ScriptsDir),
		StalenessWarnThreshold:     cfg.Images.StalenessWarnThreshold,
		StalenessAutopullThreshold: cfg.Images.StalenessAutopullThreshold,
		DefaultWebUIBind:           cfg.WebUI.Bind,
		DefaultWebUIPort:           cfg.WebUI.Port,
	}
}

// resolveScriptsDir determines the host directory used to store otter's
// provisioned scripts. The configured value always takes priority; if unset,
// it falls back to $HOME/.local/share/otter. If HOME is also unset, it
// returns an empty string, leaving validation to the caller that actually
// needs the directory (see insidecontainer.ProvisionScripts).
func resolveScriptsDir(configured string) string {
	if configured != "" {
		return configured
	}

	if home := os.Getenv("HOME"); home != "" {
		return filepath.Join(home, ".local", "share", "otter")
	}

	return ""
}
