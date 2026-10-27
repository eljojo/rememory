package html

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"strings"

	"github.com/eljojo/rememory/internal/translations"
)

// FriendInfo holds friend contact information for the UI.
type FriendInfo struct {
	Name       string `json:"name"`
	Contact    string `json:"contact,omitempty"`
	ShareIndex int    `json:"shareIndex"` // 1-based share index for this friend
}

// MaxEmbeddedManifestSize is the maximum size of MANIFEST.age that will be
// embedded (base64-encoded) in recover.html. Manifests at or below this size
// are included so recovery can work without the separate MANIFEST.age file.
const MaxEmbeddedManifestSize = 10 << 20 // 10 MiB

// PersonalizationData holds the data to personalize recover.html for a specific friend.
type PersonalizationData struct {
	Holder       string       `json:"holder"`                 // This friend's name
	HolderShare  string       `json:"holderShare"`            // This friend's encoded share
	OtherFriends []FriendInfo `json:"otherFriends"`           // List of other friends
	Threshold    int          `json:"threshold"`              // Required shares (K)
	Total        int          `json:"total"`                  // Total shares (N)
	Language     string       `json:"language,omitempty"`     // Default UI language for this friend
	ManifestB64  string       `json:"manifestB64,omitempty"`  // Base64-encoded MANIFEST.age (when <= MaxEmbeddedManifestSize)
	TlockEnabled bool         `json:"tlockEnabled,omitempty"` // Signals tlock-js should be included
}

// tlockWaitingHTML is the time-lock waiting UI injected into recover.html.
// The CSS lives in assets/tlock-waiting.css (embedded as tlockWaitingCSS)
// and is injected as a <style> block alongside this HTML when tlock is enabled.
const tlockWaitingHTML = `<div id="tlock-waiting" class="tlock-waiting hidden" aria-live="polite">
        <div class="tlock-waiting-icon">&#128336;</div>
        <div class="tlock-waiting-body">
          <strong id="tlock-waiting-title" data-i18n="tlock_waiting_title">Not yet</strong>
          <p id="tlock-waiting-date"></p>
          <p class="tlock-waiting-hint"><a href="{{GITHUB_PAGES}}/docs#timelock" target="_blank" data-i18n="tlock_learn_more">What is this?</a></p>
        </div>
      </div>`

// RecoverHTMLOptions holds optional parameters for GenerateRecoverHTML.
type RecoverHTMLOptions struct {
	NoTlock          bool              // Omit tlock-js even from generic recover.html
	Selfhosted       bool              // Full selfhosted server mode (nav rewrites + config)
	SelfhostedConfig *SelfhostedConfig // Config injected into the HTML for selfhosted mode
	StaticHosted     bool              // Static hosting mode (manifest fetch, no nav rewrites)
}

// GenerateRecoverHTML creates the complete recover.html with all assets embedded.
// Uses native JavaScript crypto (no WASM required).
// personalization can be nil for a generic recover.html, or provided to personalize for a specific friend.
func GenerateRecoverHTML(personalization *PersonalizationData, opts ...RecoverHTMLOptions) string {
	// Process content template
	content := recoverHTMLTemplate

	// Parse options
	var selfhosted, staticHosted bool
	var selfhostedConfig *SelfhostedConfig
	var noTlock bool
	if len(opts) > 0 {
		selfhosted = opts[0].Selfhosted
		staticHosted = opts[0].StaticHosted
		selfhostedConfig = opts[0].SelfhostedConfig
		noTlock = opts[0].NoTlock
	}

	// Static hosted mode: auto-create config pointing to ./MANIFEST.age
	if staticHosted && selfhostedConfig == nil {
		selfhostedConfig = &SelfhostedConfig{HasManifest: true, ManifestURL: "./MANIFEST.age"}
	}

	// Two-variant model for recover.html, organized by network posture:
	//
	//   app.js       (offline)  — zero HTTP calls, no tlock/drand code
	//   app-tlock.js (network)  — tlock decryption via HTTP to drand relays
	//
	// Which variant to use:
	//   - Generic/standalone (personalization == nil): use app-tlock.js so
	//     GitHub Pages and selfhosted can handle any manifest
	//   - Personalized tlock bundle (TlockEnabled): use app-tlock.js
	//   - Personalized non-tlock bundle: use app.js (smaller, offline)
	//   - --no-timelock flag: force app.js even for generic

	// NoTlock + TlockEnabled is a programming error — no valid code path should produce this.
	if noTlock && personalization != nil && personalization.TlockEnabled {
		panic("html: NoTlock and TlockEnabled are mutually exclusive — this is a programming error")
	}
	includeTlock := !noTlock && (personalization == nil || personalization.TlockEnabled)

	// Inject tlock waiting HTML (with CSS) or empty
	if includeTlock {
		content = strings.Replace(content, "{{TLOCK_WAITING_HTML}}",
			"<style>"+tlockWaitingCSS+"</style>\n"+tlockWaitingHTML, 1)
	} else {
		content = strings.Replace(content, "{{TLOCK_WAITING_HTML}}", "", 1)
	}

	// Build CSP connect-src
	var cspConnectSrc string
	if includeTlock {
		cspConnectSrc = drandCSPConnectSrc()
	} else {
		cspConnectSrc = "blob:"
	}
	if selfhosted || staticHosted {
		cspConnectSrc += " 'self'"
	}

	// CSP meta tag
	headMeta := `<meta name="generator" content="ReMemory {{VERSION}}">
  <meta http-equiv="Content-Security-Policy" content="default-src 'none'; script-src 'nonce-{{CSP_NONCE}}' 'wasm-unsafe-eval'; style-src 'unsafe-inline'; img-src blob: data:; connect-src ` + cspConnectSrc + `; form-action 'none';">`

	// Language selector for nav
	navExtras := `<select class="lang-select" id="lang-select" aria-label="Language">
        ` + translations.LangSelectOptions() + `
      </select>`

	// Embed selfhosted config (or null)
	var selfhostedConfigJSON string
	if selfhostedConfig != nil {
		configData, _ := json.Marshal(selfhostedConfig)
		selfhostedConfigJSON = string(configData)
	} else {
		selfhostedConfigJSON = "null"
	}

	// Embed personalization data as JSON (or null)
	var personalizationJSON string
	if personalization != nil {
		data, _ := json.Marshal(personalization)
		personalizationJSON = string(data)
	} else {
		personalizationJSON = "null"
	}

	// Embed README basenames for ZIP extraction
	readmeNames, _ := json.Marshal(translations.ReadmeBasenames())

	// Select the appropriate app JS variant
	selectedAppJS := appJS
	if includeTlock {
		selectedAppJS = appTlockJS
	}

	// Build all scripts
	var scripts strings.Builder

	// Translations
	// Translations (docs link rewriting + rememoryUpdateUI are handled by core i18n.js)
	scripts.WriteString(i18nScript(I18nScriptOptions{
		Component:           "recover",
		UseNonce:            true,
		ExtraDeclarations:   `const docsLangs = ` + DocsLanguagesJS() + `;`,
		DOMContentLoadedPre: i18nRecoverInitJS,
	}))

	// Personalization data
	scripts.WriteString(`

  <!-- Personalization data (embedded for this specific friend) -->
  <script nonce="{{CSP_NONCE}}">
    window.PERSONALIZATION = ` + personalizationJSON + `;
    window.README_NAMES = ` + string(readmeNames) + `;
    window.SELFHOSTED_CONFIG = ` + selfhostedConfigJSON + `;
  </script>`)

	// Tlock config and drand
	if includeTlock {
		scripts.WriteString("\n\n  <!-- Time-lock decryption (conditionally included when tlock is needed) -->\n  " + drandConfigScript())
	}

	// Application logic
	scripts.WriteString("\n\n  <!-- Application logic (native JavaScript crypto, no WASM) -->\n  <script nonce=\"{{CSP_NONCE}}\">" + sharedJS + "\n" + selectedAppJS + "</script>")

	// Bundle nav: shown when personalized, replaces the default nav
	bundleNavHTML := `<div class="nav-links hidden" id="nav-links-bundle">
        <a href="{{GITHUB_PAGES}}/docs" target="_blank" data-i18n="nav_guide">Guide</a>
      </div>`

	result := applyLayout(LayoutOptions{
		Title:      "ReMemory Recovery Tool",
		HeadMeta:   headMeta,
		Selfhosted: selfhosted,
		NavExtras: bundleNavHTML + `
      ` + navExtras,
		BeforeContainer: `<!-- Toast notifications container -->
  <div id="toast-container" class="toast-container" role="alert" aria-live="polite"></div>

  <!-- QR Scanner modal -->
  <div id="qr-scanner-modal" class="qr-scanner-modal hidden" role="dialog" aria-modal="true" aria-label="QR code scanner">
    <div class="qr-scanner-header">
      <span data-i18n="scan_title">Scan a QR code</span>
      <button id="qr-scanner-close" class="qr-scanner-close" aria-label="Close">&times;</button>
    </div>
    <div class="qr-scanner-body">
      <video id="qr-video" autoplay playsinline></video>
      <div class="qr-scanner-overlay"></div>
    </div>
    <div class="qr-scanner-hint">
      <span data-i18n="scan_hint">Point your camera at the QR code on a friend's printed copy</span>
    </div>
  </div>`,
		Content: content,
		FooterContent: `<p>ReMemory {{VERSION}}</p>
    <p>
      <span data-i18n="need_help">Need help?</span>
      <a href="{{GITHUB_PAGES}}/docs#recovering" target="_blank">Docs</a> &#xB7;
      <a href="{{GITHUB_URL}}" target="_blank" data-i18n="download_cli">ReMemory on GitHub</a>
    </p>`,
		Scripts: scripts.String(),
	})

	// Apply CSP nonce to all script tags
	result = applyCSPNonce(result)

	return result
}

// compressAndEncode gzip-compresses data and returns base64-encoded result.
// This reduces WASM size by ~70% in the embedded HTML.
func compressAndEncode(data []byte) string {
	var buf bytes.Buffer
	gz, err := gzip.NewWriterLevel(&buf, gzip.BestCompression)
	if err != nil {
		panic("gzip.NewWriterLevel: " + err.Error())
	}
	if _, err := gz.Write(data); err != nil {
		panic("gzip.Write: " + err.Error())
	}
	if err := gz.Close(); err != nil {
		panic("gzip.Close: " + err.Error())
	}
	return base64.StdEncoding.EncodeToString(buf.Bytes())
}
