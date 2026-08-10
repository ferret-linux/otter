package webui

import (
	"net/http"

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
