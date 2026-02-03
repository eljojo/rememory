package manifest

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const (
	// MaxFileSize is the maximum size of a single file during extraction (100 MB).
	MaxFileSize = 100 * 1024 * 1024
	// MaxTotalSize is the maximum total size of all extracted files (1 GB).
	MaxTotalSize = 1024 * 1024 * 1024
)

// Archive creates a tar.gz archive of the given directory.
// The archive preserves the directory structure relative to the source.
func Archive(w io.Writer, sourceDir string) error {
	sourceDir, err := filepath.Abs(sourceDir)
	if err != nil {
		return fmt.Errorf("resolving path: %w", err)
	}

	info, err := os.Stat(sourceDir)
	if err != nil {
		return fmt.Errorf("accessing directory: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("not a directory: %s", sourceDir)
	}

	gzw := gzip.NewWriter(w)
	defer gzw.Close()

	tw := tar.NewWriter(gzw)
	defer tw.Close()

	err = filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Create tar header
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return fmt.Errorf("creating header for %s: %w", path, err)
		}

		// Preserve directory structure relative to source's parent
		relPath, err := filepath.Rel(filepath.Dir(sourceDir), path)
		if err != nil {
			return fmt.Errorf("computing relative path: %w", err)
		}
		header.Name = relPath

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
		return fmt.Errorf("walking directory: %w", err)
	}

	return nil
}

// Extract unpacks a tar.gz archive to the destination directory.
// Returns the path to the extracted directory (the root folder from the archive).
func Extract(r io.Reader, destDir string) (string, error) {
	destDir, err := filepath.Abs(destDir)
	if err != nil {
		return "", fmt.Errorf("resolving path: %w", err)
	}

	if err := os.MkdirAll(destDir, 0755); err != nil {
		return "", fmt.Errorf("creating destination: %w", err)
	}

	gzr, err := gzip.NewReader(r)
	if err != nil {
		return "", fmt.Errorf("creating gzip reader: %w", err)
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
			return "", fmt.Errorf("reading tar: %w", err)
		}

		// Track the root directory
		parts := strings.Split(header.Name, string(filepath.Separator))
		if len(parts) > 0 && rootDir == "" {
			rootDir = parts[0]
		}

		target := filepath.Join(destDir, header.Name)

		// Security: prevent path traversal
		if !strings.HasPrefix(filepath.Clean(target)+string(filepath.Separator), filepath.Clean(destDir)+string(filepath.Separator)) {
			return "", fmt.Errorf("invalid path in archive: %s", header.Name)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, os.FileMode(header.Mode)); err != nil {
				return "", fmt.Errorf("creating directory %s: %w", target, err)
			}

		case tar.TypeReg:
			// Security: enforce file size limit
			if header.Size > MaxFileSize {
				return "", fmt.Errorf("file exceeds maximum size of %d bytes", MaxFileSize)
			}
			totalSize += header.Size
			if totalSize > MaxTotalSize {
				return "", fmt.Errorf("archive exceeds maximum total size of %d bytes", MaxTotalSize)
			}

			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return "", fmt.Errorf("creating parent directory: %w", err)
			}

			f, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return "", fmt.Errorf("creating file %s: %w", target, err)
			}

			// Use LimitReader to enforce size limit during actual copy
			limitedReader := io.LimitReader(tr, MaxFileSize+1)
			written, err := io.Copy(f, limitedReader)
			f.Close()
			if err != nil {
				return "", fmt.Errorf("writing file %s: %w", target, err)
			}
			if written > MaxFileSize {
				return "", fmt.Errorf("file exceeds maximum size during extraction")
			}

		default:
			// Skip other types (symlinks, etc.) for security
		}
	}

	if rootDir == "" {
		return "", fmt.Errorf("empty archive")
	}

	return filepath.Join(destDir, rootDir), nil
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
