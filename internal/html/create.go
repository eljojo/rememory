package html

import (
	"encoding/base64"
	"strings"
)

// GenerateRememoryHTML creates the complete rememory.html with all assets embedded.
// wasmBytes should be the compiled recover.wasm binary.
// version is the rememory version string.
// githubURL is the URL to download CLI binaries.
func GenerateRememoryHTML(wasmBytes []byte, version, githubURL string) string {
	html := rememoryHTMLTemplate

	// Embed styles
	html = strings.Replace(html, "{{STYLES}}", stylesCSS, 1)

	// Embed wasm_exec.js
	html = strings.Replace(html, "{{WASM_EXEC}}", wasmExecJS, 1)

	// Embed create-app.js
	html = strings.Replace(html, "{{CREATE_APP_JS}}", createAppJS, 1)

	// Embed WASM as base64
	wasmB64 := base64.StdEncoding.EncodeToString(wasmBytes)
	html = strings.Replace(html, "{{WASM_BASE64}}", wasmB64, 1)

	// Replace version and GitHub URL
	html = strings.Replace(html, "{{VERSION}}", version, -1)
	html = strings.Replace(html, "{{GITHUB_URL}}", githubURL, -1)

	return html
}
