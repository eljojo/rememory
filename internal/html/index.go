package html

import (
	"strings"
)

// GenerateIndexHTML creates the landing page HTML with embedded CSS.
// version is the rememory version string.
// githubURL is the URL to download CLI binaries.
func GenerateIndexHTML(version, githubURL string) string {
	html := indexHTMLTemplate

	// Embed styles
	html = strings.Replace(html, "{{STYLES}}", stylesCSS, 1)

	// Embed dataflow animation
	html = strings.Replace(html, "{{DATAFLOW_JS}}", dataflowJS, 1)

	// Replace version and GitHub URL
	html = strings.Replace(html, "{{VERSION}}", version, -1)
	html = strings.Replace(html, "{{GITHUB_URL}}", githubURL, -1)

	return html
}
