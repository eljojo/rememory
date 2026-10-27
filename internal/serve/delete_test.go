package serve

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A save interrupted after its metadata was written leaves a bundle that
// is listed but has no MANIFEST.age. The admin must still be able to
// delete it.
func TestDeleteBundleWithoutManifest(t *testing.T) {
	srv := newTestServer(t)
	setupPassword(t, srv, "owner-password")

	id := "00000000-0000-4000-8000-000000000001"
	dir := filepath.Join(srv.Store().dir, "bundles", id)
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "meta.json"), []byte(`{"id":"`+id+`"}`), 0600); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest("DELETE", "/api/bundle", strings.NewReader(`{"id":"`+id+`","password":"owner-password"}`))
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("got %d %s, want 200", w.Code, w.Body.String())
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Error("bundle directory is still there")
	}
}
