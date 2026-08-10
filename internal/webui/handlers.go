package webui

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/coder/websocket"

	"github.com/ferret-linux/otter/pkg/commands"
	"github.com/ferret-linux/otter/pkg/containermanager"
)

func (s *server) index(w http.ResponseWriter, r *http.Request) {
	result, err := commands.NewListCommand(s.cm).Execute(r.Context(), commands.ListOptions{})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := s.templates.ExecuteTemplate(w, "layout", result.Containers); err != nil {
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
			if err := s.templates.ExecuteTemplate(w, "row", c); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
			return
		}
	}
	w.WriteHeader(http.StatusOK)
}

func (s *server) terminalPage(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if err := s.templates.ExecuteTemplate(w, "terminal", name); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *server) logsPage(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if err := s.templates.ExecuteTemplate(w, "logs", name); err != nil {
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

func (s *server) inspectPage(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")

	result, err := commands.NewInspectCommand(s.cm).Execute(r.Context(), commands.InspectOptions{
		ContainerName: name,
		Manager:       s.cm.Name(),
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := s.templates.ExecuteTemplate(w, "inspect", result); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
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

func (s *server) newContainerPage(w http.ResponseWriter, r *http.Request) {
	data := createFormData{
		GenerateEntry: true,
		DefaultImage:  s.cfg.DefaultContainerImage,
	}
	if err := s.templates.ExecuteTemplate(w, "create", data); err != nil {
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
	if err := s.templates.ExecuteTemplate(w, "create", data); err != nil {
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
