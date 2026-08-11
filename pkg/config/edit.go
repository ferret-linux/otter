package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

const userConfigFileMode = 0o600

// sectionHeaderPattern matches a TOML table header line like "[settings]",
// capturing the section name. It intentionally does not match dotted or
// quoted table names ("[a.b]", "[\"a\"]") — Fields only ever uses plain
// single-word section names, so anything more exotic in a hand-edited file
// is left alone rather than misparsed.
var sectionHeaderPattern = regexp.MustCompile(`^\[([A-Za-z0-9_-]+)\]\s*(#.*)?$`)

// keyLinePattern matches a "key = value" line, capturing the key and
// requiring the key to be a plain bare TOML key (matching what Fields
// uses) so quoted or dotted keys in a hand-edited file aren't misidentified
// as one of our known fields.
var keyLinePattern = regexp.MustCompile(`^([A-Za-z0-9_-]+)\s*=`)

// SaveUserValue sets one field's value in the user's own config file (the
// last path from GetConfigFilePaths), creating the file and any missing
// parent directory if needed, while leaving every other line in the file
// untouched — including comments and formatting — since this rewrites only
// the one matching line (or appends one) rather than decoding and
// re-encoding the whole file.
//
// value is the field's new value already formatted as it should appear in
// TOML (a quoted string, or a bare "true"/"false"/number) — see
// FormatTOMLValue.
func SaveUserValue(path FieldPath, value string) error {
	if !isKnownField(path) {
		return fmt.Errorf("unknown config field %s.%s", path.Section, path.Key)
	}

	paths, err := GetConfigFilePaths()
	if err != nil {
		return fmt.Errorf("failed to get config file paths: %w", err)
	}
	userPath := paths[len(paths)-1]

	original, err := os.ReadFile(userPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("failed to read config file %q: %w", userPath, err)
		}
		original = nil
	}

	updated := setKeyInSection(string(original), path.Section, path.Key, value)

	if err := os.MkdirAll(filepath.Dir(userPath), 0o700); err != nil {
		return fmt.Errorf("failed to create config directory for %q: %w", userPath, err)
	}
	if err := os.WriteFile(userPath, []byte(updated), userConfigFileMode); err != nil {
		return fmt.Errorf("failed to write config file %q: %w", userPath, err)
	}

	return nil
}

// isKnownField reports whether path is one of the fields Settings knows
// about (see Fields), so SaveUserValue never writes an arbitrary
// caller-supplied section/key into the user's config file.
func isKnownField(path FieldPath) bool {
	for _, f := range Fields {
		if f.Path == path {
			return true
		}
	}
	return false
}

// setKeyInSection returns content with key's value set to value inside
// [section], handling four cases:
//  1. the section and key both already exist: that line's value is
//     replaced in place, its surrounding lines (including comments)
//     untouched.
//  2. the section exists but the key doesn't: a new "key = value" line is
//     appended right before the section's next header (or at the file's
//     end if it's the last section).
//  3. the section doesn't exist at all: a new "[section]\nkey = value\n"
//     block is appended at the end of the file.
//  4. content is empty: same as case 3, producing a minimal new file.
func setKeyInSection(content, section, key, value string) string {
	newLine := fmt.Sprintf("%s = %s", key, value)

	if strings.TrimSpace(content) == "" {
		return fmt.Sprintf("[%s]\n%s\n", section, newLine)
	}

	lines := strings.Split(content, "\n")
	sectionStart := -1
	sectionEnd := len(lines) // exclusive; line index where the section's content ends
	keyLineIdx := -1

	for i, line := range lines {
		if m := sectionHeaderPattern.FindStringSubmatch(line); m != nil {
			if sectionStart != -1 {
				// We've reached the section after ours; its content ends here.
				sectionEnd = i
				break
			}
			if m[1] == section {
				sectionStart = i
			}
			continue
		}
		if sectionStart != -1 && keyLineIdx == -1 {
			if m := keyLinePattern.FindStringSubmatch(line); m != nil && m[1] == key {
				keyLineIdx = i
			}
		}
	}

	switch {
	case sectionStart == -1:
		// Case 3/4: section doesn't exist, append a new block. Ensure the
		// file ends with exactly one newline before appending so the new
		// section doesn't run onto an existing line.
		joined := strings.TrimRight(content, "\n")
		return joined + fmt.Sprintf("\n\n[%s]\n%s\n", section, newLine)

	case keyLineIdx != -1:
		// Case 1: replace the existing key line, preserving anything else
		// on that line after the value would normally be malformed TOML
		// anyway (inline comments after a value are valid TOML, but since
		// we don't parse the old value we can't safely preserve a
		// trailing "# comment" without risking it not being one — so a
		// plain replacement is the correct, unsurprising behavior here).
		lines[keyLineIdx] = newLine
		return strings.Join(lines, "\n")

	default:
		// Case 2: section exists, key doesn't. Insert the new line right
		// before sectionEnd, which is either the next section header or
		// end-of-file.
		insertAt := sectionEnd
		// Walk backward from insertAt to skip trailing blank lines within
		// this section, so the new key lands with the rest of the
		// section's keys rather than after a deliberate blank separator.
		for insertAt > sectionStart+1 && strings.TrimSpace(lines[insertAt-1]) == "" {
			insertAt--
		}
		out := make([]string, 0, len(lines)+1)
		out = append(out, lines[:insertAt]...)
		out = append(out, newLine)
		out = append(out, lines[insertAt:]...)
		return strings.Join(out, "\n")
	}
}

// FormatTOMLValue formats a raw form value as TOML source for the given
// field's type, inferred from Fields via the same type switch fieldString
// uses to read values, so SaveUserValue's caller (the webui Settings
// handler) doesn't need its own copy of which fields are bool/int/string.
func FormatTOMLValue(path FieldPath, raw string) (string, error) {
	switch kindOfField(path) {
	case fieldKindBool:
		switch raw {
		case "true", "on", "1":
			return "true", nil
		case "false", "off", "", "0":
			return "false", nil
		default:
			return "", fmt.Errorf("invalid boolean value %q for %s.%s", raw, path.Section, path.Key)
		}
	case fieldKindInt:
		n, err := strconv.Atoi(strings.TrimSpace(raw))
		if err != nil {
			return "", fmt.Errorf("invalid integer value %q for %s.%s: %w", raw, path.Section, path.Key, err)
		}
		return strconv.Itoa(n), nil
	default: // fieldKindString
		return strconv.Quote(raw), nil
	}
}

type fieldKind int

const (
	fieldKindString fieldKind = iota
	fieldKindBool
	fieldKindInt
)

// kindOfField reports how a field's value should be formatted in TOML,
// mirroring the Go types declared in types.go (fileConfig's nested
// structs) for exactly the fields listed in Fields.
func kindOfField(path FieldPath) fieldKind {
	switch path {
	case FieldPath{"settings", "init-system"},
		FieldPath{"settings", "rootful"},
		FieldPath{"settings", "userns-nolimit"},
		FieldPath{"preferences", "no-entry"},
		FieldPath{"webui", "allow-remote"}:
		return fieldKindBool
	case FieldPath{"images", "staleness-warn-threshold"},
		FieldPath{"images", "staleness-autopull-threshold"},
		FieldPath{"webui", "port"}:
		return fieldKindInt
	default:
		return fieldKindString
	}
}
