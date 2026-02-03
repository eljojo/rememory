// Package core provides shared cryptographic and data handling functions
// that work in both CLI and WASM environments.
package core

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
)

// HashString returns the SHA-256 hash of a string, prefixed with "sha256:".
func HashString(s string) string {
	return HashBytes([]byte(s))
}

// HashBytes returns the SHA-256 hash of bytes, prefixed with "sha256:".
func HashBytes(b []byte) string {
	h := sha256.Sum256(b)
	return "sha256:" + hex.EncodeToString(h[:])
}

// VerifyHash checks if the given hash matches the expected value.
// Uses constant-time comparison to prevent timing attacks.
func VerifyHash(got, expected string) bool {
	return subtle.ConstantTimeCompare([]byte(got), []byte(expected)) == 1
}
