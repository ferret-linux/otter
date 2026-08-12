package webui

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/coder/websocket"

	"github.com/ferret-linux/otter/pkg/commands"
	"github.com/ferret-linux/otter/pkg/config"
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
		s.notify("error", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := indexData{
		pageData: pageData{Nav: "home", PageTitle: "instances"},
		Rows:     rows,
	}
	if err := s.templates.ExecuteTemplate(w, "layout", data); err != nil {
		s.notify("error", err.Error())
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
		s.notify("error", err.Error())
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
		s.notify("error", err.Error())
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
		s.notify("error", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := inspectData{InspectResult: result, Upgrading: s.cm.IsUpgrading(ctx, name)}
	if err := s.templates.ExecuteTemplate(w, "container_panel", data); err != nil {
		s.notify("error", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// action handles POST /containers/{name}/{action} for action in start,
// stop, pause, restart, lock, unlock, remove.
//
// Two callers use this route with different expectations of what comes
// back: Home's row is no longer interactive (see templates/row.html — rows
// only carry the panel-opening hx-get now), so in practice every caller is
// the detail panel's action buttons (see templates/inspect.html's
// "container_panel" block), which target #detail-panel and expect a
// re-rendered panel back. The optional ?view=row query parameter exists
// for any future caller that instead wants the old row-fragment response,
// so this handler isn't hard-wired to only ever serve the panel.
//
// On a successful remove, the container no longer exists to inspect, so
// this responds with an out-of-band swap that both clears the row (if
// still present, e.g. a stale row from a caller that isn't the panel) and
// resets #detail-panel to its empty state, rather than trying to re-render
// inspect data for a container that's gone.
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
		s.notify("error", "unknown action: "+action)
		http.Error(w, "unknown action: "+action, http.StatusBadRequest)
		return
	}
	if err != nil {
		s.notify("error", fmt.Sprintf("%s %s failed: %s", action, name, err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.notify("success", fmt.Sprintf("%s: %s succeeded", name, action))

	if action == "remove" {
		// hx-swap-oob targets #row-{name} directly (deleting it if present)
		// and #detail-panel back to its empty state, so both the list and
		// the panel reflect the removal in one response regardless of
		// which element htmx's own hx-target on the triggering button
		// pointed at.
		fmt.Fprintf(w,
			`<div id="row-%s" hx-swap-oob="delete"></div>`+
				`<div id="detail-panel" class="detail-panel" hx-swap-oob="true"><div class="detail-panel-empty">Select an instance to view details</div></div>`,
			name,
		)
		return
	}

	if r.URL.Query().Get("view") == "row" {
		result, err := commands.NewListCommand(s.cm).Execute(ctx, commands.ListOptions{})
		if err != nil {
			s.notify("error", err.Error())
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
					s.notify("error", err.Error())
					http.Error(w, err.Error(), http.StatusInternalServerError)
				}
				return
			}
		}
		w.WriteHeader(http.StatusOK)
		return
	}

	panelResult, err := commands.NewInspectCommand(s.cm).Execute(ctx, commands.InspectOptions{
		ContainerName: name,
		Manager:       s.cm.Name(),
	})
	if err != nil {
		s.notify("error", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	panelData := inspectData{InspectResult: panelResult, Upgrading: s.cm.IsUpgrading(ctx, name)}
	if err := s.templates.ExecuteTemplate(w, "container_panel", panelData); err != nil {
		s.notify("error", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
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
		s.notify("error", err.Error())
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
		s.notify("error", err.Error())
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
		s.notify("error", err.Error())
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
		s.notify("error", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// logsListPage handles GET /logs: the Logs picker page, listing every
// container to view its runtime log tail, replacing the old per-row "Logs"
// link and standalone /containers/{name}/logs page.
func (s *server) logsListPage(w http.ResponseWriter, r *http.Request) {
	rows, err := s.listRows(r.Context())
	if err != nil {
		s.notify("error", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := struct {
		pageData
		Rows []rowData
	}{pageData: pageData{Nav: "logs", PageTitle: "logs"}, Rows: rows}

	if err := s.templates.ExecuteTemplate(w, "layout", data); err != nil {
		s.notify("error", err.Error())
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
		s.notify("error", err.Error())
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
		s.notify("error", "streaming unsupported")
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
		s.notify("error", fmt.Sprintf("logs %s: %s", name, err))
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
// request finishing, the browser tab closing, or a live-view websocket
// connection dropping — matching how the underlying otter-init --upgrade
// run is not tied to any particular otter-side process either. Actual
// upgrade-in-progress status is read back from s.cm.IsUpgrading (the
// container.upgrading marker file), not tracked separately here, so it
// stays correct even if the webui process restarts mid-upgrade.
//
// The upgrade's stdin is backed by a *session (see session.go) from the
// moment it starts, before any browser has attached — so a package-manager
// prompt that shows up before anyone opens the watch view simply waits on
// session's stdin pipe, exactly as it would wait on a real terminal, until
// a viewer attaches, is granted control, and answers it.
//
// If an upgrade is already in progress for this container, this is a
// no-op: it does not start a second concurrent run.
func (s *server) upgradeAction(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	ctx := r.Context()

	if !s.cm.IsUpgrading(ctx, name) {
		sessionKey := "upgrade:" + name
		sess, sessIO := newSession(false)
		s.sessionsMu.Lock()
		s.sessions[sessionKey] = sess
		s.sessionsMu.Unlock()

		s.notify("info", fmt.Sprintf("upgrade started: %s", name))

		go func() {
			defer func() {
				sess.close()
				s.sessionsMu.Lock()
				delete(s.sessions, sessionKey)
				s.sessionsMu.Unlock()
			}()

			if err := commands.NewUpgradeCommand(s.config(), s.cm).Execute(context.Background(), &commands.UpgradeOptions{
				ContainerNames: []string{name},
				Stdin:          sessIO.Stdin,
				Stdout:         sess,
				Stderr:         sess,
			}); err != nil {
				_, _ = sess.Write([]byte(fmt.Sprintf("error: %s", err.Error())))
				s.notify("error", fmt.Sprintf("upgrade failed: %s: %s", name, err))
				return
			}
			s.notify("success", fmt.Sprintf("upgrade finished: %s", name))
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
		s.notify("error", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	panelData := inspectData{InspectResult: panelResult, Upgrading: s.cm.IsUpgrading(ctx, name)}
	if err := s.templates.ExecuteTemplate(w, "container_panel", panelData); err != nil {
		s.notify("error", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// upgradeWS bridges a browser websocket to a running upgrade's session, if
// one is currently in progress for this container: output flows to the
// browser for every attached connection, and input flows to the upgrade's
// stdin only from whichever connection currently holds control (see
// session.go). Unlike terminalWS, the upgrade itself is not tied to this
// connection (see upgradeAction) — this handler only attaches to and
// detaches from an already-running session; closing this connection stops
// the browser from watching/controlling, not the upgrade itself. If no
// upgrade is in progress, the connection is closed immediately.
func (s *server) upgradeWS(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")

	s.sessionsMu.Lock()
	sess, ok := s.sessions["upgrade:"+name]
	s.sessionsMu.Unlock()

	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{})
	if err != nil {
		return
	}
	defer conn.CloseNow()

	if !ok {
		conn.Close(websocket.StatusNormalClosure, "no upgrade currently in progress")
		return
	}

	s.runAttachedSession(r.Context(), conn, sess)
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
		s.notify("error", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := struct {
		pageData
		Rows []registryRowData
	}{pageData: pageData{Nav: "registry", PageTitle: "registry"}, Rows: rows}

	if err := s.templates.ExecuteTemplate(w, "layout", data); err != nil {
		s.notify("error", err.Error())
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
		s.notify("info", fmt.Sprintf("pulling %s", name))
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
				s.notify("error", fmt.Sprintf("failed to pull %s: %s", name, err))
				return
			}
			s.notify("success", fmt.Sprintf("pulled %s", name))
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
		s.notify("error", fmt.Sprintf("failed to remove %s: %s", name, err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.notify("success", fmt.Sprintf("removed %s", name))

	s.renderRegistryRow(w, ctx, name)
}

// renderRegistryRow re-fetches the registry and re-renders a single row for
// htmx to swap in, matching the row-refresh pattern action and
// upgradeAction use for the container table.
func (s *server) renderRegistryRow(w http.ResponseWriter, ctx context.Context, name string) {
	rows, err := s.registryRows(ctx, false)
	if err != nil {
		s.notify("error", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	for _, row := range rows {
		if row.Name == name {
			if err := s.templates.ExecuteTemplate(w, "registry_row", row); err != nil {
				s.notify("error", err.Error())
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
			DefaultImage:  s.config().DefaultContainerImage,
			Image:         r.URL.Query().Get("image"),
		},
	}
	if err := s.templates.ExecuteTemplate(w, "layout", data); err != nil {
		s.notify("error", err.Error())
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
		s.notify("error", err.Error())
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
		DefaultImage:       s.config().DefaultContainerImage,
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

	if _, err := commands.NewCreateCommand(s.config(), s.cm, nil).Execute(r.Context(), opts); err != nil {
		data.Error = err.Error()
		s.renderCreateForm(w, data)
		return
	}

	s.notify("success", fmt.Sprintf("created %s", data.Name))
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (s *server) renderCreateForm(w http.ResponseWriter, data createFormData) {
	if data.Error != "" {
		s.notify("error", data.Error)
	}
	w.WriteHeader(http.StatusUnprocessableEntity)
	full := struct {
		pageData
		createFormData
	}{
		pageData:       pageData{Nav: "create", PageTitle: "create new"},
		createFormData: data,
	}
	if err := s.templates.ExecuteTemplate(w, "layout", full); err != nil {
		s.notify("error", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// settingsFieldView is one config field as rendered on the Settings page:
// config.FieldState plus the human label from config.Fields and a
// lowercase CSS-safe badge name derived from its Source, so the template
// doesn't need Go-side Source constants to pick a badge class.
type settingsFieldView struct {
	config.FieldState
	Label      string
	SourceName string // "default", "system", "user", or "override"
}

// sourceName maps a config.Source to the lowercase name settingsFieldView
// and the settings template use for CSS classes and display text.
func sourceName(src config.Source) string {
	switch src {
	case config.SourceSystem:
		return "system"
	case config.SourceUser:
		return "user"
	case config.SourceOverride:
		return "override"
	default:
		return "default"
	}
}

// settingsFieldViews zips config.LoadProvenance's per-field states with
// their labels from config.Fields, relying on LoadProvenance iterating
// Fields in the same order it's declared (see provenance.go) to zip by
// index rather than needing a lookup by FieldPath.
func settingsFieldViews() ([]settingsFieldView, error) {
	states, err := config.LoadProvenance()
	if err != nil {
		return nil, err
	}
	views := make([]settingsFieldView, len(states))
	for i, st := range states {
		label := ""
		if i < len(config.Fields) {
			label = config.Fields[i].Label
		}
		views[i] = settingsFieldView{FieldState: st, Label: label, SourceName: sourceName(st.Source)}
	}
	return views, nil
}

// settingsPage handles GET /settings: shows every config field's current
// effective value and where it comes from (system config, user config, an
// override of a system default, or otter's built-in default — see
// config.LoadProvenance), each individually editable and saved via
// settingsSave.
func (s *server) settingsPage(w http.ResponseWriter, r *http.Request) {
	views, err := settingsFieldViews()
	if err != nil {
		s.notify("error", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := struct {
		pageData
		Fields []settingsFieldView
	}{pageData: pageData{Nav: "settings", PageTitle: "settings"}, Fields: views}

	if err := s.templates.ExecuteTemplate(w, "layout", data); err != nil {
		s.notify("error", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// settingsSave handles POST /settings/save: writes one field's new value
// into the user's own otter.conf (see config.SaveUserValue — system config
// files are never written to), reloads s.cfg from disk so the change is
// visible elsewhere in this running webui process without a restart (see
// server.setConfig), and redirects back to the Settings page.
//
// Saving webui.token is a special case: since s.token gates every request
// via withAuth, and the browser's session cookie is only valid for the
// token that was active when it was set, changing the token immediately
// invalidates the current session. To avoid the user locking themselves
// out, the redirect after a token save carries the new value as
// ?token=..., which withAuth's existing query-param flow (see server.go)
// picks up and turns into a fresh valid session cookie in the same
// response round-trip.
func (s *server) settingsSave(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		s.notify("error", err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	path := config.FieldPath{Section: r.FormValue("section"), Key: r.FormValue("key")}
	raw := r.FormValue("value")

	formatted, err := config.FormatTOMLValue(path, raw)
	if err != nil {
		s.notify("error", err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := config.SaveUserValue(path, formatted); err != nil {
		s.notify("error", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	newCfg, err := config.LoadValues()
	if err != nil {
		s.notify("error", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.setConfig(newCfg)
	s.notify("success", fmt.Sprintf("saved %s.%s", path.Section, path.Key))

	if path.Section == "webui" && path.Key == "token" {
		http.Redirect(w, r, "/settings?token="+url.QueryEscape(raw), http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, "/settings", http.StatusSeeOther)
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
// uses — just with Stdin/Stdout/Stderr redirected through a *session (see
// session.go) instead of the otter process's own terminal. The first
// connection to arrive for a given container name creates the session and
// starts Enter(); later connections for the same container name (e.g. a
// second browser tab) attach to that already-running session as viewers
// instead of starting a second, independent shell — matching how
// upgradeWS's session is shared across every attached viewer.
//
// Terminal input and control messages (resize, control requests) share
// each connection with its output — binary messages are input bytes, text
// messages are JSON control messages — see clientMessage and
// runSessionReadLoop.
func (s *server) terminalWS(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")

	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{})
	if err != nil {
		return
	}
	defer conn.CloseNow()

	ctx := r.Context()

	sessionKey := "shell:" + name
	s.sessionsMu.Lock()
	sess, existed := s.sessions[sessionKey]
	var sessIO sessionIO
	if !existed {
		sess, sessIO = newSession(true)
		s.sessions[sessionKey] = sess
	}
	s.sessionsMu.Unlock()

	if !existed {
		go func() {
			defer func() {
				sess.close()
				s.sessionsMu.Lock()
				delete(s.sessions, sessionKey)
				s.sessionsMu.Unlock()
			}()

			_, _ = commands.NewEnterCommand(s.cm).Execute(context.Background(), commands.EnterOptions{
				ContainerName: name,
				ForceTTY:      true,
				Stdin:         sessIO.Stdin,
				Stdout:        sess,
				Stderr:        sess,
				Resize:        sess.resizeCh,
			})
		}()
	}

	s.runAttachedSession(ctx, conn, sess)
}

// runAttachedSession attaches conn to sess as a new connection (controller
// if none is attached yet, viewer otherwise), replays buffered output,
// then runs the read and write loops until the connection or session ends.
// Shared by terminalWS and upgradeWS since both are session-backed the
// same way.
func (s *server) runAttachedSession(ctx context.Context, conn *websocket.Conn, sess *session) {
	c, ok := sess.attach(conn)
	if !ok {
		conn.Close(websocket.StatusNormalClosure, "session ended")
		return
	}
	defer sess.detach(c)

	go runSessionWriteLoop(ctx, conn, c)

	resizeCh := sess.resizeCh
	if resizeCh == nil {
		resizeCh = make(chan containermanager.WinSize, 1) // upgrade sessions have no PTY resize target
	}

	reason := "session ended"
	if err := runSessionReadLoop(ctx, conn, sess, c, resizeCh); err != nil {
		reason = "connection closed"
	}
	conn.Close(websocket.StatusNormalClosure, reason)
}
