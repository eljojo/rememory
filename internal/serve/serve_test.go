package serve

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// agePrefix is the standard age file header. Test manifests must start with
// this so the upload handler's format validation accepts them.
const agePrefix = "age-encryption.org/v1\n"

func newTestServer(t *testing.T) *Server {
	t.Helper()
	dir := t.TempDir()
	srv, err := New(Config{
		DataDir:         dir,
		MaxManifestSize: 10 << 20, // 10 MB
		Version:         "test",
	})
	if err != nil {
		t.Fatalf("creating test server: %v", err)
	}
	return srv
}

func setupPassword(t *testing.T, srv *Server, password string) {
	t.Helper()
	body := `{"password":"` + password + `"}`
	req := httptest.NewRequest("POST", "/api/setup", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("setup failed: %d %s", w.Code, w.Body.String())
	}
}

func uploadManifest(t *testing.T, srv *Server, manifest []byte, meta map[string]any) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)

	fw, err := mw.CreateFormFile("manifest", "MANIFEST.age")
	if err != nil {
		t.Fatal(err)
	}
	fw.Write(manifest)

	if meta != nil {
		metaJSON, _ := json.Marshal(meta)
		mw.WriteField("meta", string(metaJSON))
	}
	mw.Close()

	req := httptest.NewRequest("POST", "/api/bundle", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	return w
}

func TestSetupFlow(t *testing.T) {
	srv := newTestServer(t)

	// Before setup, status should show no password
	req := httptest.NewRequest("GET", "/api/status", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	var status map[string]any
	json.NewDecoder(w.Body).Decode(&status)
	if status["hasPassword"] != false {
		t.Error("expected hasPassword=false before setup")
	}

	// Setup password
	setupPassword(t, srv, "testpassword123")

	// Status should now show password set
	req = httptest.NewRequest("GET", "/api/status", nil)
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	json.NewDecoder(w.Body).Decode(&status)
	if status["hasPassword"] != true {
		t.Error("expected hasPassword=true after setup")
	}

	// Second setup should return 409
	body := `{"password":"another"}`
	req = httptest.NewRequest("POST", "/api/setup", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	if w.Code != http.StatusConflict {
		t.Errorf("expected 409 for second setup, got %d", w.Code)
	}
}

func TestBundleLifecycle(t *testing.T) {
	srv := newTestServer(t)
	setupPassword(t, srv, "adminpass")

	// Upload a manifest
	manifest := []byte(agePrefix + "fake-encrypted-manifest-data")
	meta := map[string]any{
		"name":      "test-recovery",
		"threshold": 2,
		"total":     3,
	}
	w := uploadManifest(t, srv, manifest, meta)
	if w.Code != http.StatusOK {
		t.Fatalf("upload failed: %d %s", w.Code, w.Body.String())
	}

	var result map[string]any
	json.NewDecoder(w.Body).Decode(&result)
	if result["id"] == nil || result["id"] == "" {
		t.Error("expected non-empty bundle ID")
	}

	// Status should show manifest exists
	req := httptest.NewRequest("GET", "/api/status", nil)
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	var status map[string]any
	json.NewDecoder(w.Body).Decode(&status)
	if status["hasManifest"] != true {
		t.Error("expected hasManifest=true after upload")
	}

	bundleID := result["id"].(string)

	// Download manifest by ID
	req = httptest.NewRequest("GET", "/api/bundle/manifest?id="+bundleID, nil)
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for manifest download, got %d", w.Code)
	}
	if !bytes.Equal(w.Body.Bytes(), manifest) {
		t.Error("downloaded manifest doesn't match uploaded")
	}

	// Delete without ID should fail
	body := `{"password":"adminpass"}`
	req = httptest.NewRequest("DELETE", "/api/bundle", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing ID, got %d", w.Code)
	}

	// Delete with wrong password
	body = `{"id":"` + bundleID + `","password":"wrong"}`
	req = httptest.NewRequest("DELETE", "/api/bundle", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for wrong password, got %d", w.Code)
	}

	// Delete with correct password and ID
	body = `{"id":"` + bundleID + `","password":"adminpass"}`
	req = httptest.NewRequest("DELETE", "/api/bundle", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for delete, got %d %s", w.Code, w.Body.String())
	}

	// Status should show no manifest
	req = httptest.NewRequest("GET", "/api/status", nil)
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	json.NewDecoder(w.Body).Decode(&status)
	if status["hasManifest"] != false {
		t.Error("expected hasManifest=false after delete")
	}
}

func TestSetupRequired(t *testing.T) {
	srv := newTestServer(t)

	// Upload should fail without setup
	manifest := []byte("test")
	w := uploadManifest(t, srv, manifest, nil)
	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403 without setup, got %d", w.Code)
	}
}

func TestLargeManifest(t *testing.T) {
	dir := t.TempDir()
	srv, err := New(Config{
		DataDir:         dir,
		MaxManifestSize: 1 << 10, // 1 KB for testing
		Version:         "test",
	})
	if err != nil {
		t.Fatal(err)
	}
	setupPassword(t, srv, "pass12345678")

	// Upload manifest larger than limit
	bigManifest := make([]byte, 2<<10) // 2 KB
	w := uploadManifest(t, srv, bigManifest, nil)
	if w.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("expected 413 for large manifest, got %d", w.Code)
	}

	// Upload manifest within limit (must have valid age header)
	smallManifest := []byte(agePrefix + "small-test-data")
	w = uploadManifest(t, srv, smallManifest, nil)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for small manifest, got %d %s", w.Code, w.Body.String())
	}
}

func TestUploadRejectsInvalidFormat(t *testing.T) {
	srv := newTestServer(t)
	setupPassword(t, srv, "formatpass")

	// Random data that isn't age or tlock should be rejected
	w := uploadManifest(t, srv, []byte("not-a-valid-manifest"), nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid format, got %d", w.Code)
	}

	// Empty data should be rejected
	w = uploadManifest(t, srv, []byte{}, nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty upload, got %d", w.Code)
	}
}

func TestPathTraversalPrevention(t *testing.T) {
	s := &Store{dir: t.TempDir()}
	// Attempt to delete with path traversal ID
	err := s.Delete("../../etc/passwd")
	if err == nil {
		t.Error("expected error for path traversal ID")
	}
}

func TestDeletePathTraversal(t *testing.T) {
	srv := newTestServer(t)
	setupPassword(t, srv, "deletetraversal")

	for _, malicious := range []string{
		"../../etc/passwd",
		"../admin.age",
		"not-a-uuid",
	} {
		body := `{"id":"` + malicious + `","password":"deletetraversal"}`
		req := httptest.NewRequest("DELETE", "/api/bundle", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		srv.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400 for malicious delete id %q, got %d", malicious, w.Code)
		}
	}
}

func TestManifestPathTraversal(t *testing.T) {
	srv := newTestServer(t)
	setupPassword(t, srv, "traversalpass")

	// Upload a valid bundle so the server is in a working state
	uploadManifest(t, srv, []byte(agePrefix+"real-manifest"), map[string]any{"name": "legit"})

	// Attempt to fetch manifest with path traversal ID
	for _, malicious := range []string{
		"../../etc/passwd",
		"../admin.age",
		"..%2F..%2Fetc%2Fpasswd",
		"not-a-uuid",
	} {
		req := httptest.NewRequest("GET", "/api/bundle/manifest?id="+malicious, nil)
		w := httptest.NewRecorder()
		srv.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400 for malicious id %q, got %d", malicious, w.Code)
		}
	}
}

func TestConcurrentAccess(t *testing.T) {
	srv := newTestServer(t)
	setupPassword(t, srv, "concurrentpass")

	var wg sync.WaitGroup
	errors := make(chan error, 10)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			manifest := []byte(agePrefix + strings.Repeat("x", 100))
			w := uploadManifest(t, srv, manifest, map[string]any{
				"name": "concurrent-test",
			})
			if w.Code != http.StatusOK {
				errors <- io.EOF
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		if err != nil {
			t.Errorf("concurrent upload failed: %v", err)
		}
	}

	// Verify at least one bundle exists
	if !srv.store.HasManifest() {
		t.Error("expected at least one manifest after concurrent uploads")
	}
}

func TestRootPage(t *testing.T) {
	srv := newTestServer(t)

	// Before setup: should show setup page (200, not redirect)
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for setup page, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "Set up ReMemory") {
		t.Error("expected setup page content")
	}

	// After setup, no manifest: should show home page with empty state
	setupPassword(t, srv, "redirectpass1")
	req = httptest.NewRequest("GET", "/", nil)
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for home page, got %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "About ReMemory") {
		t.Error("expected home page intro")
	}
	if !strings.Contains(body, "var BUNDLES = []") {
		t.Error("expected empty bundles array in home page")
	}

	// After upload: should show home page with bundle data
	manifest := []byte(agePrefix + "test-manifest")
	uploadManifest(t, srv, manifest, map[string]any{"name": "my-bundle", "threshold": 2, "total": 3})
	req = httptest.NewRequest("GET", "/", nil)
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for home page with bundle, got %d", w.Code)
	}
	body = w.Body.String()
	if !strings.Contains(body, `"total":3`) {
		t.Error("expected bundle metadata in home page after upload")
	}
}

func TestStoreList(t *testing.T) {
	srv := newTestServer(t)
	setupPassword(t, srv, "listpass12345")

	// List should return empty initially
	metas, err := srv.store.List()
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(metas) != 0 {
		t.Errorf("expected 0 bundles, got %d", len(metas))
	}

	// Upload two bundles with a pause so timestamps differ
	uploadManifest(t, srv, []byte(agePrefix+"manifest-1"), map[string]any{"name": "first"})
	time.Sleep(1100 * time.Millisecond)
	uploadManifest(t, srv, []byte(agePrefix+"manifest-2"), map[string]any{"name": "second"})

	metas, err = srv.store.List()
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(metas) != 2 {
		t.Errorf("expected 2 bundles, got %d", len(metas))
	}
	// Newest first
	if metas[0].Name != "second" {
		t.Errorf("expected newest bundle first, got %q", metas[0].Name)
	}
}

func TestAPIListBundles(t *testing.T) {
	srv := newTestServer(t)
	setupPassword(t, srv, "listpass12345")

	// Empty list
	req := httptest.NewRequest("GET", "/api/bundles", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var bundles []BundleMeta
	json.NewDecoder(w.Body).Decode(&bundles)
	if len(bundles) != 0 {
		t.Errorf("expected 0 bundles, got %d", len(bundles))
	}

	// Upload and list again
	uploadManifest(t, srv, []byte(agePrefix+"manifest-data"), map[string]any{"name": "test-bundle", "threshold": 2, "total": 3})

	req = httptest.NewRequest("GET", "/api/bundles", nil)
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	json.NewDecoder(w.Body).Decode(&bundles)
	if len(bundles) != 1 {
		t.Errorf("expected 1 bundle, got %d", len(bundles))
	}
	if bundles[0].Name != "test-bundle" {
		t.Errorf("expected name 'test-bundle', got %q", bundles[0].Name)
	}
}

func TestManifestByID(t *testing.T) {
	srv := newTestServer(t)
	setupPassword(t, srv, "manifestpass1")

	// Upload two bundles with a pause so timestamps differ
	w1 := uploadManifest(t, srv, []byte(agePrefix+"manifest-A"), map[string]any{"name": "first"})
	var r1 map[string]any
	json.NewDecoder(w1.Body).Decode(&r1)
	id1 := r1["id"].(string)

	time.Sleep(1100 * time.Millisecond)

	w2 := uploadManifest(t, srv, []byte(agePrefix+"manifest-B"), map[string]any{"name": "second"})
	var r2 map[string]any
	json.NewDecoder(w2.Body).Decode(&r2)
	id2 := r2["id"].(string)

	// Fetch by specific ID
	req := httptest.NewRequest("GET", "/api/bundle/manifest?id="+id1, nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if !bytes.Equal(w.Body.Bytes(), []byte(agePrefix+"manifest-A")) {
		t.Error("expected manifest-A for first ID")
	}

	// Fetch second by ID
	req = httptest.NewRequest("GET", "/api/bundle/manifest?id="+id2, nil)
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	if !bytes.Equal(w.Body.Bytes(), []byte(agePrefix+"manifest-B")) {
		t.Error("expected manifest-B for second ID")
	}

	// Fetch without ID returns 400
	req = httptest.NewRequest("GET", "/api/bundle/manifest", nil)
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing ID, got %d", w.Code)
	}
}

func TestEmptyPasswordRejected(t *testing.T) {
	srv := newTestServer(t)

	body := `{"password":""}`
	req := httptest.NewRequest("POST", "/api/setup", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty password, got %d", w.Code)
	}
}

// Any page the owner visits could otherwise post to a server on their
// network, e.g. claim the admin password of a fresh instance on localhost.
func TestCrossSiteRequestsRefused(t *testing.T) {
	srv := newTestServer(t)

	req := httptest.NewRequest("POST", "/api/setup", strings.NewReader(`{"password":"attacker-pw"}`))
	req.Header.Set("Content-Type", "text/plain")
	req.Header.Set("Origin", "https://evil.example")
	req.Header.Set("Sec-Fetch-Site", "cross-site")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Errorf("cross-site setup: got %d, want 403", w.Code)
	}
	if IsSetup(srv.Store()) {
		t.Fatal("cross-site request set the admin password")
	}

	// Same-origin browser requests and scripts without browser headers work.
	req = httptest.NewRequest("POST", "/api/setup", strings.NewReader(`{"password":"owner-password"}`))
	req.Header.Set("Origin", "http://example.com")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("same-origin setup: got %d %s", w.Code, w.Body.String())
	}
}
