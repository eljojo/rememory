package serve

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Behind a reverse proxy the server listens on 127.0.0.1, so it doesn't
// generate its admin password, yet anyone can reach the setup page through
// the proxy. Setup requests that came through a proxy are refused.
func TestSetupThroughProxyIsRefused(t *testing.T) {
	for _, header := range []string{"X-Forwarded-For", "X-Real-IP", "Forwarded", "X-Forwarded-Host"} {
		srv := newTestServer(t)
		req := httptest.NewRequest("POST", "/api/setup", strings.NewReader(`{"password":"stranger-password"}`))
		req.Header.Set(header, "203.0.113.7")
		w := httptest.NewRecorder()
		srv.ServeHTTP(w, req)
		if w.Code != http.StatusForbidden || IsSetup(srv.Store()) {
			t.Errorf("setup with %s: got %d (set up: %v), want 403", header, w.Code, IsSetup(srv.Store()))
		}
		if !strings.Contains(w.Body.String(), "REMEMORY_ADMIN_PASSWORD") {
			t.Errorf("setup with %s: the refusal should say how to set the password: %q", header, w.Body.String())
		}
	}
}

// DNS rebinding: a page on another site points its own name at 127.0.0.1,
// and the browser then sends a same-origin request with that name as Host.
// A browser may only run setup when it addresses this machine by name.
func TestSetupFromBrowserNeedsLocalHostName(t *testing.T) {
	post := func(host, origin string) (*Server, int) {
		srv := newTestServer(t)
		req := httptest.NewRequest("POST", "/api/setup", strings.NewReader(`{"password":"some-password"}`))
		req.Host = host
		if origin != "" {
			req.Header.Set("Origin", origin)
			req.Header.Set("Sec-Fetch-Site", "same-origin")
		}
		w := httptest.NewRecorder()
		srv.ServeHTTP(w, req)
		return srv, w.Code
	}

	if srv, code := post("evil.example:8080", "http://evil.example:8080"); code != http.StatusForbidden || IsSetup(srv.Store()) {
		t.Errorf("rebound browser request: got %d, want 403", code)
	}
	for _, host := range []string{"localhost:8080", "127.0.0.1:8080", "[::1]:8080"} {
		if _, code := post(host, "http://"+host); code != http.StatusOK {
			t.Errorf("browser on %s: got %d, want 200", host, code)
		}
	}
	if _, code := post("example.com", ""); code != http.StatusOK {
		t.Errorf("non-browser client: got %d, want 200", code)
	}
}

// A server reachable from other machines gets its admin password at startup
// (generated, or REMEMORY_ADMIN_PASSWORD). If admin.age disappears while it
// runs, setup must stay closed rather than wait for the first visitor.
func TestReachableServerRefusesSetup(t *testing.T) {
	srv, err := New(Config{Host: "0.0.0.0", DataDir: t.TempDir(), MaxManifestSize: 10 << 20, Version: "test"})
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest("POST", "/api/setup", strings.NewReader(`{"password":"stranger-password"}`))
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden || IsSetup(srv.Store()) {
		t.Errorf("setup on a reachable server: got %d, want 403", w.Code)
	}
}

// Password checks refuse anything over maxPasswordLen, so a longer
// REMEMORY_ADMIN_PASSWORD could be stored but never used.
func TestEnvPasswordTooLongIsRefused(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := EnsureAdminPassword(store, "0.0.0.0", strings.Repeat("a", maxPasswordLen+1)); err == nil {
		t.Error("an unusable password was accepted")
	}
	if IsSetup(store) {
		t.Error("an unusable password was stored")
	}
}
