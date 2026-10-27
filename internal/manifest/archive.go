package manifest

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/eljojo/rememory/internal/core"
)

// ArchiveResult contains the result of an archive operation.
type ArchiveResult struct {
	// Warnings contains messages about files that were skipped (symlinks, etc.)
	Warnings []string
}

// ArchiveTarGz creates a tar.gz archive of the given directory.
// The archive preserves the directory structure relative to the source.
// Returns warnings about any skipped files (symlinks, special files, etc.)
func ArchiveTarGz(w io.Writer, sourceDir string) (*ArchiveResult, error) {
	result := &ArchiveResult{}

	sourceDir, err := filepath.Abs(sourceDir)
	if err != nil {
		return nil, fmt.Errorf("resolving path: %w", err)
	}

	info, err := os.Stat(sourceDir)
	if err != nil {
		return nil, fmt.Errorf("accessing directory: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("not a directory: %s", sourceDir)
	}

	gzw := gzip.NewWriter(w)
	defer gzw.Close()

	tw := tar.NewWriter(gzw)
	defer tw.Close()

	err = filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Compute relative path for display
		relPath, err := filepath.Rel(filepath.Dir(sourceDir), path)
		if err != nil {
			return fmt.Errorf("computing relative path: %w", err)
		}

		// Check for symlinks and other special files
		mode := info.Mode()
		if mode&os.ModeSymlink != 0 {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("skipping symlink: %s (symlinks are not preserved for security)", relPath))
			return nil
		}
		if !mode.IsRegular() && !mode.IsDir() {
			typeName := describeFileType(mode)
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("skipping %s: %s (only regular files and directories are archived)", typeName, relPath))
			return nil
		}

		// Create tar header
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return fmt.Errorf("creating header for %s: %w", path, err)
		}

		header.Name = filepath.ToSlash(relPath)

		// Ensure directory entries end with /
		if info.IsDir() {
			header.Name += "/"
		}

		if err := tw.WriteHeader(header); err != nil {
			return fmt.Errorf("writing header for %s: %w", path, err)
		}

		// Only write content for regular files
		if !info.Mode().IsRegular() {
			return nil
		}

		f, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("opening %s: %w", path, err)
		}
		defer f.Close()

		if _, err := io.Copy(tw, f); err != nil {
			return fmt.Errorf("copying %s: %w", path, err)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("walking directory: %w", err)
	}

	return result, nil
}

// describeFileType returns a human-readable description of a file type.
func describeFileType(mode os.FileMode) string {
	switch {
	case mode&os.ModeSymlink != 0:
		return "symlink"
	case mode&os.ModeDevice != 0:
		return "device file"
	case mode&os.ModeNamedPipe != 0:
		return "named pipe"
	case mode&os.ModeSocket != 0:
		return "socket"
	case mode&os.ModeCharDevice != 0:
		return "character device"
	case mode&os.ModeIrregular != 0:
		return "irregular file"
	default:
		return "special file"
	}
}

// ExtractResult contains the result of an extract operation.
type ExtractResult struct {
	// Path is the path to the extracted directory (root folder from archive)
	Path string
	// Warnings contains messages about files that were skipped (symlinks, etc.)
	Warnings []string
}

// ExtractTarGz unpacks a tar.gz archive to the destination directory.
// Returns the path to the extracted directory and any warnings about skipped files.
func ExtractTarGz(r io.Reader, destDir string) (*ExtractResult, error) {
	result := &ExtractResult{}

	destDir, err := filepath.Abs(destDir)
	if err != nil {
		return nil, fmt.Errorf("resolving path: %w", err)
	}

	if err := os.MkdirAll(destDir, 0755); err != nil {
		return nil, fmt.Errorf("creating destination: %w", err)
	}

	gzr, err := gzip.NewReader(r)
	if err != nil {
		return nil, fmt.Errorf("creating gzip reader: %w", err)
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	var rootDir string
	var totalSize int64

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("reading tar: %w", err)
		}

		// Track the root directory
		parts := strings.Split(header.Name, "/")
		if len(parts) > 0 && rootDir == "" {
			rootDir = parts[0]
		}

		target := filepath.Join(destDir, header.Name)

		// Security: prevent path traversal
		if !strings.HasPrefix(filepath.Clean(target)+string(filepath.Separator), filepath.Clean(destDir)+string(filepath.Separator)) {
			return nil, fmt.Errorf("invalid path in archive: %s", header.Name)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, os.FileMode(header.Mode)&0777); err != nil {
				return nil, fmt.Errorf("creating directory %s: %w", target, err)
			}

		case tar.TypeReg:
			// Security: enforce file size limit
			totalSize += header.Size
			if err := core.CheckArchiveLimits(header.Name, header.Size, totalSize); err != nil {
				return nil, err
			}

			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return nil, fmt.Errorf("creating parent directory: %w", err)
			}

			f, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(header.Mode)&0666)
			if err != nil {
				return nil, fmt.Errorf("creating file %s: %w", target, err)
			}

			// Use LimitReader to enforce size limit during actual copy
			limitedReader := io.LimitReader(tr, core.MaxFileSize+1)
			written, err := io.Copy(f, limitedReader)
			closeErr := f.Close()
			if err != nil {
				return nil, fmt.Errorf("writing file %s: %w", target, err)
			}
			if closeErr != nil {
				return nil, fmt.Errorf("closing file %s: %w", target, closeErr)
			}
			if written > core.MaxFileSize {
				return nil, fmt.Errorf("file exceeds maximum size during extraction")
			}

		case tar.TypeSymlink:
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("skipping symlink in archive: %s (symlinks not extracted for security)", header.Name))

		case tar.TypeLink:
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("skipping hard link in archive: %s (hard links not extracted for security)", header.Name))

		default:
			typeName := describeTarType(header.Typeflag)
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("skipping %s in archive: %s (only regular files and directories are extracted)", typeName, header.Name))
		}
	}

	if rootDir == "" {
		return nil, fmt.Errorf("empty archive")
	}

	result.Path = filepath.Join(destDir, rootDir)
	return result, nil
}

// describeTarType returns a human-readable description of a tar entry type.
func describeTarType(typeflag byte) string {
	switch typeflag {
	case tar.TypeSymlink:
		return "symlink"
	case tar.TypeLink:
		return "hard link"
	case tar.TypeChar:
		return "character device"
	case tar.TypeBlock:
		return "block device"
	case tar.TypeFifo:
		return "named pipe (FIFO)"
	default:
		return "special file"
	}
}

// ArchiveZip creates a ZIP archive of the given directory.
// The archive preserves the directory structure relative to the source.
// Returns warnings about any skipped files (symlinks, special files, etc.)
func ArchiveZip(w io.Writer, sourceDir string) (*ArchiveResult, error) {
	result := &ArchiveResult{}

	sourceDir, err := filepath.Abs(sourceDir)
	if err != nil {
		return nil, fmt.Errorf("resolving path: %w", err)
	}

	info, err := os.Stat(sourceDir)
	if err != nil {
		return nil, fmt.Errorf("accessing directory: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("not a directory: %s", sourceDir)
	}

	zw := zip.NewWriter(w)
	defer zw.Close()

	err = filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Compute relative path for display
		relPath, err := filepath.Rel(filepath.Dir(sourceDir), path)
		if err != nil {
			return fmt.Errorf("computing relative path: %w", err)
		}

		// Check for symlinks and other special files
		mode := info.Mode()
		if mode&os.ModeSymlink != 0 {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("skipping symlink: %s (symlinks are not preserved for security)", relPath))
			return nil
		}
		if !mode.IsRegular() && !mode.IsDir() {
			typeName := describeFileType(mode)
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("skipping %s: %s (only regular files and directories are archived)", typeName, relPath))
			return nil
		}

		// Skip the root directory itself (ZIP doesn't need an explicit root entry)
		if path == sourceDir {
			return nil
		}

		// Ensure directory entries end with /
		name := filepath.ToSlash(relPath)
		if info.IsDir() {
			name += "/"
		}

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return fmt.Errorf("creating header for %s: %w", path, err)
		}
		header.Name = name
		if info.IsDir() {
			header.Method = zip.Store
		} else {
			header.Method = zip.Deflate
		}

		fw, err := zw.CreateHeader(header)
		if err != nil {
			return fmt.Errorf("writing header for %s: %w", path, err)
		}

		// Only write content for regular files
		if !info.Mode().IsRegular() {
			return nil
		}

		f, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("opening %s: %w", path, err)
		}
		defer f.Close()

		if _, err := io.Copy(fw, f); err != nil {
			return fmt.Errorf("copying %s: %w", path, err)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("walking directory: %w", err)
	}

	return result, nil
}

// ExtractZip unpacks a ZIP archive to the destination directory.
// Returns the path to the extracted directory and any warnings about skipped files.
func ExtractZip(r io.ReaderAt, size int64, destDir string) (*ExtractResult, error) {
	result := &ExtractResult{}

	destDir, err := filepath.Abs(destDir)
	if err != nil {
		return nil, fmt.Errorf("resolving path: %w", err)
	}

	zr, err := zip.NewReader(r, size)
	if err != nil {
		return nil, fmt.Errorf("opening zip: %w", err)
	}

	// Check every size before creating or writing anything, so a refused
	// archive leaves no partial folder behind.
	var totalSize int64
	for _, f := range zr.File {
		if f.FileInfo().IsDir() {
			continue
		}
		totalSize += int64(f.UncompressedSize64)
		if err := core.CheckArchiveLimits(f.Name, int64(f.UncompressedSize64), totalSize); err != nil {
			return nil, err
		}
	}

	if err := os.MkdirAll(destDir, 0755); err != nil {
		return nil, fmt.Errorf("creating destination: %w", err)
	}

	var rootDir string

	for _, f := range zr.File {
		// Track the root directory
		parts := strings.SplitN(f.Name, "/", 2)
		if len(parts) > 0 && rootDir == "" {
			rootDir = parts[0]
		}

		target := filepath.Join(destDir, f.Name)

		// Security: prevent path traversal
		if !strings.HasPrefix(filepath.Clean(target)+string(filepath.Separator), filepath.Clean(destDir)+string(filepath.Separator)) {
			return nil, fmt.Errorf("invalid path in archive: %s", f.Name)
		}

		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0755); err != nil {
				return nil, fmt.Errorf("creating directory %s: %w", target, err)
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return nil, fmt.Errorf("creating parent directory: %w", err)
		}

		outFile, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, f.Mode()&0666)
		if err != nil {
			return nil, fmt.Errorf("creating file %s: %w", target, err)
		}

		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return nil, fmt.Errorf("opening zip entry %s: %w", f.Name, err)
		}

		limitedReader := io.LimitReader(rc, core.MaxFileSize+1)
		written, err := io.Copy(outFile, limitedReader)
		rc.Close()
		closeErr := outFile.Close()
		if err != nil {
			return nil, fmt.Errorf("writing file %s: %w", target, err)
		}
		if closeErr != nil {
			return nil, fmt.Errorf("closing file %s: %w", target, closeErr)
		}
		if written > core.MaxFileSize {
			return nil, fmt.Errorf("file exceeds maximum size during extraction")
		}
	}

	if rootDir == "" {
		return nil, fmt.Errorf("empty archive")
	}

	result.Path = filepath.Join(destDir, rootDir)
	return result, nil
}

// ExtractAuto detects the archive format (ZIP or tar.gz) and extracts accordingly.
// It reads the first bytes to determine the format, then delegates to the appropriate extractor.
func ExtractAuto(r io.ReadSeeker, destDir string) (*ExtractResult, error) {
	// Read magic bytes to detect format
	magic := make([]byte, 2)
	n, err := r.Read(magic)
	if err != nil {
		return nil, fmt.Errorf("reading archive header: %w", err)
	}
	if n < 2 {
		return nil, fmt.Errorf("archive too small to detect format")
	}

	// Rewind to the beginning
	if _, err := r.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("rewinding reader: %w", err)
	}

	// ZIP: starts with PK\x03\x04
	if magic[0] == 0x50 && magic[1] == 0x4B {
		// Read all data so we can create a ReaderAt
		data, err := io.ReadAll(r)
		if err != nil {
			return nil, fmt.Errorf("reading zip data: %w", err)
		}
		reader := bytes.NewReader(data)
		return ExtractZip(reader, int64(len(data)), destDir)
	}

	// gzip: starts with \x1f\x8b
	if magic[0] == 0x1f && magic[1] == 0x8b {
		// tar streams, so read the headers once first: a refused archive
		// then leaves no partial folder behind.
		if err := checkTarGzSizes(r); err != nil {
			return nil, err
		}
		if _, err := r.Seek(0, io.SeekStart); err != nil {
			return nil, fmt.Errorf("rewinding reader: %w", err)
		}
		return ExtractTarGz(r, destDir)
	}

	return nil, fmt.Errorf("unrecognized archive format (expected ZIP or tar.gz)")
}

// checkTarGzSizes applies core.CheckArchiveLimits to the regular files in a
// tar.gz stream without writing anything.
func checkTarGzSizes(r io.Reader) error {
	gzr, err := gzip.NewReader(r)
	if err != nil {
		return fmt.Errorf("creating gzip reader: %w", err)
	}
	defer gzr.Close()
	tr := tar.NewReader(gzr)
	var total int64
	for {
		header, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return fmt.Errorf("reading tar: %w", err)
		}
		if header.Typeflag != tar.TypeReg {
			continue
		}
		total += header.Size
		if err := core.CheckArchiveLimits(header.Name, header.Size, total); err != nil {
			return err
		}
	}
}

// CheckSizes applies core.CheckArchiveLimits to the regular files under
// dir (the ones ArchiveZip includes), so an archive is refused before it is
// made rather than when someone tries to recover it.
func CheckSizes(dir string) error {
	var total int64
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			rel = path
		}
		total += info.Size()
		return core.CheckArchiveLimits(filepath.ToSlash(rel), info.Size(), total)
	})
}

// CountFiles counts the number of regular files in a directory.
func CountFiles(dir string) (int, error) {
	count := 0
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.Mode().IsRegular() {
			count++
		}
		return nil
	})
	return count, err
}

// DirSize calculates the total size of all files in a directory.
func DirSize(dir string) (int64, error) {
	var size int64
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.Mode().IsRegular() {
			size += info.Size()
		}
		return nil
	})
	return size, err
}
