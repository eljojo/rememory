package serve

import (
	"fmt"
	"io/fs"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/eljojo/rememory/docs"
	"github.com/eljojo/rememory/internal/html"
)

// Config holds the configuration for the server.
type Config struct {
	Host            string
	Port            string
	DataDir         string
	MaxManifestSize int // Maximum MANIFEST.age size in bytes
	Version         string
	BuildDate       string
}

// Server implements http.Handler for the self-hosted ReMemory web app.
type Server struct {
	store           *Store
	maxManifestSize int
	version         string
	// reachable: listening beyond loopback. Such a server gets its admin
	// password at startup and never offers setup.
	reachable bool
	mux       *http.ServeMux
	handler   http.Handler

	// authMu serializes admin setup and password checks: each runs a 256 MiB
	// scrypt, so concurrent requests queue instead of exhausting memory.
	authMu sync.Mutex
}

// New creates a new Server from the given config.
func New(cfg Config) (*Server, error) {
	store, err := NewStore(cfg.DataDir)
	if err != nil {
		return nil, fmt.Errorf("initializing store: %w", err)
	}

	html.SetVersion(cfg.Version)
	html.SetBuildDate(cfg.BuildDate)

	s := &Server{
		store:           store,
		maxManifestSize: cfg.MaxManifestSize,
		version:         cfg.Version,
		reachable:       !isLoopback(cfg.Host),
		mux:             http.NewServeMux(),
	}

	s.routes()
	// Refuse cross-site state-changing requests (setup, upload, delete), so
	// a page the owner visits can't act on a server reachable from their
	// browser. Same-origin requests and non-browser clients pass.
	s.handler = http.NewCrossOriginProtection().Handler(s.mux)
	return s, nil
}

// routes registers all HTTP routes.
func (s *Server) routes() {
	// Pages (filename-based routes)
	s.mux.HandleFunc("GET /", s.handleRoot)
	s.mux.HandleFunc("GET /maker.html", s.handleCreate)
	s.mux.HandleFunc("GET /recover.html", s.handleRecover)
	s.mux.HandleFunc("GET /about.html", s.handleAbout)
	s.mux.HandleFunc("GET /docs.html", s.docsHandler("en"))
	for _, lang := range html.DocsLanguages() {
		s.mux.HandleFunc("GET /docs."+lang+".html", s.docsHandler(lang))
	}

	// Redirect old clean routes to filename routes (backward compat).
	// Query strings are preserved so /recover?id=X → /recover.html?id=X.
	for _, r := range [][2]string{
		{"/create", "/maker.html"},
		{"/recover", "/recover.html"},
		{"/about", "/about.html"},
		{"/docs", "/docs.html"},
		{"/index.html", "/about.html"},
	} {
		from, to := r[0], r[1]
		s.mux.HandleFunc("GET "+from, func(w http.ResponseWriter, r *http.Request) {
			target := to
			if r.URL.RawQuery != "" {
				target += "?" + r.URL.RawQuery
			}
			http.Redirect(w, r, target, http.StatusMovedPermanently)
		})
	}

	// Static assets (docs screenshots, embedded in binary).
	// Only English screenshots are embedded. Non-English language paths
	// fall back to English (e.g. /screenshots/de/friends.png → en/friends.png).
	screenshotsFS, _ := fs.Sub(docs.ScreenshotsFS, "screenshots")
	fileServer := http.FileServer(http.FS(screenshotsFS))
	s.mux.HandleFunc("GET /screenshots/", func(w http.ResponseWriter, r *http.Request) {
		// Try the requested path first
		path := strings.TrimPrefix(r.URL.Path, "/screenshots/")
		if _, err := fs.Stat(screenshotsFS, path); err != nil {
			// Not found — try English fallback: replace leading lang dir with "en"
			parts := strings.SplitN(path, "/", 2)
			if len(parts) == 2 {
				enPath := "en/" + parts[1]
				if _, err := fs.Stat(screenshotsFS, enPath); err == nil {
					r.URL.Path = "/screenshots/" + enPath
				}
			}
		}
		http.StripPrefix("/screenshots/", fileServer).ServeHTTP(w, r)
	})

	// API
	s.mux.HandleFunc("GET /api/status", s.handleAPIStatus)
	s.mux.HandleFunc("POST /api/setup", s.handleAPISetup)
	s.mux.HandleFunc("GET /api/bundles", s.handleAPIListBundles)
	s.mux.HandleFunc("POST /api/bundle", s.handleAPISaveBundle)
	s.mux.HandleFunc("DELETE /api/bundle", s.handleAPIDeleteBundle)
	s.mux.HandleFunc("GET /api/bundle/manifest", s.handleAPIManifest)
}

// ServeHTTP implements http.Handler.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.handler.ServeHTTP(w, r)
}

// ListenAndServe starts the HTTP server.
func (s *Server) ListenAndServe(addr string) error {
	return s.httpServer(addr).ListenAndServe()
}

// httpServer bounds how long a client may take to send its headers or keep
// an idle connection. Whole-request read/write timeouts are left unset so
// large manifests can still be uploaded and downloaded over slow links.
func (s *Server) httpServer(addr string) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           s,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       2 * time.Minute,
	}
}

// Store returns the server's store.
func (s *Server) Store() *Store {
	return s.store
}
