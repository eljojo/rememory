package core

import (
	"archive/tar"
	"archive/zip"
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

// CheckArchiveLimits reports a file larger than MaxFileSize, or a running
// total (including this file) larger than MaxTotalSize. Producers (seal,
// the maker) check before archiving and extractors before writing, so
// anything ReMemory archives can be extracted again.
func CheckArchiveLimits(name string, size, total int64) error {
	if size > MaxFileSize {
		return fmt.Errorf("%s is %s; each file can be at most %s", name, formatLimitSize(size), formatLimitSize(MaxFileSize))
	}
	if total > MaxTotalSize {
		return fmt.Errorf("the files add up to more than %s, the most that can be recovered (reached at %s)", formatLimitSize(MaxTotalSize), name)
	}
	return nil
}

func formatLimitSize(n int64) string {
	unit, div := "MB", float64(1<<20)
	if n >= 1<<30 {
		unit, div = "GB", float64(1<<30)
	}
	return fmt.Sprintf("%.1f %s", float64(n)/div, unit)
}

// ExtractedFile represents a file extracted from an archive.
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

// ExtractZip extracts files from ZIP data in memory.
// Same security checks (path traversal, size limits) as ExtractTarGz.
func ExtractZip(zipData []byte) ([]ExtractedFile, error) {
	r, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		return nil, fmt.Errorf("opening zip: %w", err)
	}

	var files []ExtractedFile
	var totalSize int64

	// Regex to detect path traversal
	pathTraversal := regexp.MustCompile(`(^|/)\.\.(/|$)`)

	for _, f := range r.File {
		// Security: reject path traversal
		if pathTraversal.MatchString(f.Name) {
			return nil, fmt.Errorf("archive contains invalid path: %s", f.Name)
		}

		// Skip directories
		if f.FileInfo().IsDir() {
			continue
		}

		// Security: enforce file size limits
		if int64(f.UncompressedSize64) > MaxFileSize {
			return nil, fmt.Errorf("file %s exceeds maximum allowed size (%d bytes)", f.Name, MaxFileSize)
		}
		totalSize += int64(f.UncompressedSize64)
		if totalSize > MaxTotalSize {
			return nil, fmt.Errorf("archive exceeds maximum total size (%d bytes)", MaxTotalSize)
		}

		rc, err := f.Open()
		if err != nil {
			return nil, fmt.Errorf("opening zip entry %s: %w", f.Name, err)
		}

		limitedReader := io.LimitReader(rc, MaxFileSize+1)
		data, err := io.ReadAll(limitedReader)
		rc.Close()
		if err != nil {
			return nil, fmt.Errorf("reading file %s from archive: %w", f.Name, err)
		}
		if int64(len(data)) > MaxFileSize {
			return nil, fmt.Errorf("file %s exceeds maximum size during extraction", f.Name)
		}

		files = append(files, ExtractedFile{
			Name: f.Name,
			Data: data,
		})
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("empty archive")
	}

	return files, nil
}

// ExtractArchive detects the archive format and extracts accordingly.
// ZIP archives start with PK\x03\x04, gzip with \x1f\x8b.
func ExtractArchive(data []byte) ([]ExtractedFile, error) {
	if len(data) < 2 {
		return nil, fmt.Errorf("archive too small to detect format")
	}

	// ZIP: starts with PK\x03\x04
	if data[0] == 0x50 && data[1] == 0x4B {
		return ExtractZip(data)
	}

	// gzip: starts with \x1f\x8b
	if data[0] == 0x1f && data[1] == 0x8b {
		return ExtractTarGz(data)
	}

	return nil, fmt.Errorf("unrecognized archive format (expected ZIP or tar.gz)")
}
