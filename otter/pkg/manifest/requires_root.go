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

	for _, name := range selected {
		item, ok := byName[name]
		if !ok {
			return false, fmt.Errorf("box '%s' not found in manifest", name)
		}
		if item.Settings.Rootful != nil && *item.Settings.Rootful ||
			item.Settings.Rootful == nil && defaultRootful {
			return true, nil
		}
	}

	return false, nil
}
