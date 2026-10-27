package serve

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

	"github.com/eljojo/rememory/internal/core"
	"github.com/eljojo/rememory/internal/html"
)

const (
	maxAuthBody    = 4 << 10 // setup and delete bodies: an ID and a password
	maxPasswordLen = 1 << 10

	maxUploadOverhead = 64 << 10 // multipart framing and the meta field
	maxMetaSize       = 4 << 10
	maxBundleName     = 256
)

func (s *Server) handleRoot(w http.ResponseWriter, r *http.Request) {
	if !IsSetup(s.store) {
		s.handleSetupPage(w, r)
		return
	}
	bundles, _ := s.store.List()
	if bundles == nil {
		bundles = []BundleMeta{}
	}
	bundlesJSON := html.HomeBundlesJSON(bundles)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, html.GenerateHomeHTML(bundlesJSON))
}

func (s *Server) handleSetupPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, html.GenerateSetupHTML())
}

func (s *Server) handleCreate(w http.ResponseWriter, r *http.Request) {
	if !IsSetup(s.store) {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	createWASM := html.GetCreateWASMBytes()
	if len(createWASM) == 0 {
		http.Error(w, "create.wasm not embedded — rebuild with 'make build'", http.StatusInternalServerError)
		return
	}

	content := html.GenerateMakerHTML(createWASM, html.MakerHTMLOptions{
		Selfhosted: true,
		SelfhostedConfig: &html.SelfhostedConfig{
			MaxManifestSize: s.maxManifestSize,
			HasManifest:     s.store.HasManifest(),
		},
	})
	fmt.Fprint(w, content)
}

func (s *Server) handleRecover(w http.ResponseWriter, r *http.Request) {
	if !IsSetup(s.store) {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	// Resolve the bundle ID: use the query param if provided, otherwise fall
	// back to the only bundle, so a bare /recover works for a household with
	// one. With several, the newest could be anyone's upload; the friend picks
	// from the home page or adds MANIFEST.age themselves.
	var manifestURL string
	if id := r.URL.Query().Get("id"); id != "" {
		manifestURL = "/api/bundle/manifest?id=" + url.QueryEscape(id)
	} else if bundles, err := s.store.List(); err == nil && len(bundles) == 1 {
		manifestURL = "/api/bundle/manifest?id=" + url.QueryEscape(bundles[0].ID)
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	content := html.GenerateRecoverHTML(nil, html.RecoverHTMLOptions{
		Selfhosted: true,
		SelfhostedConfig: &html.SelfhostedConfig{
			MaxManifestSize: s.maxManifestSize,
			HasManifest:     s.store.HasManifest(),
			ManifestURL:     manifestURL,
		},
	})
	fmt.Fprint(w, content)
}

func (s *Server) handleAbout(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	content := html.GenerateIndexHTML(true)
	fmt.Fprint(w, content)
}

func (s *Server) docsHandler(lang string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, html.GenerateDocsHTML(lang, true))
	}
}

// API handlers

func (s *Server) handleAPIStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"hasPassword":     IsSetup(s.store),
		"hasManifest":     s.store.HasManifest(),
		"maxManifestSize": s.maxManifestSize,
	})
}

func (s *Server) handleAPISetup(w http.ResponseWriter, r *http.Request) {
	s.authMu.Lock()
	defer s.authMu.Unlock()

	if IsSetup(s.store) {
		http.Error(w, "Admin password is already set.", http.StatusConflict)
		return
	}

	// Anyone can reach this endpoint before setup, so keep the body small.
	r.Body = http.MaxBytesReader(w, r.Body, maxAuthBody)
	var body struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Invalid request body.", http.StatusBadRequest)
		return
	}

	if body.Password == "" {
		http.Error(w, "Password cannot be empty.", http.StatusBadRequest)
		return
	}
	if len(body.Password) > maxPasswordLen {
		http.Error(w, "Password is too long.", http.StatusBadRequest)
		return
	}

	if err := SetPassword(s.store, body.Password); err != nil {
		if errors.Is(err, ErrAlreadySet) {
			http.Error(w, "Admin password is already set.", http.StatusConflict)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"ok": true})
}

func (s *Server) handleAPIListBundles(w http.ResponseWriter, r *http.Request) {
	bundles, err := s.store.List()
	if err != nil {
		http.Error(w, "Could not list bundles.", http.StatusInternalServerError)
		return
	}
	if bundles == nil {
		bundles = []BundleMeta{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(bundles)
}

func (s *Server) handleAPISaveBundle(w http.ResponseWriter, r *http.Request) {
	if !IsSetup(s.store) {
		http.Error(w, "Set up an admin password first.", http.StatusForbidden)
		return
	}

	// Cap the whole request: ParseMultipartForm's argument only sets how much
	// is kept in memory, and spools the rest to disk without limit.
	r.Body = http.MaxBytesReader(w, r.Body, int64(s.maxManifestSize)+maxUploadOverhead)

	// Parse multipart form (manifest file + metadata JSON)
	if err := r.ParseMultipartForm(int64(s.maxManifestSize) + maxUploadOverhead); err != nil {
		http.Error(w, "Could not parse form.", http.StatusBadRequest)
		return
	}

	file, _, err := r.FormFile("manifest")
	if err != nil {
		http.Error(w, "Missing manifest file.", http.StatusBadRequest)
		return
	}
	defer file.Close()

	manifest, err := io.ReadAll(io.LimitReader(file, int64(s.maxManifestSize)+1))
	if err != nil {
		http.Error(w, "Could not read manifest.", http.StatusBadRequest)
		return
	}
	if len(manifest) > s.maxManifestSize {
		http.Error(w, fmt.Sprintf("Manifest too large (%d bytes). Maximum is %d bytes.",
			len(manifest), s.maxManifestSize), http.StatusRequestEntityTooLarge)
		return
	}

	// Validate that the upload is a recognizable age-encrypted file:
	// either a plain age file (header prefix) or a tlock container (ZIP with tlock.json).
	const ageHeader = "age-encryption.org/v1\n"
	isAge := len(manifest) >= len(ageHeader) && string(manifest[:len(ageHeader)]) == ageHeader
	if !isAge && !core.IsTlockContainer(manifest) {
		http.Error(w, "Not a valid age-encrypted file.", http.StatusBadRequest)
		return
	}

	// Uploads are anonymous and every page view reads every bundle's
	// metadata, so it has to stay small.
	metaStr := r.FormValue("meta")
	if len(metaStr) > maxMetaSize {
		http.Error(w, "Metadata too large.", http.StatusBadRequest)
		return
	}
	var meta BundleMeta
	if metaStr != "" {
		if err := json.Unmarshal([]byte(metaStr), &meta); err != nil {
			http.Error(w, "Invalid metadata.", http.StatusBadRequest)
			return
		}
	}
	if len(meta.Name) > maxBundleName || meta.Threshold < 0 || meta.Threshold > 255 || meta.Total < 0 || meta.Total > 255 {
		http.Error(w, "Invalid metadata.", http.StatusBadRequest)
		return
	}

	id, err := s.store.Save(manifest, meta)
	if err != nil {
		http.Error(w, "Could not save bundle.", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"id": id})
}

func (s *Server) handleAPIDeleteBundle(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxAuthBody)
	var body struct {
		ID       string `json:"id"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Invalid request body.", http.StatusBadRequest)
		return
	}

	if body.ID == "" {
		http.Error(w, "Missing bundle ID.", http.StatusBadRequest)
		return
	}

	if !isValidUUID(body.ID) {
		http.Error(w, "Invalid bundle ID.", http.StatusBadRequest)
		return
	}

	// Cheap checks first: every password check is a 256 MiB scrypt.
	if len(body.Password) > maxPasswordLen {
		http.Error(w, "Password is too long.", http.StatusBadRequest)
		return
	}
	// The bundle's folder, not its MANIFEST.age: an interrupted save can
	// leave a listed bundle without one, and it must still be deletable.
	if _, err := os.Stat(filepath.Dir(s.store.ManifestPath(body.ID))); err != nil {
		http.Error(w, "Bundle not found.", http.StatusNotFound)
		return
	}

	s.authMu.Lock()
	ok := CheckPassword(s.store, body.Password)
	s.authMu.Unlock()
	if !ok {
		http.Error(w, "Wrong password.", http.StatusUnauthorized)
		return
	}

	if err := s.store.Delete(body.ID); err != nil {
		http.Error(w, "Bundle not found.", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"ok": true})
}

func (s *Server) handleAPIManifest(w http.ResponseWriter, r *http.Request) {
	bundleID := r.URL.Query().Get("id")
	if bundleID == "" {
		http.Error(w, "Missing bundle ID.", http.StatusBadRequest)
		return
	}
	if !isValidUUID(bundleID) {
		http.Error(w, "Invalid bundle ID.", http.StatusBadRequest)
		return
	}

	manifestPath := s.store.ManifestPath(bundleID)
	f, err := os.Open(manifestPath)
	if err != nil {
		http.Error(w, "Manifest not found.", http.StatusNotFound)
		return
	}
	defer f.Close()

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", "attachment; filename=MANIFEST.age")
	io.Copy(w, f)
}
