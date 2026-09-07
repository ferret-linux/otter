package commands

import (
	"fmt"
	"io"
	"reflect"
	"slices"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/ferret-linux/otter/pkg/config"

	"github.com/charmbracelet/x/ansi"
)

// colorRed is used only by the settings editor (an invalid field's
// brackets, the save-failed status line), so it lives here rather than
// alongside documentation.go's palette.
//
//nolint:gochecknoglobals // package-level styles, matching documentation.go's convention
var colorRed = lipgloss.Color("1")

// settingsKeyMap defines the keybindings for `otter settings`. Enter is
// deliberately reused for three distinct actions depending on context
// (expand/collapse a section, open a text field for editing or flip a
// toggle, commit an in-progress edit) rather than adding separate bindings
// for each — the same contextual-reuse pattern docKeyMap's Open binding
// already uses in documentation.go.
type settingsKeyMap struct {
	Up         key.Binding
	SwitchPane key.Binding
	Edit       key.Binding
	Cancel     key.Binding
	Save       key.Binding
	Quit       key.Binding
}

//nolint:gochecknoglobals // keymap is the idiomatic bubbles/key pattern
var settingsKeys = settingsKeyMap{
	Up:         key.NewBinding(key.WithKeys("up", "down")),
	SwitchPane: key.NewBinding(key.WithKeys("left", "right")),
	Edit:       key.NewBinding(key.WithKeys("enter")),
	Cancel:     key.NewBinding(key.WithKeys("esc")),
	Save:       key.NewBinding(key.WithKeys("s")),
	Quit:       key.NewBinding(key.WithKeys("q", "ctrl+c")),
}

// settingsFieldKind is one of the two widget kinds the settings editor
// supports — see PROMPT.md's design spec §4: a dedicated third "cycle"
// widget for shell/container-manager was considered and rejected in favor
// of validating those two fields as textinput on commit.
type settingsFieldKind int

const (
	settingsKindToggle settingsFieldKind = iota
	settingsKindText
)

// settingsEntry is one row of the settings tree: either a section header
// (container/images/settings/preferences — selectable, Enter
// expands/collapses it) or a leaf field within a section. getText/setText
// and getBool/setBool read and write the field directly on a
// *config.FileConfig, so the same static entry list can be reused against
// any loaded config without rebuilding it.
type settingsEntry struct {
	isSection bool

	// section is the section header's own label for a section row, or the
	// owning section's label for a leaf row.
	section string

	// field is the leaf's own label (e.g. "hostname"); empty for section
	// header rows.
	field string

	kind settingsFieldKind

	// description is shown in the detail pane, taken from
	// docs/05.configuration.md's reference table.
	description string

	// options, if non-empty, is the fixed set of values a text field is
	// validated against on commit (shell, container-manager). Left empty
	// for fields with no fixed set (hostname, name, scripts-dir,
	// sudo-program — sudo-program's suggestions are not a closed set, see
	// the design spec §3).
	options []string

	// numeric marks a text field that must parse as an int on commit
	// (the two staleness thresholds).
	numeric bool

	getText func(*config.FileConfig) string
	setText func(*config.FileConfig, string)
	getBool func(*config.FileConfig) bool
	setBool func(*config.FileConfig, bool)
}

// buildSettingsEntries returns the fixed, ordered list of every section and
// field the settings editor exposes, matching config.FileConfig's shape
// exactly (see pkg/config/types.go).
func buildSettingsEntries() []settingsEntry {
	return []settingsEntry{
		{isSection: true, section: "container"},
		{
			section: "container", field: "hostname", kind: settingsKindText,
			description: "Default hostname for newly created containers, if --hostname isn't passed.",
			getText:     func(c *config.FileConfig) string { return c.Container.Hostname },
			setText:     func(c *config.FileConfig, v string) { c.Container.Hostname = v },
		},
		{
			section: "container", field: "name", kind: settingsKindText,
			description: "Default container name used when neither a name nor --image is given to otter create.",
			getText:     func(c *config.FileConfig) string { return c.Container.Name },
			setText:     func(c *config.FileConfig, v string) { c.Container.Name = v },
		},

		{isSection: true, section: "images"},
		{
			section: "images", field: "default", kind: settingsKindText,
			description: "Image used by otter create when --image isn't passed. Accepts a short registry name (see otter registry list) or a full image reference.",
			getText:     func(c *config.FileConfig) string { return c.Images.Default },
			setText:     func(c *config.FileConfig, v string) { c.Images.Default = v },
		},
		{
			section: "images", field: "staleness-warn-threshold", kind: settingsKindText, numeric: true,
			description: "Build-count difference behind upstream at which otter warns about a stale local image. 0 disables warning.",
			getText:     func(c *config.FileConfig) string { return strconv.Itoa(c.Images.StalenessWarnThreshold) },
			setText:     func(c *config.FileConfig, v string) { c.Images.StalenessWarnThreshold, _ = strconv.Atoi(v) },
		},
		{
			section: "images", field: "staleness-autopull-threshold", kind: settingsKindText, numeric: true,
			description: "Build-count difference behind upstream at which otter auto-pulls instead of just warning. 0 disables auto-pull.",
			getText:     func(c *config.FileConfig) string { return strconv.Itoa(c.Images.StalenessAutopullThreshold) },
			setText:     func(c *config.FileConfig, v string) { c.Images.StalenessAutopullThreshold, _ = strconv.Atoi(v) },
		},

		{isSection: true, section: "settings"},
		{
			section: "settings", field: "shell", kind: settingsKindText, options: []string{"bash", "zsh", "fish", "nu"},
			description: "Default shell inside new containers: bash, zsh, fish, or nu.",
			getText:     func(c *config.FileConfig) string { return c.Settings.Shell },
			setText:     func(c *config.FileConfig, v string) { c.Settings.Shell = v },
		},
		{
			section: "settings", field: "init-system", kind: settingsKindToggle,
			description: "Enable an init system (systemd) by default, as if --init were always passed.",
			getBool:     func(c *config.FileConfig) bool { return c.Settings.InitSystem },
			setBool:     func(c *config.FileConfig, v bool) { c.Settings.InitSystem = v },
		},
		{
			section: "settings", field: "rootful", kind: settingsKindToggle,
			description: "Run containers as root by default, as if --root were always passed.",
			getBool:     func(c *config.FileConfig) bool { return c.Settings.Rootful },
			setBool:     func(c *config.FileConfig, v bool) { c.Settings.Rootful = v },
		},
		{
			section: "settings", field: "userns-nolimit", kind: settingsKindToggle,
			description: "Disable the rootless uid/gid range limit for user namespaces by default.",
			getBool:     func(c *config.FileConfig) bool { return c.Settings.UsernsNoLimit },
			setBool:     func(c *config.FileConfig, v bool) { c.Settings.UsernsNoLimit = v },
		},
		{
			section: "settings", field: "scripts-dir", kind: settingsKindText,
			description: "Host directory where otter's provisioned helper scripts (otter-init, otter-export, otter-host-exec) are stored.",
			getText:     func(c *config.FileConfig) string { return c.Settings.ScriptsDir },
			setText:     func(c *config.FileConfig, v string) { c.Settings.ScriptsDir = v },
		},

		{isSection: true, section: "preferences"},
		{
			section: "preferences", field: "container-manager", kind: settingsKindText,
			options:     []string{"autodetect", "podman", "docker", "nerdctl"},
			description: "Which container manager to use: autodetect, podman, docker, or nerdctl.",
			getText:     func(c *config.FileConfig) string { return c.Preferences.ContainerManager },
			setText:     func(c *config.FileConfig, v string) { c.Preferences.ContainerManager = v },
		},
		{
			section: "preferences", field: "sudo-program", kind: settingsKindText,
			description: "Privilege-escalation program used for --root operations. autodetect tries sudo, sudo-rs, doas, run0, then pkexec, in that order.",
			getText:     func(c *config.FileConfig) string { return c.Preferences.SudoProgram },
			setText:     func(c *config.FileConfig, v string) { c.Preferences.SudoProgram = v },
		},
		{
			section: "preferences", field: "no-entry", kind: settingsKindToggle,
			description: "Skip desktop entry generation by default, as if --no-entry were always passed to otter create.",
			getBool:     func(c *config.FileConfig) bool { return c.Preferences.NoEntry },
			setBool:     func(c *config.FileConfig, v bool) { c.Preferences.NoEntry = v },
		},
	}
}

// settingsTreeItem adapts a settingsEntry to list.Item.
type settingsTreeItem struct {
	entry settingsEntry
}

func (i settingsTreeItem) FilterValue() string {
	if i.entry.isSection {
		return i.entry.section
	}
	return i.entry.field
}

// settingsTreeItems adapts a slice of settingsEntry into the []list.Item
// shape the list widget expects.
func settingsTreeItems(entries []settingsEntry) []list.Item {
	items := make([]list.Item, len(entries))
	for i, e := range entries {
		items[i] = settingsTreeItem{entry: e}
	}
	return items
}

// visibleSettingsEntries returns the subset of entries that should
// currently be rendered, given which sections are collapsed — every field
// row following a collapsed section header is hidden until that section's
// next header row (mirroring visibleDocEntries, but only ever one level
// deep, since a settings section never nests another section).
func visibleSettingsEntries(entries []settingsEntry, collapsed map[string]bool) []settingsEntry {
	var out []settingsEntry
	skip := false
	for _, e := range entries {
		if e.isSection {
			skip = collapsed[e.section]
			out = append(out, e)
			continue
		}
		if skip {
			continue
		}
		out = append(out, e)
	}
	return out
}

// settingsTreeDelegate renders settingsTreeItem rows, reusing
// documentation.go's tree palette (dirNameStyle, selectedTreeStyle,
// unselectedTreeStyle) so the two TUIs read as the same app.
type settingsTreeDelegate struct {
	collapsed map[string]bool
	// rowWidth is the tree pane's interior width; long section/field
	// labels are ellipsized to fit within it so the border box always
	// matches the width set in setSize (see docTreeDelegate.rowWidth).
	rowWidth int
}

func (d settingsTreeDelegate) Height() int  { return 1 }
func (d settingsTreeDelegate) Spacing() int { return 0 }

func (d settingsTreeDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d settingsTreeDelegate) Render(w io.Writer, m list.Model, index int, it list.Item) {
	ti, ok := it.(settingsTreeItem)
	if !ok {
		return
	}
	e := ti.entry

	if e.isSection {
		nameStyle := dirNameStyle
		if index == m.Index() {
			nameStyle = selectedTreeStyle.Bold(true)
		}
		indicator := "▸ "
		if !d.collapsed[e.section] {
			indicator = "▾ "
		}
		budget := d.rowWidth - lipgloss.Width(indicator)
		label := ansi.Truncate(e.section, budget, "…")
		row := nameStyle.Render(indicator + label)
		if pad := d.rowWidth - lipgloss.Width(row); pad > 0 {
			row += nameStyle.Render(strings.Repeat(" ", pad))
		}
		fmt.Fprint(w, row)
		return
	}

	nameStyle := unselectedTreeStyle
	if index == m.Index() {
		nameStyle = selectedTreeStyle
	}
	const leafPrefix = "  ● "
	budget := d.rowWidth - lipgloss.Width(leafPrefix)
	field := ansi.Truncate(e.field, budget, "…")
	row := leafPrefix + nameStyle.Render(field)
	if pad := d.rowWidth - lipgloss.Width(row); pad > 0 {
		row += nameStyle.Render(strings.Repeat(" ", pad))
	}
	fmt.Fprint(w, row)
}

// settingsFocusPane identifies which pane currently has input focus,
// mirroring docFocusPane.
type settingsFocusPane int

const (
	settingsFocusTree settingsFocusPane = iota
	settingsFocusDetail
)

func toggleSettingsFocus(f settingsFocusPane) settingsFocusPane {
	if f == settingsFocusTree {
		return settingsFocusDetail
	}
	return settingsFocusTree
}

// SettingsModel is `otter settings`'s top-level bubbletea model: a
// two-pane terminal UI for viewing and editing otter.conf, structurally
// mirroring DocumentationModel (tree pane + detail pane, same border and
// tree styles).
type SettingsModel struct {
	tree      list.Model
	entries   []settingsEntry
	visible   []settingsEntry
	collapsed map[string]bool
	focus     settingsFocusPane

	// cfg is the working copy being edited; original is a snapshot taken
	// at load (and refreshed after every successful save), used only to
	// compute dirty() via reflect.DeepEqual.
	cfg      *config.FileConfig
	original *config.FileConfig

	isEditing bool
	editor    textinput.Model
	editError string

	confirmingQuit bool
	justSaved      bool
	saveErr        string

	width, height int
	// treeWidth is the tree pane's box width (interior plus border),
	// computed in setSize as 25% of the terminal width.
	treeWidth int
}

// NewSettingsModel loads otter.conf's current merged state and builds the
// initial settings UI model. Returns an error if the config can't be
// loaded — the same failure LoadValues would hit elsewhere in otter, not
// something specific to the settings editor.
func NewSettingsModel() (SettingsModel, error) {
	cfg, err := config.LoadFileConfig()
	if err != nil {
		return SettingsModel{}, fmt.Errorf("failed to load settings: %w", err)
	}
	original := *cfg

	entries := buildSettingsEntries()
	collapsed := map[string]bool{}
	visible := visibleSettingsEntries(entries, collapsed)

	l := list.New(settingsTreeItems(visible), settingsTreeDelegate{collapsed: collapsed, rowWidth: minTreeWidth - docPaneFrameSize}, 0, 0)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetShowHelp(false)
	l.SetShowPagination(false)
	l.SetFilteringEnabled(false)

	return SettingsModel{
		tree:      l,
		entries:   entries,
		visible:   visible,
		collapsed: collapsed,
		cfg:       cfg,
		original:  &original,
	}, nil
}

func (m SettingsModel) Init() tea.Cmd { return nil }

// dirty reports whether cfg has changed since the last load or save.
func (m SettingsModel) dirty() bool {
	return !reflect.DeepEqual(*m.cfg, *m.original)
}

func (m SettingsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m.setSize(msg.Width, msg.Height), nil

	case tea.KeyPressMsg:
		return m.handleKey(msg)
	}

	if m.isEditing {
		var cmd tea.Cmd
		m.editor, cmd = m.editor.Update(msg)
		return m, cmd
	}

	var cmd tea.Cmd
	m.tree, cmd = m.tree.Update(msg)
	return m, cmd
}

func (m SettingsModel) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	// Any keypress clears a just-shown "saved" confirmation, so it never
	// lingers past the next thing the user does. Set before the rest of
	// this call runs, so a keypress that itself triggers a new save (the
	// Save case below) can still set it again within the same call.
	m.justSaved = false

	if m.confirmingQuit {
		m.confirmingQuit = false
		if key.Matches(msg, settingsKeys.Quit) {
			return m, tea.Quit
		}
		// Any other key cancels the warning without also performing its
		// own action — a single keypress shouldn't both dismiss the
		// warning and silently do something else.
		return m, nil
	}

	if m.isEditing {
		return m.handleEditKey(msg)
	}

	switch {
	case key.Matches(msg, settingsKeys.Quit):
		if m.dirty() {
			m.confirmingQuit = true
			return m, nil
		}
		return m, tea.Quit

	case key.Matches(msg, settingsKeys.Save):
		if err := config.SaveFileConfig(m.cfg); err != nil {
			m.saveErr = err.Error()
			return m, nil
		}
		original := *m.cfg
		m.original = &original
		m.saveErr = ""
		m.justSaved = true
		return m, nil

	case key.Matches(msg, settingsKeys.SwitchPane):
		m.focus = toggleSettingsFocus(m.focus)
		return m, nil

	case key.Matches(msg, settingsKeys.Edit):
		return m.handleEnter()
	}

	var cmd tea.Cmd
	m.tree, cmd = m.tree.Update(msg)
	return m, cmd
}

// handleEnter runs when Enter is pressed outside of edit mode: it expands
// or collapses a selected section header, flips a selected toggle field
// immediately, or opens a selected text field for editing.
func (m SettingsModel) handleEnter() (tea.Model, tea.Cmd) {
	sel, ok := m.tree.SelectedItem().(settingsTreeItem)
	if !ok {
		return m, nil
	}
	e := sel.entry

	if e.isSection {
		return m.toggleCollapsed(e.section), nil
	}

	switch e.kind {
	case settingsKindToggle:
		e.setBool(m.cfg, !e.getBool(m.cfg))
		return m, nil
	case settingsKindText:
		return m.startEditing(e), nil
	}
	return m, nil
}

// toggleCollapsed flips section's collapsed state, rebuilds the tree's
// visible entries and list items to match, and keeps the cursor resting on
// section's own header row — mirroring DocumentationModel.toggleCollapsed.
func (m SettingsModel) toggleCollapsed(section string) SettingsModel {
	m.collapsed[section] = !m.collapsed[section]
	m.visible = visibleSettingsEntries(m.entries, m.collapsed)
	m.tree.SetItems(settingsTreeItems(m.visible))

	for i, e := range m.visible {
		if e.isSection && e.section == section {
			m.tree.Select(i)
			break
		}
	}
	return m
}

// startEditing opens e's text field for editing, seeding the textinput
// with its current value.
//
// NOTE: textinput.Model's Focus() method returns a tea.Cmd in bubbles v1
// (used to start cursor-blink animation); this call could not be verified
// against charm.land/bubbles/v2 specifically in this session — the
// charm.land domain isn't reachable from this sandbox's network allowlist,
// and no local module cache or Go toolchain was available to inspect or
// build against the real v2 API. If v2's signature differs, this is the
// one line to adjust.
func (m SettingsModel) startEditing(e settingsEntry) SettingsModel {
	ti := textinput.New()
	ti.SetValue(e.getText(m.cfg))
	ti.Focus()
	ti.CursorEnd()

	m.editor = ti
	m.isEditing = true
	m.editError = ""
	m.focus = settingsFocusDetail
	return m
}

func (m SettingsModel) handleEditKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, settingsKeys.Cancel):
		m.isEditing = false
		m.editError = ""
		m.focus = settingsFocusTree
		return m, nil

	case key.Matches(msg, settingsKeys.Edit):
		return m.commitEdit()
	}

	var cmd tea.Cmd
	m.editor, cmd = m.editor.Update(msg)
	return m, cmd
}

// commitEdit validates the in-progress edit against the selected field's
// rules (numeric, or a fixed option set) and, if valid, writes it back to
// cfg and closes the editor. An invalid value rejects and stays in edit
// mode — see settingsEntry.description for how the detail pane surfaces
// this (red brackets + a short "invalid value" line, no restatement of the
// valid set, since the description above it already states one).
func (m SettingsModel) commitEdit() (tea.Model, tea.Cmd) {
	sel, ok := m.tree.SelectedItem().(settingsTreeItem)
	if !ok {
		m.isEditing = false
		return m, nil
	}
	e := sel.entry
	val := m.editor.Value()

	if e.numeric {
		if _, err := strconv.Atoi(val); err != nil {
			m.editError = "invalid value"
			return m, nil
		}
	}
	if len(e.options) > 0 && !slices.Contains(e.options, val) {
		m.editError = "invalid value"
		return m, nil
	}

	e.setText(m.cfg, val)
	m.isEditing = false
	m.editError = ""
	m.focus = settingsFocusTree
	return m, nil
}

func (m SettingsModel) setSize(width, height int) SettingsModel {
	const statusAndHelpRows = 2

	m.treeWidth = int(float64(width) * treePaneFraction)
	if m.treeWidth < minTreeWidth {
		m.treeWidth = minTreeWidth
	}
	treeRowWidth := m.treeWidth - docPaneFrameSize
	if treeRowWidth < 0 {
		treeRowWidth = 0
	}
	m.tree.SetDelegate(settingsTreeDelegate{collapsed: m.collapsed, rowWidth: treeRowWidth})

	contentWidth := width - m.treeWidth
	if contentWidth < 0 {
		contentWidth = 0
	}
	paneHeight := height - statusAndHelpRows
	if paneHeight < 0 {
		paneHeight = 0
	}

	m.tree.SetSize(treeRowWidth, paneHeight-docPaneFrameSize)

	if m.isEditing {
		detailWidth := contentWidth - docPaneFrameSize
		const bracketDecorationWidth = 4 // accounts for "[ " and " ]"
		editorWidth := detailWidth - bracketDecorationWidth
		if editorWidth < 0 {
			editorWidth = 0
		}
		m.editor.SetWidth(editorWidth)
	}

	return m
}

func (m SettingsModel) View() tea.View {
	treeBorder := paneBorderStyle
	detailBorder := paneBorderStyle
	if m.focus == settingsFocusTree {
		treeBorder = focusedPaneBorderStyle
	} else {
		detailBorder = focusedPaneBorderStyle
	}

	sel, hasSel := m.tree.SelectedItem().(settingsTreeItem)

	var detailTitle, detailBody string
	switch {
	case !hasSel:
		detailTitle = "settings"
	case sel.entry.isSection:
		detailTitle = sel.entry.section
		detailBody = "(section — press enter to expand or collapse)"
	default:
		detailTitle = sel.entry.field
		detailBody = m.renderFieldDetail(sel.entry)
	}

	treePane := treeBorder.Render(m.tree.View())

	// The detail pane, like the tree, is rendered as a bordered box that
	// auto-sizes to its content — so without explicitly padding it out to
	// its allocated width the border would shrink to the description's
	// natural width and stop short of the terminal edge. Pad the interior
	// to the full allocated pane width (and let lipgloss word-wrap long
	// descriptions to it) so the box always reaches 25%/75%-of-window.
	detailWidth := m.width - m.treeWidth - docPaneFrameSize
	if detailWidth < 0 {
		detailWidth = 0
	}
	detailContent := lipgloss.NewStyle().Bold(true).Render(detailTitle) + "\n\n" +
		lipgloss.NewStyle().Width(detailWidth).Render(detailBody)
	contentPane := detailBorder.Render(detailContent)

	body := lipgloss.JoinHorizontal(lipgloss.Top, treePane, contentPane)
	status := m.renderStatusLine()
	help := renderSettingsHelp(m.dirty())

	v := tea.NewView(body + "\n" + status + "\n" + help)
	v.AltScreen = true
	return v
}

// renderFieldDetail renders the currently-selected leaf field's value
// widget plus its description into the detail pane, per settingsEntry.kind.
func (m SettingsModel) renderFieldDetail(e settingsEntry) string {
	var value string

	switch e.kind {
	case settingsKindToggle:
		value = "[ ] off"
		if e.getBool(m.cfg) {
			value = "[x] on"
		}

	case settingsKindText:
		bracketStyle := lipgloss.NewStyle()
		if m.isEditing && m.editError != "" {
			bracketStyle = bracketStyle.Foreground(colorRed)
		}
		if m.isEditing {
			value = bracketStyle.Render("[ ") + m.editor.View() + bracketStyle.Render(" ]")
		} else {
			value = "[ " + e.getText(m.cfg) + " ]"
		}
	}

	body := value + "\n\n" + e.description

	if m.isEditing && m.editError != "" {
		body += "\n" + lipgloss.NewStyle().Foreground(colorRed).Render("✗ "+m.editError)
	}

	return body
}

// renderStatusLine renders the always-visible status line: an unsaved-quit
// warning takes priority, then a save-failure message, then a transient
// "saved" confirmation, then whether there are unsaved edits, and finally
// the resting "no changes" state — so exactly one of these five is shown
// at all times, never a blank line.
func (m SettingsModel) renderStatusLine() string {
	switch {
	case m.confirmingQuit:
		return lipgloss.NewStyle().Foreground(colorRed).
			Render("unsaved changes — press q again to discard, any other key to cancel")
	case m.saveErr != "":
		return lipgloss.NewStyle().Foreground(colorRed).Render("✗ failed to save: " + m.saveErr)
	case m.justSaved:
		return lipgloss.NewStyle().Foreground(colorGreen).Render("✓ saved")
	case m.dirty():
		return lipgloss.NewStyle().Foreground(colorYellow).Render("● unsaved changes")
	default:
		return helpStyle.Render("○ no changes")
	}
}

// renderSettingsHelp renders the bottom keybinding line by hand, rather
// than via bubbles/help.Model as documentation.go does, because the save
// entry's color must change with dirty independently of the other keys —
// help.Model styles every key/desc pair uniformly and has no per-key
// override, so it can't produce that on its own.
func renderSettingsHelp(dirty bool) string {
	saveStyle := helpStyle
	if dirty {
		saveStyle = lipgloss.NewStyle().Foreground(colorYellow)
	}

	segments := []string{
		helpStyle.Render("↑/↓") + " " + helpStyle.Render("move"),
		helpStyle.Render("←/→") + " " + helpStyle.Render("switch pane"),
		helpStyle.Render("↵") + " " + helpStyle.Render("edit/toggle"),
		saveStyle.Render("s") + " " + helpStyle.Render("save"),
		helpStyle.Render("q") + " " + helpStyle.Render("quit"),
	}
	return strings.Join(segments, "  ")
}
