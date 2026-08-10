// Package webui implements otter's local web dashboard: a small HTTP server
// that lists containers and lets you start/stop/pause/remove them, plus a
// browser-based terminal for entering a container. It reuses pkg/commands —
// the same layer the CLI is built on — so behavior stays in one place.
package webui

import (
	"context"
	"crypto/subtle"
	"embed"
	"errors"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"net/url"
	"time"

	"github.com/ferret-linux/otter/pkg/config"
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
	cfg       *config.Values
	rootful   bool
	templates *template.Template
	token     string
}

// webUISessionCookie is the name of the cookie set once a request presents
// a valid ?token= query parameter, so subsequent requests — including the
// websocket terminal's upgrade handshake, which carries cookies but can't
// set custom headers from browser JS — are authenticated without the token
// needing to appear in every URL.
const webUISessionCookie = "otter_webui_session"

// Serve starts the otter webui HTTP server on addr and blocks until ctx is
// canceled or the server fails. On cancellation it shuts the server down
// gracefully.
func Serve(ctx context.Context, cm containermanager.ContainerManager, cfg *config.Values, rootful bool, addr string, token string) error {
	tmpl, err := template.ParseFS(templateFS, "templates/*.html")
	if err != nil {
		return fmt.Errorf("failed to parse webui templates: %w", err)
	}

	staticSub, err := fs.Sub(staticFS, "static")
	if err != nil {
		return fmt.Errorf("failed to prepare webui static assets: %w", err)
	}

	s := &server{cm: cm, cfg: cfg, rootful: rootful, templates: tmpl, token: token}

	mux := http.NewServeMux()
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServerFS(staticSub)))
	mux.HandleFunc("GET /{$}", s.index)
	mux.HandleFunc("GET /containers/new", s.newContainerPage)
	mux.HandleFunc("POST /containers", s.createContainer)
	mux.HandleFunc("POST /containers/{name}/{action}", s.action)
	mux.HandleFunc("GET /containers/{name}/inspect", s.inspectPage)
	mux.HandleFunc("GET /containers/{name}/terminal", s.terminalPage)
	mux.HandleFunc("GET /ws/containers/{name}/terminal", s.terminalWS)
	mux.HandleFunc("GET /containers/{name}/logs", s.logsPage)
	mux.HandleFunc("GET /sse/containers/{name}/logs", s.logsSSE)

	httpServer := &http.Server{
		Addr:              addr,
		Handler:           s.withAuth(mux),
		ReadHeaderTimeout: 10 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		ui.DefaultLogger.Info("webui listening", "url", fmt.Sprintf("http://%s/?token=%s", addr, url.QueryEscape(token)))
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

// withAuth gates every request behind s.token. A request is authenticated
// either by an existing session cookie matching the token, or by a
// ?token=... query parameter matching it — in which case withAuth sets the
// session cookie and redirects to the same URL with the token stripped, so
// it doesn't linger in browser history or get echoed back in a Referer
// header on subsequent navigation.
func (s *server) withAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if cookie, err := r.Cookie(webUISessionCookie); err == nil &&
			subtle.ConstantTimeCompare([]byte(cookie.Value), []byte(s.token)) == 1 {
			next.ServeHTTP(w, r)
			return
		}

		if t := r.URL.Query().Get("token"); t != "" &&
			subtle.ConstantTimeCompare([]byte(t), []byte(s.token)) == 1 {
			http.SetCookie(w, &http.Cookie{
				Name:     webUISessionCookie,
				Value:    s.token,
				Path:     "/",
				HttpOnly: true,
				SameSite: http.SameSiteStrictMode,
			})
			redirectURL := *r.URL
			q := redirectURL.Query()
			q.Del("token")
			redirectURL.RawQuery = q.Encode()
			http.Redirect(w, r, redirectURL.String(), http.StatusFound)
			return
		}

		http.Error(w, "unauthorized: open the URL printed at webui startup (it includes ?token=...) to establish a session", http.StatusUnauthorized)
	})
}
