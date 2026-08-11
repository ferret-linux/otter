package config

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
)

// Source describes where a config field's current effective value comes
// from, for display in the webui Settings page (see internal/webui). It has
// no bearing on LoadValues, which only ever needs the merged result.
type Source int

const (
	// SourceDefault means neither a system nor the user config file
	// defines this key; the value shown is otter's built-in default (see
	// defaults()).
	SourceDefault Source = iota
	// SourceSystem means at least one of the three system-wide files
	// defines this key and the user file does not.
	SourceSystem
	// SourceUser means the user file defines this key and no system file
	// does.
	SourceUser
	// SourceOverride means both a system file and the user file define
	// this key, and the user's effective value differs from the system's
	// effective value.
	SourceOverride
)

// FieldPath identifies one config field by its TOML section and key, e.g.
// {"settings", "rootful"}. It's the unit Settings reads provenance for and
// writes edits to — see SaveUserValue.
type FieldPath struct {
	Section string
	Key     string
}

// FieldInfo is one row of the field table Settings renders: where to find
// the value in fileConfig (Path), and a human label for display.
type FieldInfo struct {
	Path  FieldPath
	Label string
}

// Fields lists every editable/displayable config field, in display order.
// This is the single source of truth Settings uses to render the page and
// to validate incoming field paths on save — see SaveUserValue.
var Fields = []FieldInfo{
	{FieldPath{"container", "name"}, "Default container name"},
	{FieldPath{"container", "hostname"}, "Default hostname"},
	{FieldPath{"images", "default"}, "Default image"},
	{FieldPath{"images", "staleness-warn-threshold"}, "Staleness warn threshold"},
	{FieldPath{"images", "staleness-autopull-threshold"}, "Staleness autopull threshold"},
	{FieldPath{"settings", "shell"}, "Default shell"},
	{FieldPath{"settings", "init-system"}, "Init system by default"},
	{FieldPath{"settings", "rootful"}, "Rootful by default"},
	{FieldPath{"settings", "userns-nolimit"}, "Disable userns limit by default"},
	{FieldPath{"settings", "scripts-dir"}, "Scripts directory"},
	{FieldPath{"preferences", "container-manager"}, "Container manager"},
	{FieldPath{"preferences", "sudo-program"}, "Sudo program"},
	{FieldPath{"preferences", "no-entry"}, "Skip desktop entry by default"},
	{FieldPath{"webui", "bind"}, "WebUI bind address"},
	{FieldPath{"webui", "port"}, "WebUI port"},
	{FieldPath{"webui", "allow-remote"}, "WebUI allow remote"},
	{FieldPath{"webui", "token"}, "WebUI token"},
}

// FieldState is one field's current effective value, its provenance, and
// (when a system file also defines it) the system-layer value, so the
// webui can show both when Source is SourceOverride.
type FieldState struct {
	Path        FieldPath
	Value       string // string form for display; Settings parses back to bool/int as needed on save
	Source      Source
	SystemValue string // only meaningful when Source == SourceOverride or SourceSystem
}

// LoadProvenance reports, per field in Fields, whether the system layer
// (the three non-user paths from GetConfigFilePaths) and the user layer
// each define that key, and what each says — using toml.MetaData.IsDefined
// for presence, since a plain bool/string/int struct field can't
// distinguish "unset" from its zero value on its own.
//
// Each of the four files is decoded separately (unlike LoadValues, which
// decodes all four into one accumulating struct) so presence can be
// checked per file; a field's system-layer definedness is the logical OR
// of that field being defined in any of the three system files, and its
// system-layer value is the *last* system file (in priority order) that
// defines it — matching the override order LoadValues itself uses.
func LoadProvenance() ([]FieldState, error) {
	paths, err := GetConfigFilePaths()
	if err != nil {
		return nil, fmt.Errorf("failed to get config file paths: %w", err)
	}
	if len(paths) == 0 {
		return nil, fmt.Errorf("no config file paths resolved")
	}

	userPath := paths[len(paths)-1]
	systemPaths := paths[:len(paths)-1]

	systemCfg, systemDefined, err := decodeSystemLayer(systemPaths)
	if err != nil {
		return nil, err
	}

	var userCfg fileConfig
	userDefined := map[FieldPath]bool{}
	if data, rerr := os.ReadFile(userPath); rerr == nil {
		meta, derr := toml.Decode(string(data), &userCfg)
		if derr != nil {
			return nil, fmt.Errorf("failed to parse TOML in %q: %w", userPath, derr)
		}
		for _, f := range Fields {
			userDefined[f.Path] = meta.IsDefined(f.Path.Section, f.Path.Key)
		}
	} else if !os.IsNotExist(rerr) {
		return nil, fmt.Errorf("failed to read config file %q: %w", userPath, rerr)
	}

	states := make([]FieldState, 0, len(Fields))
	for _, f := range Fields {
		hasSystem := systemDefined[f.Path]
		hasUser := userDefined[f.Path]

		systemVal := fieldString(systemCfg, f.Path)
		userVal := fieldString(userCfg, f.Path)

		state := FieldState{Path: f.Path}

		switch {
		case hasUser && hasSystem:
			state.Value = userVal
			state.SystemValue = systemVal
			if userVal == systemVal {
				state.Source = SourceUser
			} else {
				state.Source = SourceOverride
			}
		case hasUser:
			state.Value = userVal
			state.Source = SourceUser
		case hasSystem:
			state.Value = systemVal
			state.SystemValue = systemVal
			state.Source = SourceSystem
		default:
			state.Value = fieldString(defaults(), f.Path)
			state.Source = SourceDefault
		}

		states = append(states, state)
	}

	return states, nil
}

// decodeSystemLayer decodes the three system-wide config files in priority
// order into one accumulating fileConfig (matching LoadValues' own
// layering, so the returned value reflects the same "effective system
// value" a plain LoadValues call would see if the user file didn't exist),
// and separately tracks, per field, whether *any* of the three files
// defines it — checked independently per file since toml.MetaData doesn't
// support merging two instances into one.
func decodeSystemLayer(paths []string) (fileConfig, map[FieldPath]bool, error) {
	var cfg fileConfig
	defined := map[FieldPath]bool{}

	for _, p := range paths {
		data, err := os.ReadFile(p)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return cfg, nil, fmt.Errorf("failed to read config file %q: %w", p, err)
		}
		meta, err := toml.Decode(string(data), &cfg)
		if err != nil {
			return cfg, nil, fmt.Errorf("failed to parse TOML in %q: %w", p, err)
		}
		for _, f := range Fields {
			if meta.IsDefined(f.Path.Section, f.Path.Key) {
				defined[f.Path] = true
			}
		}
	}

	return cfg, defined, nil
}

// fieldString reads one field out of cfg by its TOML path and formats it
// as a string for display, regardless of its underlying Go type.
func fieldString(cfg fileConfig, path FieldPath) string {
	switch path.Section {
	case "container":
		switch path.Key {
		case "hostname":
			return cfg.Container.Hostname
		case "name":
			return cfg.Container.Name
		}
	case "images":
		switch path.Key {
		case "default":
			return cfg.Images.Default
		case "staleness-warn-threshold":
			return fmt.Sprintf("%d", cfg.Images.StalenessWarnThreshold)
		case "staleness-autopull-threshold":
			return fmt.Sprintf("%d", cfg.Images.StalenessAutopullThreshold)
		}
	case "settings":
		switch path.Key {
		case "shell":
			return cfg.Settings.Shell
		case "init-system":
			return fmt.Sprintf("%t", cfg.Settings.InitSystem)
		case "rootful":
			return fmt.Sprintf("%t", cfg.Settings.Rootful)
		case "userns-nolimit":
			return fmt.Sprintf("%t", cfg.Settings.UsernsNoLimit)
		case "scripts-dir":
			return cfg.Settings.ScriptsDir
		}
	case "preferences":
		switch path.Key {
		case "container-manager":
			return cfg.Preferences.ContainerManager
		case "sudo-program":
			return cfg.Preferences.SudoProgram
		case "no-entry":
			return fmt.Sprintf("%t", cfg.Preferences.NoEntry)
		}
	case "webui":
		switch path.Key {
		case "bind":
			return cfg.WebUI.Bind
		case "port":
			return fmt.Sprintf("%d", cfg.WebUI.Port)
		case "allow-remote":
			return fmt.Sprintf("%t", cfg.WebUI.AllowRemote)
		case "token":
			return cfg.WebUI.Token
		}
	}
	return ""
}
