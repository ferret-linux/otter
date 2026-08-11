package webui

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/ferret-linux/otter/pkg/commands"
	"github.com/ferret-linux/otter/pkg/manifest"
	"github.com/ferret-linux/otter/pkg/ui"
)

// manifestSession holds one parsed manifest between the Manifests page's
// preview and apply requests (see manifestsParse and manifestsApply). The
// parse -> select boxes -> apply flow spans multiple separate HTTP
// requests, so the manifest source needs to survive between them:
// commands.AssembleCommand.Execute re-parses the manifest itself given a
// path or URL, it doesn't take already-parsed items.
type manifestSession struct {
	// Source is what gets passed as AssembleOptions.ManifestPath: either
	// the user-supplied URL, or the path of a server-side temp file
	// holding an uploaded manifest's bytes.
	Source string
	// TempFile is true if Source is a temp file this session owns and
	// must remove once the session ends (see manifestsApply's cleanup),
	// as opposed to a URL, which owns nothing to clean up.
	TempFile bool
}

// manifestBoxView is one manifest box as shown on the Manifests page: its
// name plus enough fields to preview what applying it would do. For a box
// that doesn't exist yet, Image/Rootful/Init come from the parsed
// manifest.Item (already include-merged and bool-defaulted by
// manifest.Parse). For a box that already exists, they come from its
// actual current state via commands.InspectCommand instead — not a
// preview of what Replace would produce.
type manifestBoxView struct {
	Name    string
	Exists  bool
	Image   string
	Rootful bool
	Init    bool
}

// manifestsPageData is the Manifests page's view model, covering all three
// states it can render: the initial empty form, a parse error, and a
// successful preview with boxes to select from.
type manifestsPageData struct {
	pageData
	SessionID string
	Boxes     []manifestBoxView
	Error     string
}

// newManifestSessionID returns a random, URL-safe session identifier,
// generated the same way resolveWebUIToken (internal/cli/webui.go)
// generates the webui access token.
func newManifestSessionID() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("failed to generate session id: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

// manifestsPage handles GET /manifests: the empty landing state, with a
// URL field and a file upload for supplying a manifest.
func (s *server) manifestsPage(w http.ResponseWriter, r *http.Request) {
	data := manifestsPageData{pageData: pageData{Nav: "manifests", PageTitle: "manifests"}}
	if err := s.templates.ExecuteTemplate(w, "layout", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// manifestSource extracts a manifest source (URL or uploaded file) from a
// POST /manifests/parse request, returning the string to hand to
// manifest.Parse and whether it's a temp file the caller now owns.
//
// Browsers never expose a real filesystem path from a file input — only
// bytes and a bare filename — so an uploaded file's bytes are written to a
// server-side temp file first; manifest.Parse (pkg/manifest/fetcher.go)
// already accepts a plain path or a URL, so no changes were needed there.
func (s *server) manifestSource(r *http.Request) (source string, tempFile bool, err error) {
	if err := r.ParseMultipartForm(32 << 20); err != nil && !errors.Is(err, http.ErrNotMultipart) {
		return "", false, fmt.Errorf("failed to parse form: %w", err)
	}

	if u := r.FormValue("url"); u != "" {
		return u, false, nil
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		if errors.Is(err, http.ErrMissingFile) {
			return "", false, errors.New("provide a manifest URL or upload a file")
		}
		return "", false, fmt.Errorf("failed to read uploaded file: %w", err)
	}
	defer file.Close()

	tmp, err := os.CreateTemp("", "otter-manifest-*.toml")
	if err != nil {
		return "", false, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer tmp.Close()

	if _, err := io.Copy(tmp, file); err != nil {
		os.Remove(tmp.Name())
		return "", false, fmt.Errorf("failed to save uploaded file: %w", err)
	}

	return tmp.Name(), true, nil
}

// buildManifestBoxViews cross-references each parsed manifest item against
// the current container list (the same list commands.NewListCommand
// backs the Home page with) to tag it New or Existing, and fills in the
// preview fields described on manifestBoxView.
func (s *server) buildManifestBoxViews(ctx context.Context, items []manifest.Item) ([]manifestBoxView, error) {
	rows, err := s.listRows(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list existing containers: %w", err)
	}
	existing := make(map[string]bool, len(rows))
	for _, row := range rows {
		existing[row.Name] = true
	}

	boxes := make([]manifestBoxView, len(items))
	for i, item := range items {
		view := manifestBoxView{Name: item.Name, Exists: existing[item.Name]}
		if view.Exists {
			result, err := commands.NewInspectCommand(s.cm).Execute(ctx, commands.InspectOptions{ContainerName: item.Name})
			if err != nil {
				return nil, fmt.Errorf("failed to inspect existing box '%s': %w", item.Name, err)
			}
			view.Image = result.Image
			view.Rootful = result.Rootful
			view.Init = result.Init
		} else {
			// manifest.Parse guarantees Settings.Rootful and
			// Settings.InitSystem are non-nil by the time Item values are
			// returned (see the doc comment on manifest.Settings), so
			// dereferencing here is safe without a nil check, matching
			// how pkg/commands/assemble.go's createItem already does it.
			view.Image = item.Image
			view.Rootful = *item.Settings.Rootful
			view.Init = *item.Settings.InitSystem
		}
		boxes[i] = view
	}
	return boxes, nil
}

// renderManifestsError re-renders the Manifests page's result fragment
// with an error message, for htmx to swap into #manifests-result.
func (s *server) renderManifestsError(w http.ResponseWriter, msg string) {
	w.WriteHeader(http.StatusUnprocessableEntity)
	data := manifestsPageData{
		pageData: pageData{Nav: "manifests", PageTitle: "manifests"},
		Error:    msg,
	}
	if err := s.templates.ExecuteTemplate(w, "manifests_result", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// manifestsParse handles POST /manifests/parse. It accepts either a "url"
// form field or an uploaded "file", parses the manifest via the existing,
// unmodified manifest.Parse, tags each box New or Existing (see
// buildManifestBoxViews), stores a session for the later apply step, and
// renders the box-preview fragment.
func (s *server) manifestsParse(w http.ResponseWriter, r *http.Request) {
	source, tempFile, err := s.manifestSource(r)
	if err != nil {
		s.renderManifestsError(w, err.Error())
		return
	}

	items, err := manifest.Parse(r.Context(), source)
	if err != nil {
		if tempFile {
			os.Remove(source)
		}
		s.renderManifestsError(w, fmt.Sprintf("failed to parse manifest: %s", err))
		return
	}
	if len(items) == 0 {
		if tempFile {
			os.Remove(source)
		}
		s.renderManifestsError(w, "manifest defines no boxes")
		return
	}

	boxes, err := s.buildManifestBoxViews(r.Context(), items)
	if err != nil {
		if tempFile {
			os.Remove(source)
		}
		s.renderManifestsError(w, err.Error())
		return
	}

	sessionID, err := newManifestSessionID()
	if err != nil {
		if tempFile {
			os.Remove(source)
		}
		s.renderManifestsError(w, err.Error())
		return
	}

	s.manifestSessionsMu.Lock()
	s.manifestSessions[sessionID] = &manifestSession{Source: source, TempFile: tempFile}
	s.manifestSessionsMu.Unlock()

	data := manifestsPageData{
		pageData:  pageData{Nav: "manifests", PageTitle: "manifests"},
		SessionID: sessionID,
		Boxes:     boxes,
	}
	if err := s.templates.ExecuteTemplate(w, "manifests_result", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// manifestsApply handles POST /manifests/{sessionID}/apply. Selected box
// names plus a verb (create/replace/remove) become a single
// commands.AssembleOptions{BoxNames: selected} call, run in a background
// goroutine detached from this request's context — the same pattern
// registryPullAction uses for image pulls, chosen over live progress
// streaming since ui.Progress currently writes through the global
// ui.DefaultLogger rather than through an injectable per-request writer,
// so there is nothing today for a per-request SSE stream to consume (see
// pkg/ui/progress.go). The request returns immediately; picking up
// completion requires reloading or re-submitting the manifest.
//
// If an apply is already running for this session, this is a no-op: it
// does not start a second concurrent run.
func (s *server) manifestsApply(w http.ResponseWriter, r *http.Request) {
	sessionID := r.PathValue("sessionID")

	s.manifestSessionsMu.Lock()
	sess, ok := s.manifestSessions[sessionID]
	s.manifestSessionsMu.Unlock()
	if !ok {
		http.Error(w, "manifest session expired or not found; re-submit the manifest", http.StatusNotFound)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	boxNames := r.Form["boxes"]
	if len(boxNames) == 0 {
		http.Error(w, "select at least one box", http.StatusBadRequest)
		return
	}

	opts := commands.AssembleOptions{
		ManifestPath: sess.Source,
		SudoCommand:  s.config().SudoProgram,
		BoxNames:     boxNames,
	}
	switch r.FormValue("verb") {
	case "remove":
		opts.Delete = true
	case "replace":
		opts.Replace = true
	case "create":
		// zero value: create-or-update, matching AssembleCommand's default.
	default:
		http.Error(w, "unknown verb", http.StatusBadRequest)
		return
	}

	s.manifestApplyingMu.Lock()
	alreadyApplying := s.manifestApplying[sessionID]
	if !alreadyApplying {
		s.manifestApplying[sessionID] = true
	}
	s.manifestApplyingMu.Unlock()

	if !alreadyApplying {
		go func() {
			defer func() {
				s.manifestApplyingMu.Lock()
				delete(s.manifestApplying, sessionID)
				s.manifestApplyingMu.Unlock()

				s.manifestSessionsMu.Lock()
				delete(s.manifestSessions, sessionID)
				s.manifestSessionsMu.Unlock()

				if sess.TempFile {
					os.Remove(sess.Source)
				}
			}()
			if err := commands.NewAssembleCommand(s.config(), s.cm).Execute(context.Background(), opts); err != nil {
				ui.DefaultLogger.Error("failed to apply manifest selection", "session", sessionID, "err", err)
			}
		}()
	}

	data := manifestsPageData{pageData: pageData{Nav: "manifests", PageTitle: "manifests"}}
	if err := s.templates.ExecuteTemplate(w, "manifests_applying", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
