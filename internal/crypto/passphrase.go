package crypto

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
)

const (
	// DefaultPassphraseBytes is the number of random bytes for passphrase generation.
	// 32 bytes = 256 bits of entropy, encoded as ~43 base64 characters.
	DefaultPassphraseBytes = 32
)

// GeneratePassphrase creates a cryptographically secure passphrase.
// The passphrase is URL-safe base64 encoded (no padding) for easy handling.
func GeneratePassphrase(numBytes int) (string, error) {
	_, passphrase, err := GenerateRawPassphrase(numBytes)
	return passphrase, err
}

// GenerateRawPassphrase creates random bytes and returns both the raw bytes
// and the base64url-encoded passphrase string. Protocol v2 splits the raw bytes
// via Shamir (instead of the encoded string), then base64url-encodes after
// recombining. The passphrase string is used for age encryption.
func GenerateRawPassphrase(numBytes int) (raw []byte, passphrase string, err error) {
	if numBytes < 16 {
		return nil, "", fmt.Errorf("passphrase must be at least 16 bytes, got %d", numBytes)
	}

	raw = make([]byte, numBytes)
	if _, err := rand.Read(raw); err != nil {
		return nil, "", fmt.Errorf("generating random bytes: %w", err)
	}

	// URL-safe base64 without padding for easy copy-paste
	passphrase = base64.RawURLEncoding.EncodeToString(raw)
	return raw, passphrase, nil
}
