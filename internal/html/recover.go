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
const MaxEmbeddedManifestSize = 5 << 20 // 5 MiB

// PersonalizationData holds the data to personalize recover.html for a specific friend.
type PersonalizationData struct {
	Holder       string       `json:"holder"`                // This friend's name
	HolderShare  string       `json:"holderShare"`           // This friend's encoded share
	OtherFriends []FriendInfo `json:"otherFriends"`          // List of other friends
	Threshold    int          `json:"threshold"`             // Required shares (K)
	Total        int          `json:"total"`                 // Total shares (N)
	Language     string       `json:"language,omitempty"`    // Default UI language for this friend
	ManifestB64  string       `json:"manifestB64,omitempty"` // Base64-encoded MANIFEST.age (when <= MaxEmbeddedManifestSize)
}

// GenerateRecoverHTML creates the complete recover.html with all assets embedded.
// wasmBytes should be the compiled recover.wasm binary.
// version is the rememory version string.
// githubURL is the URL to download CLI binaries.
// personalization can be nil for a generic recover.html, or provided to personalize for a specific friend.
func GenerateRecoverHTML(wasmBytes []byte, version, githubURL string, personalization *PersonalizationData) string {
	html := recoverHTMLTemplate

	// Embed translations
	html = strings.Replace(html, "{{TRANSLATIONS}}", translations.GetTranslationsJS("recover"), 1)

	// Embed language picker (generated from translations.LangNames)
	html = strings.Replace(html, "{{LANG_OPTIONS}}", translations.LangSelectOptions(), 1)
	html = strings.Replace(html, "{{LANG_DETECT}}", translations.LangDetectJS(), 1)

	// Embed styles
	html = strings.Replace(html, "{{STYLES}}", stylesCSS, 1)

	// Embed wasm_exec.js
	html = strings.Replace(html, "{{WASM_EXEC}}", wasmExecJS, 1)

	// Embed shared.js + app.js
	html = strings.Replace(html, "{{APP_JS}}", sharedJS+"\n"+appJS, 1)

	// Embed WASM as gzip-compressed base64 (reduces size by ~70%)
	wasmB64 := compressAndEncode(wasmBytes)
	html = strings.Replace(html, "{{WASM_BASE64}}", wasmB64, 1)

	// Replace version and GitHub URL
	html = strings.Replace(html, "{{VERSION}}", version, 1)
	html = strings.Replace(html, "{{GITHUB_URL}}", githubURL, 1)

	// Embed personalization data as JSON (or null if not provided)
	var personalizationJSON string
	if personalization != nil {
		data, _ := json.Marshal(personalization)
		personalizationJSON = string(data)
	} else {
		personalizationJSON = "null"
	}
	html = strings.Replace(html, "{{PERSONALIZATION_DATA}}", personalizationJSON, 1)

	// Apply CSP nonce to all script tags
	html = applyCSPNonce(html)

	return html
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
