package html

import (
	"crypto/rand"
	"encoding/base64"
	"strings"
)

// generateCSPNonce returns a base64-encoded 16-byte random nonce for CSP script-src.
func generateCSPNonce() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic("crypto/rand failed: " + err.Error())
	}
	return base64.StdEncoding.EncodeToString(b)
}

// applyCSPNonce generates a unique nonce and replaces all {{CSP_NONCE}} placeholders.
func applyCSPNonce(html string) string {
	nonce := generateCSPNonce()
	return strings.ReplaceAll(html, "{{CSP_NONCE}}", nonce)
}
