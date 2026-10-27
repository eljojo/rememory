package html

import (
	"crypto/rand"
	"regexp"
	"strings"
	"testing"
	"time"
)

func init() {
	SetVersion("test")
}

// staticPages returns all static HTML pages for testing.
func staticPages() map[string]string {
	wasmStub := []byte{0x00, 0x61, 0x73, 0x6d} // WASM magic bytes
	return map[string]string{
		"maker.html":   GenerateMakerHTML(wasmStub, MakerHTMLOptions{}),
		"recover.html": GenerateRecoverHTML(nil),
		"about.html":   GenerateIndexHTML(false),
		"docs.html":    GenerateDocsHTML("en", false),
	}
}

// TestStaticHTMLHasNoServerCode verifies that static HTML builds (non-selfhosted)
// contain zero traces of server-interaction code. This ensures offline bundles
// never contain code that could phone home.
func TestStaticHTMLHasNoServerCode(t *testing.T) {
	// These patterns must NOT appear in static output.
	// Note: SELFHOSTED_CONFIG = null is allowed (it's inert).
	// rememoryLoadManifest is now always present (unified JS) but is inert when
	// SELFHOSTED_CONFIG is null — the fetch only activates on manifestURL.
	forbidden := []string{
		"/api/bundle",
		"/api/setup",
		"maxManifestSize",
	}

	for name, content := range staticPages() {
		for _, pattern := range forbidden {
			if strings.Contains(content, pattern) {
				t.Errorf("static %s contains server code: found %q", name, pattern)
			}
		}
	}
}

// TestSelfhostedHTMLHasServerCode verifies that selfhosted HTML builds contain
// the server integration code that static builds must not have.
func TestSelfhostedHTMLHasServerCode(t *testing.T) {
	wasmStub := []byte{0x00, 0x61, 0x73, 0x6d}
	content := GenerateMakerHTML(wasmStub, MakerHTMLOptions{
		Selfhosted:       true,
		SelfhostedConfig: &SelfhostedConfig{MaxManifestSize: 50 << 20},
	})

	required := []string{
		"/api/bundle",
		"maxManifestSize",
	}

	for _, pattern := range required {
		if !strings.Contains(content, pattern) {
			t.Errorf("selfhosted maker.html missing server code: %q not found", pattern)
		}
	}
}

// TestStaticHostedRecoverHTML verifies that static-hosted recover.html
// contains the right config for auto-fetching from a relative MANIFEST.age URL.
func TestStaticHostedRecoverHTML(t *testing.T) {
	content := GenerateRecoverHTML(nil, RecoverHTMLOptions{
		StaticHosted: true,
	})

	// Must contain manifestURL pointing to ./MANIFEST.age
	if !strings.Contains(content, `"manifestURL":"./MANIFEST.age"`) {
		t.Error("static-hosted recover.html missing manifestURL config")
	}

	// Must not contain server API endpoints
	if strings.Contains(content, "/api/bundle") {
		t.Error("static-hosted recover.html contains /api/bundle")
	}

	// Nav links should use filenames, not server routes
	if strings.Contains(content, `href="/create"`) {
		t.Error("static-hosted recover.html has server nav rewrites")
	}
	if !strings.Contains(content, `href="about.html"`) {
		t.Error("static-hosted recover.html missing standard nav links")
	}
}

// TestStaticHTMLNoUnexpectedURLs scans static HTML output for every http:// and
// https:// URL and verifies it matches an allowed prefix. This catches accidental
// network calls that could phone home from what should be offline-capable bundles.
//
// The drand URLs are expected (tlock time-lock encryption needs the drand beacon).
// Project URLs (GitHub, GitHub Pages) are documentation links, not network calls.
// Everything else should be investigated before being added here.
func TestStaticHTMLNoUnexpectedURLs(t *testing.T) {
	allowed := []string{
		// drand beacon endpoints (tlock)
		"https://api.drand.sh",
		"https://api2.drand.sh",
		"https://api3.drand.sh",
		"https://drand.cloudflare.com",
		"https://docs.drand.love",

		// project URLs
		"https://github.com/eljojo/rememory",
		"https://eljojo.github.io/rememory",
		"https://eljojo.net", // author homepage

		// docs: linked in user-facing documentation and index.html
		"https://github.com/FiloSottile/age", // age encryption library
		"https://www.youtube.com",            // index.html "Why I built this" documentary
		"https://www.cloudflare.com",         // docs: League of Entropy (tlock section)
		"https://cryptomator.org",            // docs: recommended encrypted vault tool
		"https://veracrypt.fr",               // docs: recommended encrypted vault tool

		// vendored JS: comments in bundled noble-curves/age-encryption/tlock-js code
		"https://github.com/golang/go/issues", // wasm_exec.js workaround comment
		"https://github.com/paulmillr/noble",  // noble-secp256k1 library reference
		"https://eprint.iacr.org",             // cryptography research papers
		"https://ethresear.ch",                // BLS signature verification paper
		"https://www.rfc-editor.org",          // RFC errata references
		"https://datatracker.ietf.org",        // RFC 9380 hash-to-curve
		"https://crypto.stackexchange.com",    // elliptic curve Q&A
		"https://feross.org",                  // ieee754 library license attribution
		"https://hyperelliptic.org",           // EFD curve operation formulas
		"https://developer.mozilla.org",       // Web Crypto API JSDoc references

		// index.html: Shamir's Secret Sharing article (per-language translations)
		"https://en.wikipedia.org",
		"https://es.wikipedia.org",
		"https://de.wikipedia.org",
		"https://fr.wikipedia.org",
		"https://pt.wikipedia.org",
		"https://zh.wikipedia.org",
	}

	urlRe := regexp.MustCompile(`https?://[^\s"'<>,;)\\]+`)

	for name, content := range staticPages() {
		for _, url := range urlRe.FindAllString(content, -1) {
			if hasAllowedPrefix(url, allowed) {
				continue
			}
			t.Errorf("%s: unexpected URL %q", name, url)
		}
	}
}

func hasAllowedPrefix(url string, allowed []string) bool {
	for _, prefix := range allowed {
		if strings.HasPrefix(url, prefix) {
			return true
		}
	}
	return false
}

// extractCSP pulls the CSP connect-src value from an HTML document.
func extractCSP(html string) string {
	cspRe := regexp.MustCompile(`content="[^"]*connect-src\s+([^;"]*)[;"]`)
	m := cspRe.FindStringSubmatch(html)
	if len(m) > 1 {
		return m[1]
	}
	return ""
}

// ============================================================
// Network-posture tests
//
// These tests enforce the security-critical invariant that each
// HTML variant makes only the HTTP calls its network posture
// allows. They prevent regressions where network code leaks
// into offline bundles.
// ============================================================

// TestMakerCSPNoDrandEndpoints verifies that maker.html's CSP does not include
// drand HTTP endpoints. Tlock encryption in maker.html is offline — it uses the
// embedded chain config via createOfflineClient(), never HTTP.
func TestMakerCSPNoDrandEndpoints(t *testing.T) {
	wasmStub := []byte{0x00, 0x61, 0x73, 0x6d}
	content := GenerateMakerHTML(wasmStub, MakerHTMLOptions{})

	csp := extractCSP(content)
	if strings.Contains(csp, "api.drand.sh") {
		t.Errorf("maker.html CSP allows drand endpoints (encryption is offline): %s", csp)
	}
	if !strings.Contains(csp, "blob:") {
		t.Error("maker.html CSP missing blob: (needed for WASM loading)")
	}
}

// TestMakerAlwaysIncludesTlock verifies that maker.html always includes the
// tlock UI and drand config, regardless of options. Tlock encryption is always
// available because it's offline — no --no-timelock flag applies to it.
func TestMakerAlwaysIncludesTlock(t *testing.T) {
	wasmStub := []byte{0x00, 0x61, 0x73, 0x6d}
	content := GenerateMakerHTML(wasmStub, MakerHTMLOptions{})

	if !strings.Contains(content, "DRAND_CONFIG") {
		t.Error("maker.html should always include DRAND_CONFIG (offline encryption needs chain params)")
	}
	if !strings.Contains(content, "advanced-options") {
		t.Error("maker.html should always include tlock advanced options UI")
	}
	if !strings.Contains(content, "timelock-panel") {
		t.Error("maker.html should always include tlock panel HTML")
	}
}

// TestRecoverHTMLUsesCorrectAppVariant verifies that the correct app.js variant
// is selected based on tlock requirements:
//   - Generic (no personalization): app-tlock.js (handles any manifest)
//   - Personalized non-tlock: app.js (offline, smaller)
//   - Personalized tlock: app-tlock.js (needs HTTP for decryption)
//   - --no-timelock flag: forces app.js
func TestRecoverHTMLUsesCorrectAppVariant(t *testing.T) {
	// Generic recover.html (no personalization, no --no-timelock) should include tlock
	generic := GenerateRecoverHTML(nil)
	if !strings.Contains(generic, "DRAND_CONFIG") {
		t.Error("generic recover.html should include DRAND_CONFIG (tlock variant)")
	}

	// No-tlock recover.html should not include tlock code
	noTlock := GenerateRecoverHTML(nil, RecoverHTMLOptions{NoTlock: true})
	if strings.Contains(noTlock, "DRAND_CONFIG") {
		t.Error("no-tlock recover.html should not include DRAND_CONFIG")
	}

	// Personalized non-tlock bundle should not include tlock
	nonTlockPers := GenerateRecoverHTML(&PersonalizationData{
		Holder:      "Alice",
		HolderShare: "test-share",
		Threshold:   2,
		Total:       3,
	})
	if strings.Contains(nonTlockPers, "DRAND_CONFIG") {
		t.Error("personalized non-tlock recover.html should not include DRAND_CONFIG")
	}

	// Personalized tlock bundle should include tlock
	tlockPers := GenerateRecoverHTML(&PersonalizationData{
		Holder:       "Alice",
		HolderShare:  "test-share",
		Threshold:    2,
		Total:        3,
		TlockEnabled: true,
	})
	if !strings.Contains(tlockPers, "DRAND_CONFIG") {
		t.Error("personalized tlock recover.html should include DRAND_CONFIG")
	}
}

// TestRecoverCSPMatchesVariant verifies that the CSP connect-src correctly
// reflects the network posture: drand endpoints only when tlock is included,
// blob: only for offline variants.
func TestRecoverCSPMatchesVariant(t *testing.T) {
	// Generic (tlock) should allow drand endpoints
	generic := GenerateRecoverHTML(nil)
	genericCSP := extractCSP(generic)
	if !strings.Contains(genericCSP, "api.drand.sh") {
		t.Errorf("generic recover.html CSP should allow drand endpoints: %s", genericCSP)
	}

	// No-tlock should only have blob:
	noTlock := GenerateRecoverHTML(nil, RecoverHTMLOptions{NoTlock: true})
	noTlockCSP := extractCSP(noTlock)
	if strings.Contains(noTlockCSP, "api.drand.sh") {
		t.Errorf("no-tlock recover.html CSP should not allow drand endpoints: %s", noTlockCSP)
	}

	// Selfhosted should include 'self' for manifest fetch
	selfhosted := GenerateRecoverHTML(nil, RecoverHTMLOptions{
		Selfhosted:       true,
		SelfhostedConfig: &SelfhostedConfig{MaxManifestSize: 50 << 20},
	})
	selfhostedCSP := extractCSP(selfhosted)
	if !strings.Contains(selfhostedCSP, "'self'") {
		t.Errorf("selfhosted recover.html CSP should include 'self': %s", selfhostedCSP)
	}
}

// TestNonTlockRecoverHasNoTlockWaitingUI verifies that the offline
// recover.html variant doesn't include the tlock waiting UI element.
// Note: we check for the HTML element, not the bare string — app.js may
// contain getElementById('tlock-waiting') references that are inert when
// the element is absent from the DOM.
func TestNonTlockRecoverHasNoTlockWaitingUI(t *testing.T) {
	content := GenerateRecoverHTML(nil, RecoverHTMLOptions{NoTlock: true})

	if strings.Contains(content, `id="tlock-waiting"`) {
		t.Error("no-tlock recover.html should not contain tlock-waiting UI element")
	}
}

// TestTlockRecoverHasTlockWaitingUI verifies that the tlock variant
// includes the waiting UI for time-locked bundles.
func TestTlockRecoverHasTlockWaitingUI(t *testing.T) {
	content := GenerateRecoverHTML(nil)

	if !strings.Contains(content, `id="tlock-waiting"`) {
		t.Error("generic recover.html should contain tlock-waiting UI element")
	}
}

// TestTlockEnabledBundlePanicsWithNoTlock verifies that passing both NoTlock
// and TlockEnabled panics — this combination is a programming error.
func TestTlockEnabledBundlePanicsWithNoTlock(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic when NoTlock and TlockEnabled are both set")
		}
	}()

	GenerateRecoverHTML(&PersonalizationData{
		Holder:       "Alice",
		HolderShare:  "test-share",
		Threshold:    2,
		Total:        3,
		TlockEnabled: true,
	}, RecoverHTMLOptions{NoTlock: true})
}

// rememory serve renders maker.html per request; gzip -9 over the ~14 MB
// create.wasm on every view cost about a second of CPU each time.
func TestMakerHTMLEncodesWASMOnce(t *testing.T) {
	wasm := make([]byte, 8<<20)
	if _, err := rand.Read(wasm); err != nil {
		t.Fatal(err)
	}

	start := time.Now()
	first := GenerateMakerHTML(wasm, MakerHTMLOptions{})
	cold := time.Since(start)

	start = time.Now()
	second := GenerateMakerHTML(wasm, MakerHTMLOptions{})
	warm := time.Since(start)

	if len(first) != len(second) {
		t.Fatalf("pages differ: %d vs %d bytes", len(first), len(second))
	}
	if warm > cold/4 {
		t.Errorf("second render took %v after a first of %v; the WASM was encoded again", warm, cold)
	}
}
