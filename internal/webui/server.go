// Package webui implements otter's local web dashboard: a small HTTP server
// that lists containers and lets you start/stop/pause/remove them, plus a
// browser-based terminal for entering a container. It reuses pkg/commands —
// the same layer the CLI is built on — so behavior stays in one place.
package webui

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"time"

	"github.com/ferret-linux/otter/pkg/containermanager"
	"github.com/ferret-linux/otter/pkg/ui"
)

//go:embed templates/*.html
var templateFS embed.FS

//go:embed static
var staticFS embed.FS

const shutdownTimeout = 5 * time.Second

type server struct {
	cm        containermanager.ContainerManager
	templates *template.Template
}

// Serve starts the otter webui HTTP server on addr and blocks until ctx is
// canceled or the server fails. On cancellation it shuts the server down
// gracefully.
func Serve(ctx context.Context, cm containermanager.ContainerManager, addr string) error {
	tmpl, err := template.ParseFS(templateFS, "templates/*.html")
	if err != nil {
		return fmt.Errorf("failed to parse webui templates: %w", err)
	}

	staticSub, err := fs.Sub(staticFS, "static")
	if err != nil {
		return fmt.Errorf("failed to prepare webui static assets: %w", err)
	}

	s := &server{cm: cm, templates: tmpl}

	mux := http.NewServeMux()
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServerFS(staticSub)))
	mux.HandleFunc("GET /{$}", s.index)
	mux.HandleFunc("POST /containers/{name}/{action}", s.action)
	mux.HandleFunc("GET /containers/{name}/terminal", s.terminalPage)
	mux.HandleFunc("GET /ws/containers/{name}/terminal", s.terminalWS)

	httpServer := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		ui.DefaultLogger.Info("webui listening", "addr", "http://"+addr)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("failed to shut down webui server: %w", err)
		}
		return nil
	case err := <-errCh:
		return err
	}
}
