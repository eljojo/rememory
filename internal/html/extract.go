package html

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"regexp"
)

// personalizationManifest is a minimal struct for extracting just the manifest
// from the PERSONALIZATION JSON embedded in recover.html.
type personalizationManifest struct {
	ManifestB64 string `json:"manifestB64"`
}

// personalizationRe matches the PERSONALIZATION JSON in recover.html.
// The JSON is single-line (produced by json.Marshal) and appears as:
//
//	window.PERSONALIZATION = {...};
var personalizationRe = regexp.MustCompile(`window\.PERSONALIZATION\s*=\s*(\{[^\n]*\})\s*;`)

// ExtractManifestFromHTML extracts the MANIFEST.age bytes from a personalized
// recover.html file. It finds the embedded PERSONALIZATION JSON, parses the
// manifestB64 field, and base64-decodes it.
//
// Returns an error if the HTML doesn't contain personalization data, or if
// the personalization doesn't include an embedded manifest (e.g., when
// --no-embed-manifest was used or the manifest was too large).
func ExtractManifestFromHTML(htmlContent []byte) ([]byte, error) {
	matches := personalizationRe.FindSubmatch(htmlContent)
	if len(matches) < 2 {
		return nil, fmt.Errorf("no PERSONALIZATION data found in HTML")
	}

	var p personalizationManifest
	if err := json.Unmarshal(matches[1], &p); err != nil {
		return nil, fmt.Errorf("parsing PERSONALIZATION JSON: %w", err)
	}

	if p.ManifestB64 == "" {
		return nil, fmt.Errorf("no embedded manifest in HTML (manifestB64 is empty)")
	}

	data, err := base64.StdEncoding.DecodeString(p.ManifestB64)
	if err != nil {
		return nil, fmt.Errorf("decoding manifest base64: %w", err)
	}

	return data, nil
}
