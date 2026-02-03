//go:build js && wasm

package main

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"time"

	"filippo.io/age"
	vault "github.com/hashicorp/vault/shamir"
)

// ShareInfo contains parsed share metadata.
type ShareInfo struct {
	Version   int
	Index     int
	Total     int
	Threshold int
	Holder    string
	Created   time.Time
	Checksum  string
	DataB64   string // Base64 encoded share data for transport
}

// ShareData is minimal data needed for combining.
type ShareData struct {
	Index   int
	DataB64 string
}

// ExtractedFile represents a file extracted from tar.gz.
type ExtractedFile struct {
	Name string
	Data []byte
}

const (
	shareBegin = "-----BEGIN REMEMORY SHARE-----"
	shareEnd   = "-----END REMEMORY SHARE-----"

	// MaxFileSize is the maximum size of a single file during extraction (100 MB).
	MaxFileSize = 100 * 1024 * 1024
	// MaxTotalSize is the maximum total size of all extracted files (1 GB).
	MaxTotalSize = 1024 * 1024 * 1024
)

// parseShare extracts a share from text content (which might be a full README.txt).
func parseShare(content string) (*ShareInfo, error) {
	// Find the PEM block
	beginIdx := strings.Index(content, shareBegin)
	endIdx := strings.Index(content, shareEnd)
	if beginIdx == -1 || endIdx == -1 || endIdx <= beginIdx {
		return nil, fmt.Errorf("no share found in content")
	}

	// Extract content between markers
	inner := content[beginIdx+len(shareBegin) : endIdx]
	lines := strings.Split(strings.TrimSpace(inner), "\n")

	share := &ShareInfo{}
	var dataLines []string
	inData := false

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			inData = true
			continue
		}

		if inData {
			dataLines = append(dataLines, line)
			continue
		}

		// Parse header fields
		parts := strings.SplitN(line, ": ", 2)
		if len(parts) != 2 {
			continue
		}

		key, value := parts[0], parts[1]
		switch key {
		case "Version":
			v, err := strconv.Atoi(value)
			if err != nil {
				return nil, fmt.Errorf("invalid version: %w", err)
			}
			share.Version = v
		case "Index":
			v, err := strconv.Atoi(value)
			if err != nil {
				return nil, fmt.Errorf("invalid index: %w", err)
			}
			share.Index = v
		case "Total":
			v, err := strconv.Atoi(value)
			if err != nil {
				return nil, fmt.Errorf("invalid total: %w", err)
			}
			share.Total = v
		case "Threshold":
			v, err := strconv.Atoi(value)
			if err != nil {
				return nil, fmt.Errorf("invalid threshold: %w", err)
			}
			share.Threshold = v
		case "Holder":
			share.Holder = value
		case "Created":
			t, err := time.Parse(time.RFC3339, value)
			if err != nil {
				return nil, fmt.Errorf("invalid created time: %w", err)
			}
			share.Created = t
		case "Checksum":
			share.Checksum = value
		}
	}

	// Join data lines
	dataStr := strings.Join(dataLines, "")

	// Validate it's valid base64
	if _, err := base64.StdEncoding.DecodeString(dataStr); err != nil {
		return nil, fmt.Errorf("invalid base64 data: %w", err)
	}
	share.DataB64 = dataStr

	// Validate required fields
	if share.Version == 0 {
		return nil, fmt.Errorf("missing version")
	}
	if share.Index == 0 {
		return nil, fmt.Errorf("missing index")
	}
	if share.Total == 0 {
		return nil, fmt.Errorf("missing total")
	}
	if share.Threshold == 0 {
		return nil, fmt.Errorf("missing threshold")
	}
	if share.DataB64 == "" {
		return nil, fmt.Errorf("missing share data")
	}

	return share, nil
}

// combineShares combines multiple shares to recover the passphrase.
func combineShares(shares []ShareData) (string, error) {
	if len(shares) < 2 {
		return "", fmt.Errorf("need at least 2 shares, got %d", len(shares))
	}

	// Convert to raw bytes for vault shamir
	rawShares := make([][]byte, len(shares))
	for i, s := range shares {
		data, err := base64.StdEncoding.DecodeString(s.DataB64)
		if err != nil {
			return "", fmt.Errorf("decoding share %d: %w", i+1, err)
		}
		rawShares[i] = data
	}

	// Combine using HashiCorp Vault's Shamir implementation
	secret, err := vault.Combine(rawShares)
	if err != nil {
		return "", fmt.Errorf("combining shares: %w", err)
	}

	return string(secret), nil
}

// decryptManifest decrypts age-encrypted data using a passphrase.
func decryptManifest(encryptedData []byte, passphrase string) ([]byte, error) {
	identity, err := age.NewScryptIdentity(passphrase)
	if err != nil {
		return nil, fmt.Errorf("creating identity: %w", err)
	}

	reader, err := age.Decrypt(bytes.NewReader(encryptedData), identity)
	if err != nil {
		return nil, fmt.Errorf("decrypting: %w", err)
	}

	decrypted, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("reading decrypted data: %w", err)
	}

	return decrypted, nil
}

// extractTarGz extracts files from tar.gz data in memory.
func extractTarGz(tarGzData []byte) ([]ExtractedFile, error) {
	gzr, err := gzip.NewReader(bytes.NewReader(tarGzData))
	if err != nil {
		return nil, fmt.Errorf("creating gzip reader: %w", err)
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	var files []ExtractedFile
	var totalSize int64

	// Regex to detect path traversal
	pathTraversal := regexp.MustCompile(`(^|/)\.\.(/|$)`)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("reading tar: %w", err)
		}

		// Security: reject path traversal
		if pathTraversal.MatchString(header.Name) {
			return nil, fmt.Errorf("archive contains invalid path")
		}

		// Skip directories, only extract regular files
		if header.Typeflag != tar.TypeReg {
			continue
		}

		// Security: enforce file size limits
		if header.Size > MaxFileSize {
			return nil, fmt.Errorf("file exceeds maximum allowed size")
		}
		totalSize += header.Size
		if totalSize > MaxTotalSize {
			return nil, fmt.Errorf("archive exceeds maximum total size")
		}

		// Use LimitReader for additional safety
		limitedReader := io.LimitReader(tr, MaxFileSize)
		data, err := io.ReadAll(limitedReader)
		if err != nil {
			return nil, fmt.Errorf("failed to read file from archive")
		}

		files = append(files, ExtractedFile{
			Name: header.Name,
			Data: data,
		})
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("empty archive")
	}

	return files, nil
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

		switch f.Name {
		case "README.txt":
			readmeContent = string(data)
		case "MANIFEST.age":
			manifestData = data
		}
	}

	if readmeContent == "" {
		return nil, fmt.Errorf("README.txt not found in bundle")
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
