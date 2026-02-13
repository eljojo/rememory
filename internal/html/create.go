package html

import (
	"strings"

	"github.com/eljojo/rememory/internal/translations"
)

// GenerateMakerHTML creates the complete maker.html with all assets embedded.
// createWASMBytes is the create.wasm binary (runs in browser for bundle creation).
// Note: create.wasm self-contains recover.wasm embedded within it (via html.GetRecoverWASMBytes()).
// version is the rememory version string.
// githubURL is the URL to download CLI binaries.
func GenerateMakerHTML(createWASMBytes []byte, version, githubURL string) string {
	html := makerHTMLTemplate

	// Embed translations
	html = strings.Replace(html, "{{TRANSLATIONS}}", translations.GetTranslationsJS("maker"), 1)

	// Embed language picker (generated from translations.LangNames)
	html = strings.Replace(html, "{{LANG_OPTIONS}}", translations.LangSelectOptions(), 1)
	html = strings.Replace(html, "{{LANG_DETECT}}", translations.LangDetectJS(), 1)

	// Embed styles
	html = strings.Replace(html, "{{STYLES}}", stylesCSS, 1)

	// Embed wasm_exec.js
	html = strings.Replace(html, "{{WASM_EXEC}}", wasmExecJS, 1)

	// Embed shared.js + create-app.js
	html = strings.Replace(html, "{{CREATE_APP_JS}}", sharedJS+"\n"+createAppJS, 1)

	// Embed create.wasm as gzip-compressed base64 (this runs in the browser)
	// Note: create.wasm contains recover.wasm embedded within it for generating bundles
	createWASMB64 := compressAndEncode(createWASMBytes)
	html = strings.Replace(html, "{{WASM_BASE64}}", createWASMB64, 1)

	// Replace version and GitHub URL
	html = strings.Replace(html, "{{VERSION}}", version, -1)
	html = strings.Replace(html, "{{GITHUB_URL}}", githubURL, -1)

	// Apply CSP nonce to all script tags
	html = applyCSPNonce(html)

	return html
}
