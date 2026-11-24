package manifest

import "fmt"

// RequiresRoot reports whether any selected box will be created in rootful
// mode. An empty boxNames selects every item in the manifest. A nil
// Settings.Rootful means the caller's configured default applies.
func RequiresRoot(items []Item, boxNames []string, defaultRootful bool) (bool, error) {
	selected := boxNames
	if len(selected) == 0 {
		selected = make([]string, len(items))
		for i, item := range items {
			selected[i] = item.Name
		}
	}

	byName := make(map[string]Item, len(items))
	for _, item := range items {
		byName[item.Name] = item
	}

	// Validate the whole selection first. Callers use this helper before
	// creating or removing containers, so an early return on the first
	// rootful box would let later unknown names reach createItem as zero
	// values and crash it.
	for _, name := range selected {
		if _, ok := byName[name]; !ok {
			return false, fmt.Errorf("box '%s' not found in manifest", name)
		}
	}

	for _, name := range selected {
		item := byName[name]
		if item.Settings.Rootful != nil && *item.Settings.Rootful ||
			item.Settings.Rootful == nil && defaultRootful {
			return true, nil
		}
	}

	return false, nil
}
