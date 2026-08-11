package webui

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/coder/websocket"

	"github.com/ferret-linux/otter/pkg/commands"
	"github.com/ferret-linux/otter/pkg/containermanager"
	"github.com/ferret-linux/otter/pkg/registry"
	"github.com/ferret-linux/otter/pkg/ui"
)

// rowData is the webui's per-row view model: a Container plus its lock and
// upgrade state. ListCommand doesn't include lock state (see
// commands.IsLocked's doc comment on why it's a separate, per-container
// check rather than something ListCommand computes for every container up
// front); upgrade state is the same shape of check, via
// containermanager.ContainerManager.IsUpgrading. Selected reflects whether
// this row is the currently-open detail-panel selection for this specific
// request — computed server-side per request (see containerList) so the
// row-selected highlight survives the dashboard's 3s poll, instead of
// depending only on client-side JS state that's wiped when htmx replaces
// the polled markup.
type rowData struct {
	containermanager.Container
	Locked    bool
	Upgrading bool
	Selected  bool
}

// buildRows attaches lock and upgrade state to each container for template
// rendering. This calls commands.IsLocked and s.cm.IsUpgrading once per
// container, each a real container-filesystem check — cheap for the small
// number of containers typical of a single otter host, but something to be
// aware of if that assumption stops holding.
func (s *server) buildRows(ctx context.Context, containers []containermanager.Container) []rowData {
	rows := make([]rowData, len(containers))
	for i, c := range containers {
		rows[i] = rowData{
			Container: c,
			Locked:    commands.IsLocked(ctx, s.cm, c.Name),
			Upgrading: s.cm.IsUpgrading(ctx, c.Name),
		}
	}
	return rows
}

// listRows fetches the current container list and attaches lock/upgrade
// state to each row (see buildRows), shared by index and containerList so
// the initial page load and the polled fragment stay in sync.
func (s *server) listRows(ctx context.Context) ([]rowData, error) {
	result, err := commands.NewListCommand(s.cm).Execute(ctx, commands.ListOptions{})
	if err != nil {
		return nil, err
	}
	return s.buildRows(ctx, result.Containers), nil
}

// pageData is the common view model every full page (layout.html's
// "content" block) renders with: which sidebar link is active (Nav) and a
// human title for <title> (PageTitle). Page-specific data is attached via
// embedding a concrete *Data struct below, matching the pattern
// inspectData already used for InspectResult+Upgrading before this change.
type pageData struct {
	Nav       string
	PageTitle string
}

// indexData is the Home page's view model: the container rows plus which
// container name (if any) should be pre-selected in the detail panel and
// have its row highlighted. Selection only happens via the row's onclick
// JS today (see templates/row.html), so Selected is always empty on a full
// page load; it exists so containerList (the polled fragment) can accept
// and preserve a selection across polls instead of the highlight being
// wiped every 3s when the row set is replaced.
type indexData struct {
	pageData
	Rows     []rowData
	Selected string
}

func (s *server) index(w http.ResponseWriter, r *http.Request) {
	rows, err := s.listRows(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := indexData{
		pageData: pageData{Nav: "home", PageTitle: "instances"},
		Rows:     rows,
	}
	if err := s.templates.ExecuteTemplate(w, "layout", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// containerList handles GET /containers: the same row data as index,
// rendered as just the container_list fragment (no layout/sidebar). It's
// polled by the dashboard (see templates/index.html's "container_list"
// block) so the table stays current if a container's state changes
// elsewhere — the CLI, another browser tab, or a crash — without the user
// reloading.
//
// The optional ?selected=name query parameter (sent by the poll via
// hx-vals, see templates/index.html) is echoed back into container_list so
// the currently-open detail panel's row keeps its row-selected highlight
// across polls, instead of the highlight being lost every time htmx
// replaces #container-list's rows wholesale.
func (s *server) containerList(w http.ResponseWriter, r *http.Request) {
	rows, err := s.listRows(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	selected := r.URL.Query().Get("selected")
	for i := range rows {
		rows[i].Selected = selected != "" && rows[i].Name == selected
	}

	data := struct {
		Rows     []rowData
		Selected string
	}{Rows: rows, Selected: selected}

	if err := s.templates.ExecuteTemplate(w, "container_list", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// containerPanel handles GET /containers/{name}/panel: the Home page's
// right-hand detail panel fragment for a single container, replacing the
// old standalone /containers/{name}/inspect page. htmx swaps this into
// #detail-panel (see templates/row.html's hx-get) without a full page
// navigation, matching how selecting an instance behaves in Incus's web UI
// (see the reference screenshots this design followed).
func (s *server) containerPanel(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	ctx := r.Context()

	result, err := commands.NewInspectCommand(s.cm).Execute(ctx, commands.InspectOptions{
		ContainerName: name,
		Manager:       s.cm.Name(),
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := inspectData{InspectResult: result, Upgrading: s.cm.IsUpgrading(ctx, name)}
	if err := s.templates.ExecuteTemplate(w, "container_panel", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// action handles POST /containers/{name}/{action} for action in
// start, stop, pause, remove. It re-renders the container's row on success
// (htmx swaps it in), or an empty body for a successful remove.
func (s *server) action(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	action := r.PathValue("action")
	ctx := r.Context()

	var err error
	switch action {
	case "start":
		err = commands.NewStartCommand(s.cm).Execute(ctx, &commands.StartOptions{ContainerNames: []string{name}})
	case "stop":
		err = commands.NewStopCommand(s.cm).Execute(ctx, &commands.StopOptions{ContainerNames: []string{name}})
	case "pause":
		err = commands.NewPauseCommand(s.cm).Execute(ctx, &commands.PauseOptions{ContainerNames: []string{name}})
	case "restart":
		err = commands.NewRestartCommand(s.cm).Execute(ctx, &commands.RestartOptions{ContainerNames: []string{name}})
	case "lock":
		err = commands.NewLockCommand(s.cm).Execute(ctx, commands.LockOptions{ContainerNames: []string{name}})
	case "unlock":
		err = commands.NewUnlockCommand(s.cm).Execute(ctx, commands.UnlockOptions{ContainerNames: []string{name}})
	case "remove":
		_, err = commands.NewRmCommand(s.cm).Execute(ctx, commands.RmOptions{ContainerNames: []string{name}})
	default:
		http.Error(w, "unknown action: "+action, http.StatusBadRequest)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if action == "remove" {
		// Row removed: respond with nothing so htmx's outerHTML swap clears it.
		w.WriteHeader(http.StatusOK)
		return
	}

	result, err := commands.NewListCommand(s.cm).Execute(ctx, commands.ListOptions{})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	for _, c := range result.Containers {
		if c.Name == name {
			row := rowData{
				Container: c,
				Locked:    commands.IsLocked(ctx, s.cm, c.Name),
				Upgrading: s.cm.IsUpgrading(ctx, c.Name),
			}
			if err := s.templates.ExecuteTemplate(w, "row", row); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
			return
		}
	}
	w.WriteHeader(http.StatusOK)
}

// consolePageData is the Console page's view model: every currently
// running container (attachable as a shell) and every currently upgrading
// container (attachable as a live upgrade-output view), matching the
// "Running containers" / "Upgrading containers" picker lists design — see
// templates/console.html.
type consolePageData struct {
	pageData
	Running   []rowData
	Upgrading []rowData
}

// consolePage handles GET /console: the Console picker page, listing
// running and upgrading containers to attach a shell or watch an upgrade,
// replacing the old per-row "Terminal" link and standalone
// /containers/{name}/terminal and /containers/{name}/upgrade pages.
func (s *server) consolePage(w http.ResponseWriter, r *http.Request) {
	rows, err := s.listRows(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := consolePageData{pageData: pageData{Nav: "console", PageTitle: "console"}}
	for _, row := range rows {
		if row.Upgrading {
			data.Upgrading = append(data.Upgrading, row)
			continue
		}
		if row.IsRunning() {
			data.Running = append(data.Running, row)
		}
	}

	if err := s.templates.ExecuteTemplate(w, "layout", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// consoleShellFragment handles GET /console/{name}/shell: the terminal
// detail-pane fragment for a running container, htmx-swapped into
// #picker-detail when its list item is clicked (see templates/console.html).
// It renders the same xterm/websocket bootstrap terminal.html used to, just
// as a fragment instead of a full page.
func (s *server) consoleShellFragment(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if err := s.templates.ExecuteTemplate(w, "console_shell", name); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// consoleWatchFragment handles GET /console/{name}/watch: the live
// upgrade-output detail-pane fragment for an upgrading container,
// htmx-swapped into #picker-detail the same way consoleShellFragment is.
// It renders the same SSE bootstrap upgrade.html used to, just as a
// fragment instead of a full page.
func (s *server) consoleWatchFragment(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if err := s.templates.ExecuteTemplate(w, "console_watch", name); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// logsListPage handles GET /logs: the Logs picker page, listing every
// container to view its runtime log tail, replacing the old per-row "Logs"
// link and standalone /containers/{name}/logs page.
func (s *server) logsListPage(w http.ResponseWriter, r *http.Request) {
	rows, err := s.listRows(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := struct {
		pageData
		Rows []rowData
	}{pageData: pageData{Nav: "logs", PageTitle: "logs"}, Rows: rows}

	if err := s.templates.ExecuteTemplate(w, "layout", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// logsViewFragment handles GET /logs/{name}/view: the log-tail detail-pane
// fragment for a container, htmx-swapped into #picker-detail when its list
// item is clicked (see templates/logs.html). It renders the same SSE
// bootstrap logs.html used to, just as a fragment instead of a full page.
func (s *server) logsViewFragment(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if err := s.templates.ExecuteTemplate(w, "logs_view", name); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// logsSSE streams a container's log output to the browser as
// Server-Sent Events, via the same commands.NewJournalCommand() the CLI's
// `otter journal` uses — just with Stdout/Stderr redirected to the SSE
// response instead of the otter process's own stdout/stderr. It always
// follows, capped at sseLogTailLines of initial history, so opening the
// page doesn't replay a container's entire log history. The connection
// ends when the client disconnects (r.Context() is canceled, which
// propagates down to the exec.CommandContext driving `docker/podman/nerdctl
// logs -f`) or when Journal returns.
func (s *server) logsSSE(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	sw := &sseLogWriter{w: w, flusher: flusher}

	if err := commands.NewJournalCommand(s.cm).Execute(r.Context(), commands.JournalOptions{
		ContainerName: name,
		Follow:        true,
		Tail:          sseLogTailLines,
		Stdout:        sw,
		Stderr:        sw,
	}); err != nil {
		fmt.Fprintf(w, "event: error\ndata: %s\n\n", err.Error())
		flusher.Flush()
	}
}

// inspectData is the webui's inspect-panel view model: an InspectResult
// plus upgrade state. InspectResult doesn't carry this itself since it's
// shared with the CLI's `otter inspect` (table and --json output), which
// has no notion of live upgrade status. Used by containerPanel.
type inspectData struct {
	*commands.InspectResult
	Upgrading bool
}

// upgradeAction handles POST /containers/{name}/upgrade. It starts
// commands.NewUpgradeCommand in a background goroutine detached from this
// request's context, so the upgrade is not interrupted by the triggering
// request finishing, the browser tab closing, or a live-view SSE connection
// dropping — matching how the underlying otter-init --upgrade run is not
// tied to any particular otter-side process either. Actual upgrade-in-
// progress status is read back from s.cm.IsUpgrading (the
// container.upgrading marker file), not tracked separately here, so it
// stays correct even if the webui process restarts mid-upgrade.
//
// If an upgrade is already in progress for this container, this is a
// no-op: it does not start a second concurrent run.
func (s *server) upgradeAction(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	ctx := r.Context()

	if !s.cm.IsUpgrading(ctx, name) {
		stream := newUpgradeStream()
		s.upgradeStreamsMu.Lock()
		s.upgradeStreams[name] = stream
		s.upgradeStreamsMu.Unlock()

		go func() {
			defer func() {
				stream.close()
				s.upgradeStreamsMu.Lock()
				delete(s.upgradeStreams, name)
				s.upgradeStreamsMu.Unlock()
			}()

			sw := &upgradeStreamWriter{stream: stream}
			if err := commands.NewUpgradeCommand(s.cfg, s.cm).Execute(context.Background(), &commands.UpgradeOptions{
				ContainerNames: []string{name},
				Stdout:         sw,
				Stderr:         sw,
			}); err != nil {
				stream.write(fmt.Sprintf("error: %s", err.Error()))
			}
		}()
	}

	// Upgrade is now only ever triggered from the Home detail panel, which
	// is itself an htmx fragment (see templates/container_panel.html), so
	// every caller sets HX-Request. Non-htmx callers (e.g. curl) get sent
	// to Console's watch view for this container instead of a 404, since
	// the old standalone /containers/{name}/upgrade status page no longer
	// exists — live output now only lives at /console/{name}/watch.
	if r.Header.Get("HX-Request") == "" {
		http.Redirect(w, r, "/console/"+name+"/watch", http.StatusSeeOther)
		return
	}

	// Re-render the detail panel (not the row — the panel is what shows
	// the Upgrade button and now-upgrading state) so htmx can swap
	// #detail-panel in place, matching containerPanel's own rendering.
	panelResult, err := commands.NewInspectCommand(s.cm).Execute(ctx, commands.InspectOptions{
		ContainerName: name,
		Manager:       s.cm.Name(),
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	panelData := inspectData{InspectResult: panelResult, Upgrading: s.cm.IsUpgrading(ctx, name)}
	if err := s.templates.ExecuteTemplate(w, "container_panel", panelData); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// upgradeSSE streams a running upgrade's output to the browser as
// Server-Sent Events, if one is currently in progress for this container.
// Unlike logsSSE, the upgrade itself is not tied to this connection (see
// upgradeAction) — this handler only attaches to and detaches from it;
// closing the SSE connection stops the browser from watching, not the
// upgrade itself. If no upgrade is in progress, it sends a single message
// saying so and closes.
func (s *server) upgradeSSE(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	s.upgradeStreamsMu.Lock()
	stream, ok := s.upgradeStreams[name]
	s.upgradeStreamsMu.Unlock()
	if !ok {
		fmt.Fprintf(w, "event: done\ndata: no upgrade currently in progress\n\n")
		flusher.Flush()
		return
	}

	buf, ch := stream.subscribe()
	defer stream.unsubscribe(ch)

	for _, line := range buf {
		fmt.Fprintf(w, "data: %s\n\n", line)
	}
	flusher.Flush()

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case line, open := <-ch:
			if !open {
				fmt.Fprintf(w, "event: done\ndata: upgrade finished\n\n")
				flusher.Flush()
				return
			}
			fmt.Fprintf(w, "data: %s\n\n", line)
			flusher.Flush()
		case <-time.After(30 * time.Second):
			// Keepalive comment so intermediary proxies/browsers don't
			// time out an idle connection while the upgrade is still
			// running but quiet.
			fmt.Fprint(w, ": keepalive\n\n")
			flusher.Flush()
		}
	}
}

// registryRowData is the webui's per-row view model for the registry
// panel: a commands.RegistryRow plus its rendered local-status text/class
// and whether a pull is currently running for it (see server.pulling).
type registryRowData struct {
	commands.RegistryRow
	Pulling      bool
	LocalDisplay string
	LocalClass   string
}

// registryRows fetches the current registry catalog and returns it as
// per-row view models, attaching each row's in-memory pull state.
func (s *server) registryRows(ctx context.Context, all bool) ([]registryRowData, error) {
	props, err := registry.Fetch(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch registry: %w", err)
	}
	rows := commands.BuildRegistryRows(ctx, s.cm, props, all)

	s.pullingMu.Lock()
	defer s.pullingMu.Unlock()

	out := make([]registryRowData, len(rows))
	for i, row := range rows {
		out[i] = registryRowData{
			RegistryRow:  row,
			Pulling:      s.pulling[row.Name],
			LocalDisplay: commands.RegistryLocalDisplay(row.Local, row.LocalDiff),
			LocalClass:   registryLocalClass(row.Local),
		}
	}
	return out, nil
}

// registryLocalClass maps a commands.RegistryLocalState to one of the
// existing container-status CSS classes (status-running/status-upgrading),
// reusing the same green/amber color scheme already defined for container
// state instead of introducing new styling for image state.
func registryLocalClass(state commands.RegistryLocalState) string {
	switch state {
	case commands.RegistryLocalCurrent:
		return "status-running"
	case commands.RegistryLocalBehind, commands.RegistryLocalAhead:
		return "status-upgrading"
	default:
		return "status-stopped"
	}
}

// registryPage renders the registry panel: every enabled registry entry
// with its local pull state, matching `otter registry list`'s default
// (no --all) view. Wrapped in the shared sidebar layout like every other
// top-level nav destination.
func (s *server) registryPage(w http.ResponseWriter, r *http.Request) {
	rows, err := s.registryRows(r.Context(), false)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := struct {
		pageData
		Rows []registryRowData
	}{pageData: pageData{Nav: "registry", PageTitle: "registry"}, Rows: rows}

	if err := s.templates.ExecuteTemplate(w, "layout", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// registryPullAction handles POST /registry/{name}/pull. It starts
// commands.NewRegistryPullCommand in a background goroutine detached from
// this request's context, for the same reason upgradeAction does: a pull
// can take minutes, far longer than it's reasonable to hold this request
// open for. Unlike container upgrades, there's no on-disk marker for an
// in-progress image pull, so s.pulling is the only record of pull-in-
// progress state and does not survive a webui restart — the pull itself,
// driven by the container engine, is unaffected either way.
//
// If a pull is already in progress for this name, this is a no-op: it does
// not start a second concurrent pull. The row is re-rendered immediately
// showing the "pulling" state; picking up completion requires reloading
// the registry page, since nothing pushes an update once the pull finishes.
func (s *server) registryPullAction(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")

	s.pullingMu.Lock()
	alreadyPulling := s.pulling[name]
	if !alreadyPulling {
		s.pulling[name] = true
	}
	s.pullingMu.Unlock()

	if !alreadyPulling {
		go func() {
			defer func() {
				s.pullingMu.Lock()
				delete(s.pulling, name)
				s.pullingMu.Unlock()
			}()
			if err := commands.NewRegistryPullCommand(s.cm).Execute(context.Background(), commands.RegistryPullOptions{
				Names: []string{name},
			}); err != nil {
				ui.DefaultLogger.Error(err)
			}
		}()
	}

	s.renderRegistryRow(w, r.Context(), name)
}

// registryRemoveAction handles POST /registry/{name}/remove. Removal is
// fast, so unlike pull it runs synchronously within the request.
func (s *server) registryRemoveAction(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	ctx := r.Context()

	if err := commands.NewRegistryRemoveCommand(s.cm).Execute(ctx, commands.RegistryRemoveOptions{
		Names: []string{name},
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.renderRegistryRow(w, ctx, name)
}

// renderRegistryRow re-fetches the registry and re-renders a single row for
// htmx to swap in, matching the row-refresh pattern action and
// upgradeAction use for the container table.
func (s *server) renderRegistryRow(w http.ResponseWriter, ctx context.Context, name string) {
	rows, err := s.registryRows(ctx, false)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	for _, row := range rows {
		if row.Name == name {
			if err := s.templates.ExecuteTemplate(w, "registry_row", row); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
			return
		}
	}
	w.WriteHeader(http.StatusOK)
}

// createFormData holds the create form's field values, both for the
// initial empty form and for re-rendering it with the submitted values and
// an inline error if creation fails, so nothing typed is lost.
type createFormData struct {
	Name               string
	Image              string
	Hostname           string
	Shell              string
	Clone              string
	Home               string
	Volumes            string
	AdditionalFlags    string
	AdditionalPackages string
	InitHooks          string
	PreInitHooks       string
	Memory             string
	CPUThreads         string
	Platform           string
	AlwaysPull         bool
	Init               bool
	Nvidia             bool
	NoUsernsLimit      bool
	UnshareDevsys      bool
	UnshareGroups      bool
	UnshareIPC         bool
	UnshareNetNS       bool
	UnshareProcess     bool
	GenerateEntry      bool
	Nopasswd           bool
	DefaultImage       string
	Error              string
}

// createPage handles GET /create: the "+ Create new" sidebar destination,
// wrapped in the shared sidebar layout like every other top-level nav
// destination (unlike its predecessor, /containers/new, which was a
// standalone page).
func (s *server) createPage(w http.ResponseWriter, r *http.Request) {
	data := struct {
		pageData
		createFormData
	}{
		pageData: pageData{Nav: "create", PageTitle: "create new"},
		createFormData: createFormData{
			GenerateEntry: true,
			DefaultImage:  s.cfg.DefaultContainerImage,
			Image:         r.URL.Query().Get("image"),
		},
	}
	if err := s.templates.ExecuteTemplate(w, "layout", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// createContainer handles POST /containers via commands.NewCreateCommand —
// the same command the CLI's `otter create` uses. It always creates in
// whatever mode this webui process itself is running in (s.rootful), since
// container privilege comes from which containermanager built it, not a
// per-request choice; there's no root toggle in the form. On success it
// redirects to /; on failure it re-renders the form with the submitted
// values and an inline error.
func (s *server) createContainer(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	data := createFormData{
		Name:               r.FormValue("name"),
		Image:              r.FormValue("image"),
		Hostname:           r.FormValue("hostname"),
		Shell:              r.FormValue("shell"),
		Clone:              r.FormValue("clone"),
		Home:               r.FormValue("home"),
		Volumes:            r.FormValue("volumes"),
		AdditionalFlags:    r.FormValue("additional-flags"),
		AdditionalPackages: r.FormValue("additional-packages"),
		InitHooks:          r.FormValue("init-hooks"),
		PreInitHooks:       r.FormValue("pre-init-hooks"),
		Memory:             r.FormValue("memory"),
		CPUThreads:         r.FormValue("cpu-threads"),
		Platform:           r.FormValue("platform"),
		AlwaysPull:         r.FormValue("always-pull") != "",
		Init:               r.FormValue("init") != "",
		Nvidia:             r.FormValue("nvidia") != "",
		NoUsernsLimit:      r.FormValue("no-userns-limit") != "",
		UnshareDevsys:      r.FormValue("unshare-devsys") != "",
		UnshareGroups:      r.FormValue("unshare-groups") != "",
		UnshareIPC:         r.FormValue("unshare-ipc") != "",
		UnshareNetNS:       r.FormValue("unshare-netns") != "",
		UnshareProcess:     r.FormValue("unshare-process") != "",
		GenerateEntry:      r.FormValue("generate-entry") != "",
		Nopasswd:           r.FormValue("nopasswd") != "",
		DefaultImage:       s.cfg.DefaultContainerImage,
	}

	cpuThreads := 0
	if data.CPUThreads != "" {
		var err error
		cpuThreads, err = strconv.Atoi(data.CPUThreads)
		if err != nil {
			data.Error = "cpu threads must be a whole number"
			s.renderCreateForm(w, data)
			return
		}
	}

	opts := commands.CreateOptions{
		ContainerImage:          data.Image,
		ContainerName:           data.Name,
		ContainerHostname:       data.Hostname,
		ContainerClone:          data.Clone,
		ContainerShell:          data.Shell,
		ContainerUserCustomHome: data.Home,
		ContainerPlatform:       data.Platform,
		Nopasswd:                data.Nopasswd,
		UnshareDevsys:           data.UnshareDevsys,
		// Init implies unsharing process and group namespaces, matching
		// the CLI's own --init handling (internal/cli/create.go).
		UnshareGroups:        data.UnshareGroups || data.Init,
		UnshareIpc:           data.UnshareIPC,
		UnshareNetNs:         data.UnshareNetNS,
		UnshareProcess:       data.UnshareProcess || data.Init,
		AdditionalFlags:      splitLines(data.AdditionalFlags),
		AdditionalVolumes:    splitLines(data.Volumes),
		AdditionalPackages:   splitLines(data.AdditionalPackages),
		ContainerPreInitHook: data.PreInitHooks,
		ContainerInitHook:    data.InitHooks,
		Init:                 data.Init,
		Nvidia:               data.Nvidia,
		NoUsernsLimit:        data.NoUsernsLimit,
		Memory:               data.Memory,
		CPUThreads:           cpuThreads,
		GenerateEntry:        data.GenerateEntry,
		Rootful:              s.rootful,
		ContainerAlwaysPull:  data.AlwaysPull,
	}

	if _, err := commands.NewCreateCommand(s.cfg, s.cm, nil).Execute(r.Context(), opts); err != nil {
		data.Error = err.Error()
		s.renderCreateForm(w, data)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (s *server) renderCreateForm(w http.ResponseWriter, data createFormData) {
	w.WriteHeader(http.StatusUnprocessableEntity)
	full := struct {
		pageData
		createFormData
	}{
		pageData:       pageData{Nav: "create", PageTitle: "create new"},
		createFormData: data,
	}
	if err := s.templates.ExecuteTemplate(w, "layout", full); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// splitLines splits newline-separated textarea input into a trimmed,
// non-empty slice, matching the CLI's repeatable-flag semantics for the
// same fields (--volume, --additional-flags, --additional-packages).
func splitLines(in string) []string {
	var out []string
	for _, line := range strings.Split(in, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			out = append(out, line)
		}
	}
	return out
}

// terminalWS bridges a browser websocket to an interactive shell inside the
// container, via the same containermanager.Enter() the CLI's `otter enter`
// uses — just with Stdin/Stdout/Stderr redirected to the websocket instead
// of the otter process's own terminal. See containermanager.EnterOptions and
// pkg/commands.EnterOptions for the redirection fields.
//
// Terminal input and resize control messages share this one connection:
// binary messages are input bytes, text messages are JSON resize events —
// see wsTerminalReader. Output only ever flows as binary.
func (s *server) terminalWS(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")

	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{})
	if err != nil {
		return
	}
	defer conn.CloseNow()

	ctx := r.Context()
	resizeCh := make(chan containermanager.WinSize, 1)
	reader := newWSTerminalReader(ctx, conn, resizeCh)
	writer := websocket.NetConn(ctx, conn, websocket.MessageBinary)
	defer writer.Close()

	_, err = commands.NewEnterCommand(s.cm).Execute(ctx, commands.EnterOptions{
		ContainerName: name,
		ForceTTY:      true,
		Stdin:         reader,
		Stdout:        writer,
		Stderr:        writer,
		Resize:        resizeCh,
	})

	reason := "session ended"
	if err != nil {
		reason = "session error"
	}
	conn.Close(websocket.StatusNormalClosure, reason)
}
