package commands

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/ferret-linux/otter/pkg/config"
)

// settingsJSONField is the machine-readable shape of one settings field.
// It mirrors the metadata buildSettingsEntries already carries so the GUI
// can render every field without hardcoding the schema — the same
// --json contract `reg list` and `list` use elsewhere in otter.
type settingsJSONField struct {
	Section     string   `json:"section"`
	Field       string   `json:"field"`
	Kind        string   `json:"kind"` // "text" or "toggle"
	Value       any      `json:"value"`
	Description string   `json:"description"`
	Options     []string `json:"options,omitempty"`
	Numeric     bool     `json:"numeric,omitempty"`
}

// SettingsListJSON prints the merged settings (the same view the settings
// TUI edits, see LoadFileConfig) as a JSON array of fields, keyed for the
// GUI. It is only ever wired to the hidden --list-json flag.
func SettingsListJSON() error {
	cfg, err := config.LoadFileConfig()
	if err != nil {
		return fmt.Errorf("failed to load settings: %w", err)
	}

	fields := make([]settingsJSONField, 0)
	for _, e := range buildSettingsEntries() {
		if e.isSection {
			continue
		}
		f := settingsJSONField{
			Section:     e.section,
			Field:       e.field,
			Description: e.description,
			Numeric:     e.numeric,
			Options:     e.options,
		}
		if e.kind == settingsKindToggle {
			f.Kind = "toggle"
			f.Value = e.getBool(cfg)
		} else {
			f.Kind = "text"
			f.Value = e.getText(cfg)
		}
		fields = append(fields, f)
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(fields)
}

// SettingsApplyJSON accepts a flat map of field name to value on r and
// applies it to the user's own otter.conf via SaveFileConfig. Validation
// mirrors settings_tui.go's commitEdit: text fields with a fixed option set
// must match one of the options, numeric fields must parse as an int.
// Field names are unique across sections, so the flat map needs no section
// prefix. Returned errors reject the whole payload without writing.
func SettingsApplyJSON(r io.Reader) error {
	var updates map[string]any
	if err := json.NewDecoder(r).Decode(&updates); err != nil {
		return fmt.Errorf("failed to decode settings: %w", err)
	}

	cfg, err := config.LoadFileConfig()
	if err != nil {
		return fmt.Errorf("failed to load settings: %w", err)
	}

	byField := make(map[string]settingsEntry)
	for _, e := range buildSettingsEntries() {
		if !e.isSection {
			byField[e.field] = e
		}
	}

	for field, raw := range updates {
		e, ok := byField[field]
		if !ok {
			return fmt.Errorf("unknown settings field %q", field)
		}

		switch e.kind {
		case settingsKindToggle:
			b, ok := raw.(bool)
			if !ok {
				return fmt.Errorf("settings field %q expects a boolean", field)
			}
			e.setBool(cfg, b)

		case settingsKindText:
			val, ok := raw.(string)
			if !ok {
				return fmt.Errorf("settings field %q expects a string", field)
			}
			if e.numeric {
				if _, err := strconv.Atoi(val); err != nil {
					return fmt.Errorf("settings field %q must be a whole number", field)
				}
			}
			if len(e.options) > 0 && !slices.Contains(e.options, val) {
				return fmt.Errorf("settings field %q must be one of: %s", field, strings.Join(e.options, ", "))
			}
			e.setText(cfg, val)
		}
	}

	if err := config.SaveFileConfig(cfg); err != nil {
		return fmt.Errorf("failed to save settings: %w", err)
	}
	fmt.Fprintln(os.Stdout, "ok")
	return nil
}
