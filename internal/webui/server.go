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
	"sync"
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
	cm      containermanager.ContainerManager
	rootful bool

	// cfgMu guards cfg. Config is loaded once at startup but can change
	// after a webui Settings save (see settingsSave), so every read goes
	// through s.config() and every write through s.setConfig() rather
	// than accessing cfg directly, to avoid a data race between a save on
	// one request and a read on another concurrent request.
	cfgMu sync.RWMutex
	cfg   *config.Values

	templates *template.Template
	token     string

	// upgradeStreamsMu guards upgradeStreams, the set of container names
	// currently running an upgrade started from this webui process. Actual
	// upgrade-in-progress status for row/inspect rendering comes from
	// s.cm.IsUpgrading (the container.upgrading marker file, see
	// containermanager.ContainerManager), which is authoritative and
	// survives webui restarts; this map only exists to let a live viewer
	// attach to a specific run's output, and is best-effort — it holds
	// nothing that outlives the webui process.
	upgradeStreamsMu sync.Mutex
	upgradeStreams   map[string]*upgradeStream

	// pullingMu guards pulling, the set of image names currently being
	// pulled from the registry panel via a webui-initiated
	// commands.RegistryPullCommand. Unlike upgrades, there's no on-disk
	// marker for an in-progress image pull, so — unlike upgradeStreams's
	// relationship to s.cm.IsUpgrading — this map is the only record of
	// pull-in-progress state and does not survive a webui restart.
	pullingMu sync.Mutex
	pulling   map[string]bool

	// manifestSessionsMu guards manifestSessions, the set of in-progress
	// Manifests-page sessions (see internal/webui/manifests.go): one per
	// parsed manifest, kept alive between its preview and apply requests.
	// Best-effort, in-memory only — a webui restart mid-session just means
	// the person has to re-upload or re-enter the manifest URL.
	manifestSessionsMu sync.Mutex
	manifestSessions   map[string]*manifestSession

	// manifestApplyingMu guards manifestApplying, the set of session IDs
	// currently running a detached commands.AssembleCommand.Execute from
	// the Manifests page's Apply action (see manifestsApply), mirroring
	// how s.pulling tracks in-progress registry pulls above.
	manifestApplyingMu sync.Mutex
	manifestApplying   map[string]bool

	// notifications buffers and fans out user-facing notifications (action
	// failures and successes) to every connected /sse/notifications
	// viewer — see notifications.go. Like the maps above, it's in-memory
	// only and does not survive a webui restart.
	notifications *notificationHub
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

	s := &server{
		cm: cm, rootful: rootful, templates: tmpl, token: token,
		upgradeStreams:   make(map[string]*upgradeStream),
		pulling:          make(map[string]bool),
		manifestSessions: make(map[string]*manifestSession),
		manifestApplying: make(map[string]bool),
		notifications:    newNotificationHub(),
	}
	s.setConfig(cfg)

	mux := http.NewServeMux()
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServerFS(staticSub)))
	mux.HandleFunc("GET /{$}", s.index)
	mux.HandleFunc("GET /containers", s.containerList)
	mux.HandleFunc("GET /containers/{name}/panel", s.containerPanel)
	mux.HandleFunc("POST /containers", s.createContainer)
	mux.HandleFunc("POST /containers/{name}/{action}", s.action)
	mux.HandleFunc("GET /ws/containers/{name}/terminal", s.terminalWS)
	mux.HandleFunc("GET /sse/containers/{name}/logs", s.logsSSE)
	mux.HandleFunc("POST /containers/{name}/upgrade", s.upgradeAction)
	mux.HandleFunc("GET /sse/containers/{name}/upgrade", s.upgradeSSE)
	mux.HandleFunc("GET /create", s.createPage)
	mux.HandleFunc("GET /registry", s.registryPage)
	mux.HandleFunc("POST /registry/{name}/pull", s.registryPullAction)
	mux.HandleFunc("POST /registry/{name}/remove", s.registryRemoveAction)
	mux.HandleFunc("GET /manifests", s.manifestsPage)
	mux.HandleFunc("POST /manifests/parse", s.manifestsParse)
	mux.HandleFunc("POST /manifests/{sessionID}/apply", s.manifestsApply)
	mux.HandleFunc("GET /console", s.consolePage)
	mux.HandleFunc("GET /console/{name}/shell", s.consoleShellFragment)
	mux.HandleFunc("GET /console/{name}/watch", s.consoleWatchFragment)
	mux.HandleFunc("GET /logs", s.logsListPage)
	mux.HandleFunc("GET /logs/{name}/view", s.logsViewFragment)
	mux.HandleFunc("GET /docs", s.docsPage)
	mux.HandleFunc("GET /docs/content", s.docsContentFragment)
	mux.HandleFunc("GET /settings", s.settingsPage)
	mux.HandleFunc("POST /settings/save", s.settingsSave)
	mux.HandleFunc("GET /sse/notifications", s.notificationsSSE)
	mux.HandleFunc("DELETE /notifications/{id}", s.notificationDeleteAction)
	mux.HandleFunc("DELETE /notifications", s.notificationClearAction)

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

// config returns the current in-memory config, safe for concurrent use
// alongside setConfig.
func (s *server) config() *config.Values {
	s.cfgMu.RLock()
	defer s.cfgMu.RUnlock()
	return s.cfg
}

// setConfig replaces the in-memory config, safe for concurrent use
// alongside config. Called once at startup and again after every
// successful Settings save (see settingsSave) so already-running handlers
// see a saved change without a webui restart.
func (s *server) setConfig(cfg *config.Values) {
	s.cfgMu.Lock()
	defer s.cfgMu.Unlock()
	s.cfg = cfg
}
