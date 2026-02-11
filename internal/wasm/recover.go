//go:build js && wasm

package main

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"fmt"
	"io"

	"github.com/eljojo/rememory/internal/core"
	"github.com/eljojo/rememory/internal/translations"
)

// ShareInfo contains parsed share metadata for JS interop.
// This wraps core.Share with base64-encoded data for transport to/from JS.
type ShareInfo struct {
	Version   int
	Index     int
	Total     int
	Threshold int
	Holder    string
	Created   string // RFC3339 formatted
	Checksum  string
	DataB64   string // Base64 encoded share data for transport
	Compact   string // Compact-encoded share string (e.g. RM1:2:5:3:BASE64:CHECK)
}

// ShareData is minimal data needed for combining.
type ShareData struct {
	Version int
	Index   int
	DataB64 string
}

// parseShare extracts a share from text content (which might be a full README.txt).
// Uses core.ParseShare for the actual parsing, then converts to ShareInfo for JS.
func parseShare(content string) (*ShareInfo, error) {
	share, err := core.ParseShare([]byte(content))
	if err != nil {
		return nil, err
	}

	// Verify checksum (core.ParseShare doesn't do this automatically since
	// the Verify method exists separately, but we want to catch corruption early)
	if err := share.Verify(); err != nil {
		return nil, err
	}

	return shareToInfo(share), nil
}

// parseCompactShare parses a compact-encoded share string.
func parseCompactShare(compact string) (*ShareInfo, error) {
	share, err := core.ParseCompact(compact)
	if err != nil {
		return nil, err
	}

	return shareToInfo(share), nil
}

// shareToInfo converts a core.Share to a ShareInfo for JS interop.
func shareToInfo(share *core.Share) *ShareInfo {
	return &ShareInfo{
		Version:   share.Version,
		Index:     share.Index,
		Total:     share.Total,
		Threshold: share.Threshold,
		Holder:    share.Holder,
		Created:   share.Created.Format("2006-01-02T15:04:05Z07:00"),
		Checksum:  share.Checksum,
		DataB64:   base64.StdEncoding.EncodeToString(share.Data),
		Compact:   share.CompactEncode(),
	}
}

// combineShares combines multiple shares to recover the passphrase.
// Uses core.Combine for the actual combination.
func combineShares(shares []ShareData) (string, error) {
	if len(shares) < 2 {
		return "", fmt.Errorf("need at least 2 shares, got %d", len(shares))
	}

	// Validate all shares have the same version
	for i := 1; i < len(shares); i++ {
		if shares[i].Version != shares[0].Version {
			return "", fmt.Errorf("share %d has different version (v%d vs v%d) — all shares must be from the same bundle", i+1, shares[i].Version, shares[0].Version)
		}
	}

	// Convert to raw bytes for core.Combine
	rawShares := make([][]byte, len(shares))
	for i, s := range shares {
		data, err := base64.StdEncoding.DecodeString(s.DataB64)
		if err != nil {
			return "", fmt.Errorf("decoding share %d: %w", i+1, err)
		}
		rawShares[i] = data
	}

	// Use core.Combine
	secret, err := core.Combine(rawShares)
	if err != nil {
		return "", fmt.Errorf("combining shares: %w", err)
	}

	return core.RecoverPassphrase(secret, shares[0].Version), nil
}

// decryptManifest decrypts age-encrypted data using a passphrase.
// Uses core.DecryptBytes for the actual decryption.
func decryptManifest(encryptedData []byte, passphrase string) ([]byte, error) {
	return core.DecryptBytes(encryptedData, passphrase)
}

// extractTarGz extracts files from tar.gz data in memory.
// Uses core.ExtractTarGz for the actual extraction.
func extractTarGz(tarGzData []byte) ([]core.ExtractedFile, error) {
	return core.ExtractTarGz(tarGzData)
}

// decodeShareWords converts 25 BIP39 words to raw share data bytes and share index.
// Auto-detects the word list language. The first 24 words encode the data;
// the 25th word packs 4 bits of index + 7 bits of checksum.
// Returns the decoded bytes, share index (0 if share >15), checksum, detected language, and any error.
func decodeShareWords(words []string) ([]byte, int, string, string, error) {
	data, index, lang, err := core.DecodeShareWordsAuto(words)
	if err != nil {
		return nil, 0, "", "", err
	}
	return data, index, core.HashBytes(data), string(lang), nil
}

// BundleContents represents extracted content from a bundle ZIP.
type BundleContents struct {
	Share    *ShareInfo // Parsed share from README.txt
	Manifest []byte     // Raw MANIFEST.age content
}

// extractBundle extracts share and manifest from a bundle ZIP file.
func extractBundle(zipData []byte) (*BundleContents, error) {
	r, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		return nil, fmt.Errorf("opening zip: %w", err)
	}

	var readmeContent string
	var manifestData []byte

	for _, f := range r.File {
		rc, err := f.Open()
		if err != nil {
			return nil, fmt.Errorf("opening %s: %w", f.Name, err)
		}

		data, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", f.Name, err)
		}

		switch {
		case translations.IsReadmeFile(f.Name, ".txt"):
			readmeContent = string(data)
		case f.Name == "MANIFEST.age":
			manifestData = data
		}
	}

	if readmeContent == "" {
		return nil, fmt.Errorf("README file not found in bundle")
	}

	// Parse share from README
	share, err := parseShare(readmeContent)
	if err != nil {
		return nil, fmt.Errorf("parsing share from README: %w", err)
	}

	return &BundleContents{
		Share:    share,
		Manifest: manifestData,
	}, nil
}
