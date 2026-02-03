package core

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"regexp"
)

const (
	// MaxFileSize is the maximum size of a single file during extraction (100 MB).
	MaxFileSize = 100 * 1024 * 1024
	// MaxTotalSize is the maximum total size of all extracted files (1 GB).
	MaxTotalSize = 1024 * 1024 * 1024
)

// ExtractedFile represents a file extracted from a tar.gz archive.
type ExtractedFile struct {
	Name string
	Data []byte
}

// ExtractTarGz extracts files from tar.gz data in memory.
// This is used by both CLI and WASM for in-memory extraction.
// For file-based extraction, use the manifest package.
func ExtractTarGz(tarGzData []byte) ([]ExtractedFile, error) {
	return ExtractTarGzReader(bytes.NewReader(tarGzData))
}

// ExtractTarGzReader extracts files from a tar.gz reader.
func ExtractTarGzReader(r io.Reader) ([]ExtractedFile, error) {
	gzr, err := gzip.NewReader(r)
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
			return nil, fmt.Errorf("archive contains invalid path: %s", header.Name)
		}

		// Skip directories, symlinks, and other special files
		if header.Typeflag != tar.TypeReg {
			continue
		}

		// Security: enforce file size limits
		if header.Size > MaxFileSize {
			return nil, fmt.Errorf("file %s exceeds maximum allowed size (%d bytes)", header.Name, MaxFileSize)
		}
		totalSize += header.Size
		if totalSize > MaxTotalSize {
			return nil, fmt.Errorf("archive exceeds maximum total size (%d bytes)", MaxTotalSize)
		}

		// Use LimitReader for additional safety
		limitedReader := io.LimitReader(tr, MaxFileSize)
		data, err := io.ReadAll(limitedReader)
		if err != nil {
			return nil, fmt.Errorf("reading file %s from archive: %w", header.Name, err)
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
