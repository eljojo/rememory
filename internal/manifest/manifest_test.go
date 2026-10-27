package manifest

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/eljojo/rememory/internal/core"
)

func TestArchiveExtract(t *testing.T) {
	// Create a temp directory with test files
	srcDir := t.TempDir()
	testDir := filepath.Join(srcDir, "manifest")
	if err := os.MkdirAll(testDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create test files
	files := map[string]string{
		"README.md":       "# Test Manifest",
		"secret.txt":      "super secret data",
		"subdir/file.txt": "nested file content",
	}

	for path, content := range files {
		fullPath := filepath.Join(testDir, path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	// Archive
	var buf bytes.Buffer
	archiveResult, err := ArchiveTarGz(&buf, testDir)
	if err != nil {
		t.Fatalf("archive: %v", err)
	}
	if len(archiveResult.Warnings) > 0 {
		t.Logf("archive warnings: %v", archiveResult.Warnings)
	}

	// Extract to new location
	dstDir := t.TempDir()
	extractResult, err := ExtractTarGz(&buf, dstDir)
	if err != nil {
		t.Fatalf("extract: %v", err)
	}

	// Verify files
	for path, expectedContent := range files {
		fullPath := filepath.Join(extractResult.Path, path)
		content, err := os.ReadFile(fullPath)
		if err != nil {
			t.Errorf("reading %s: %v", path, err)
			continue
		}
		if string(content) != expectedContent {
			t.Errorf("%s: got %q, want %q", path, content, expectedContent)
		}
	}
}

func TestArchiveNotDirectory(t *testing.T) {
	// Create a temp file
	f, err := os.CreateTemp("", "test")
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	defer os.Remove(f.Name())

	var buf bytes.Buffer
	_, err = ArchiveTarGz(&buf, f.Name())
	if err == nil {
		t.Error("expected error for non-directory")
	}
}

// createTarGzBytes builds a tar.gz archive in memory with arbitrary entry names.
// This allows crafting malicious archives for security testing.
func createTarGzBytes(t *testing.T, entries map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	gzw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gzw)

	for name, content := range entries {
		if err := tw.WriteHeader(&tar.Header{
			Name:     name,
			Size:     int64(len(content)),
			Mode:     0644,
			Typeflag: tar.TypeReg,
		}); err != nil {
			t.Fatalf("writing tar header for %q: %v", name, err)
		}
		if _, err := tw.Write([]byte(content)); err != nil {
			t.Fatalf("writing tar content for %q: %v", name, err)
		}
	}

	// Close tar then gzip explicitly (not defer) to ensure full flush.
	if err := tw.Close(); err != nil {
		t.Fatalf("closing tar writer: %v", err)
	}
	if err := gzw.Close(); err != nil {
		t.Fatalf("closing gzip writer: %v", err)
	}
	return buf.Bytes()
}

func TestExtractPathTraversal(t *testing.T) {
	t.Run("rejected paths", func(t *testing.T) {
		tests := []struct {
			name  string
			entry string
		}{
			{"direct traversal", "../escape.txt"},
			{"relative traversal", "subdir/../../escape.txt"},
			{"deep traversal", "foo/bar/../../../etc/shadow"},
			{"bare dotdot", ".."},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				data := createTarGzBytes(t, map[string]string{tt.entry: "malicious"})
				destDir := t.TempDir()
				_, err := ExtractTarGz(bytes.NewReader(data), destDir)
				if err == nil {
					t.Errorf("expected error for path %q, got nil", tt.entry)
				}
				if err != nil && !strings.Contains(err.Error(), "invalid path") {
					t.Errorf("expected 'invalid path' error for %q, got: %v", tt.entry, err)
				}

				// Verify no files were written outside destDir.
				entries, _ := os.ReadDir(destDir)
				if len(entries) > 0 {
					t.Errorf("expected no files written for traversal path, found %d entries", len(entries))
				}
			})
		}
	})

	t.Run("accepted safe path", func(t *testing.T) {
		data := createTarGzBytes(t, map[string]string{
			"manifest/safe.txt": "safe content",
		})
		destDir := t.TempDir()
		_, err := ExtractTarGz(bytes.NewReader(data), destDir)
		if err != nil {
			t.Fatalf("unexpected error for safe path: %v", err)
		}

		got, err := os.ReadFile(filepath.Join(destDir, "manifest", "safe.txt"))
		if err != nil {
			t.Fatalf("reading extracted file: %v", err)
		}
		if string(got) != "safe content" {
			t.Errorf("got %q, want %q", got, "safe content")
		}
	})

	// filepath.Clean resolves "foo/../bar" to "bar" which stays within destDir,
	// so the HasPrefix check correctly allows it. This differs from the core
	// package's regex which rejects any path containing ".." — both behaviors
	// are correct for their context (file-based vs in-memory extraction).
	t.Run("non-escaping dotdot accepted", func(t *testing.T) {
		data := createTarGzBytes(t, map[string]string{
			"foo/../bar.txt": "resolved content",
		})
		destDir := t.TempDir()
		_, err := ExtractTarGz(bytes.NewReader(data), destDir)
		if err != nil {
			t.Fatalf("unexpected error for non-escaping dotdot: %v", err)
		}

		got, err := os.ReadFile(filepath.Join(destDir, "bar.txt"))
		if err != nil {
			t.Fatalf("reading extracted file: %v", err)
		}
		if string(got) != "resolved content" {
			t.Errorf("got %q, want %q", got, "resolved content")
		}
	})
}

func TestCountFiles(t *testing.T) {
	dir := t.TempDir()

	// Create some files
	os.WriteFile(filepath.Join(dir, "a.txt"), []byte("a"), 0644)
	os.WriteFile(filepath.Join(dir, "b.txt"), []byte("b"), 0644)
	os.MkdirAll(filepath.Join(dir, "sub"), 0755)
	os.WriteFile(filepath.Join(dir, "sub", "c.txt"), []byte("c"), 0644)

	count, err := CountFiles(dir)
	if err != nil {
		t.Fatal(err)
	}
	if count != 3 {
		t.Errorf("got %d files, want 3", count)
	}
}

func TestDirSize(t *testing.T) {
	dir := t.TempDir()

	// Create files with known sizes
	os.WriteFile(filepath.Join(dir, "a.txt"), []byte("12345"), 0644)
	os.WriteFile(filepath.Join(dir, "b.txt"), []byte("67890"), 0644)

	size, err := DirSize(dir)
	if err != nil {
		t.Fatal(err)
	}
	if size != 10 {
		t.Errorf("got size %d, want 10", size)
	}
}

func TestArchiveNonexistent(t *testing.T) {
	var buf bytes.Buffer
	_, err := ArchiveTarGz(&buf, "/nonexistent/path")
	if err == nil {
		t.Error("expected error for nonexistent directory")
	}
}

func TestExtractInvalidGzip(t *testing.T) {
	// Not valid gzip data
	data := bytes.NewReader([]byte("not gzip data"))
	_, err := ExtractTarGz(data, t.TempDir())
	if err == nil {
		t.Error("expected error for invalid gzip")
	}
}

func TestExtractEmptyArchive(t *testing.T) {
	// Create an empty gzip stream (valid gzip, but no tar entries)
	var buf bytes.Buffer
	gzw := gzip.NewWriter(&buf)
	gzw.Close()

	_, err := ExtractTarGz(&buf, t.TempDir())
	if err == nil {
		t.Error("expected error for empty archive")
	}
}

func TestCountFilesNonexistent(t *testing.T) {
	_, err := CountFiles("/nonexistent/path")
	if err == nil {
		t.Error("expected error for nonexistent directory")
	}
}

func TestDirSizeNonexistent(t *testing.T) {
	_, err := DirSize("/nonexistent/path")
	if err == nil {
		t.Error("expected error for nonexistent directory")
	}
}

func TestArchiveSymlinkWarning(t *testing.T) {
	// Create a temp directory with a symlink
	srcDir := t.TempDir()
	testDir := filepath.Join(srcDir, "manifest")
	if err := os.MkdirAll(testDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create a regular file
	regularFile := filepath.Join(testDir, "regular.txt")
	if err := os.WriteFile(regularFile, []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a symlink
	symlinkPath := filepath.Join(testDir, "link.txt")
	if err := os.Symlink(regularFile, symlinkPath); err != nil {
		t.Skip("symlinks not supported on this platform")
	}

	// Archive should succeed but warn about symlink
	var buf bytes.Buffer
	result, err := ArchiveTarGz(&buf, testDir)
	if err != nil {
		t.Fatalf("archive: %v", err)
	}

	// Should have a warning about the symlink
	if len(result.Warnings) == 0 {
		t.Error("expected warning about symlink, got none")
	}

	foundSymlinkWarning := false
	for _, w := range result.Warnings {
		if strings.Contains(w, "symlink") {
			foundSymlinkWarning = true
			break
		}
	}
	if !foundSymlinkWarning {
		t.Errorf("expected symlink warning, got: %v", result.Warnings)
	}

	// Extract and verify only regular file is present
	dstDir := t.TempDir()
	extractResult, err := ExtractTarGz(&buf, dstDir)
	if err != nil {
		t.Fatalf("extract: %v", err)
	}

	// Regular file should exist
	if _, err := os.Stat(filepath.Join(extractResult.Path, "regular.txt")); err != nil {
		t.Errorf("regular file should exist: %v", err)
	}

	// Symlink should NOT exist (was skipped)
	if _, err := os.Stat(filepath.Join(extractResult.Path, "link.txt")); err == nil {
		t.Error("symlink should not have been archived")
	}
}

func TestArchiveEmptyDir(t *testing.T) {
	dir := t.TempDir()
	emptyDir := filepath.Join(dir, "empty")
	os.MkdirAll(emptyDir, 0755)

	var buf bytes.Buffer
	_, err := ArchiveTarGz(&buf, emptyDir)
	if err != nil {
		t.Fatalf("Archive empty dir: %v", err)
	}

	// Should still be valid archive
	dstDir := t.TempDir()
	_, err = ExtractTarGz(&buf, dstDir)
	if err != nil {
		t.Fatalf("Extract empty archive: %v", err)
	}
}

func TestArchiveZipExtractZip(t *testing.T) {
	// Create a temp directory with test files
	srcDir := t.TempDir()
	testDir := filepath.Join(srcDir, "manifest")
	if err := os.MkdirAll(testDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create test files
	files := map[string]string{
		"README.md":       "# Test Manifest",
		"secret.txt":      "super secret data",
		"subdir/file.txt": "nested file content",
	}

	for path, content := range files {
		fullPath := filepath.Join(testDir, path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	// Archive as ZIP
	var buf bytes.Buffer
	archiveResult, err := ArchiveZip(&buf, testDir)
	if err != nil {
		t.Fatalf("archive zip: %v", err)
	}
	if len(archiveResult.Warnings) > 0 {
		t.Logf("archive warnings: %v", archiveResult.Warnings)
	}

	// Verify it's actually a ZIP (magic bytes)
	data := buf.Bytes()
	if len(data) < 2 || data[0] != 0x50 || data[1] != 0x4B {
		t.Fatal("archive does not start with ZIP magic bytes")
	}

	// Extract ZIP
	dstDir := t.TempDir()
	extractResult, err := ExtractZip(bytes.NewReader(data), int64(len(data)), dstDir)
	if err != nil {
		t.Fatalf("extract zip: %v", err)
	}

	// Verify files
	for path, expectedContent := range files {
		fullPath := filepath.Join(extractResult.Path, path)
		content, err := os.ReadFile(fullPath)
		if err != nil {
			t.Errorf("reading %s: %v", path, err)
			continue
		}
		if string(content) != expectedContent {
			t.Errorf("%s: got %q, want %q", path, content, expectedContent)
		}
	}
}

// createZipBytes builds a ZIP archive in memory with arbitrary entry names.
// This allows crafting malicious archives for security testing.
func createZipBytes(t *testing.T, entries map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	for name, content := range entries {
		fw, err := zw.Create(name)
		if err != nil {
			t.Fatalf("creating zip entry for %q: %v", name, err)
		}
		if _, err := fw.Write([]byte(content)); err != nil {
			t.Fatalf("writing zip content for %q: %v", name, err)
		}
	}

	if err := zw.Close(); err != nil {
		t.Fatalf("closing zip writer: %v", err)
	}
	return buf.Bytes()
}

func TestExtractZipPathTraversal(t *testing.T) {
	t.Run("rejected paths", func(t *testing.T) {
		tests := []struct {
			name  string
			entry string
		}{
			{"direct traversal", "../escape.txt"},
			{"relative traversal", "subdir/../../escape.txt"},
			{"deep traversal", "foo/bar/../../../etc/shadow"},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				data := createZipBytes(t, map[string]string{tt.entry: "malicious"})
				destDir := t.TempDir()
				_, err := ExtractZip(bytes.NewReader(data), int64(len(data)), destDir)
				if err == nil {
					t.Errorf("expected error for path %q, got nil", tt.entry)
				}
				if err != nil && !strings.Contains(err.Error(), "invalid path") {
					t.Errorf("expected 'invalid path' error for %q, got: %v", tt.entry, err)
				}

				// Verify no files were written outside destDir.
				entries, _ := os.ReadDir(destDir)
				if len(entries) > 0 {
					t.Errorf("expected no files written for traversal path, found %d entries", len(entries))
				}
			})
		}
	})

	t.Run("accepted safe path", func(t *testing.T) {
		data := createZipBytes(t, map[string]string{
			"manifest/safe.txt": "safe content",
		})
		destDir := t.TempDir()
		_, err := ExtractZip(bytes.NewReader(data), int64(len(data)), destDir)
		if err != nil {
			t.Fatalf("unexpected error for safe path: %v", err)
		}

		got, err := os.ReadFile(filepath.Join(destDir, "manifest", "safe.txt"))
		if err != nil {
			t.Fatalf("reading extracted file: %v", err)
		}
		if string(got) != "safe content" {
			t.Errorf("got %q, want %q", got, "safe content")
		}
	})
}

func TestExtractAutoZip(t *testing.T) {
	data := createZipBytes(t, map[string]string{
		"manifest/test.txt": "zip content",
	})
	destDir := t.TempDir()
	result, err := ExtractAuto(bytes.NewReader(data), destDir)
	if err != nil {
		t.Fatalf("ExtractAuto (zip): %v", err)
	}

	got, err := os.ReadFile(filepath.Join(result.Path, "test.txt"))
	if err != nil {
		t.Fatalf("reading extracted file: %v", err)
	}
	if string(got) != "zip content" {
		t.Errorf("got %q, want %q", got, "zip content")
	}
}

func TestExtractAutoTarGz(t *testing.T) {
	data := createTarGzBytes(t, map[string]string{
		"manifest/test.txt": "targz content",
	})
	destDir := t.TempDir()
	result, err := ExtractAuto(bytes.NewReader(data), destDir)
	if err != nil {
		t.Fatalf("ExtractAuto (tar.gz): %v", err)
	}

	got, err := os.ReadFile(filepath.Join(result.Path, "test.txt"))
	if err != nil {
		t.Fatalf("reading extracted file: %v", err)
	}
	if string(got) != "targz content" {
		t.Errorf("got %q, want %q", got, "targz content")
	}
}

func TestExtractAutoUnknownFormat(t *testing.T) {
	data := []byte("not a valid archive format")
	_, err := ExtractAuto(bytes.NewReader(data), t.TempDir())
	if err == nil {
		t.Error("expected error for unknown format")
	}
	if !strings.Contains(err.Error(), "unrecognized archive format") {
		t.Errorf("expected 'unrecognized archive format' error, got: %v", err)
	}
}

// Archives ReMemory makes must be extractable by `rememory recover`, which
// caps files at core.MaxFileSize. A file over it used to be archived, then
// refused halfway through extraction, leaving a partial folder.
func TestOversizedFileRefusedBeforeWriting(t *testing.T) {
	src := t.TempDir()
	if err := os.WriteFile(filepath.Join(src, "a.txt"), []byte("small"), 0644); err != nil {
		t.Fatal(err)
	}
	big, err := os.Create(filepath.Join(src, "z-video.mp4"))
	if err != nil {
		t.Fatal(err)
	}
	big.Truncate(core.MaxFileSize + 1) // sparse: cheap to create
	big.Close()

	err = CheckSizes(src)
	if err == nil || !strings.Contains(err.Error(), "z-video.mp4") {
		t.Errorf("CheckSizes: got %v, want an error naming z-video.mp4", err)
	}

	// An archive that holds such a file (made by an older version) is refused
	// before anything is written.
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, _ := zw.Create("manifest/a.txt")
	w.Write([]byte("small"))
	w, _ = zw.Create("manifest/z-video.mp4")
	w.Write(make([]byte, core.MaxFileSize+1))
	zw.Close()

	dest := filepath.Join(t.TempDir(), "recovered") // as `rememory recover` would name it
	_, err = ExtractZip(bytes.NewReader(buf.Bytes()), int64(buf.Len()), dest)
	if err == nil || !strings.Contains(err.Error(), "z-video.mp4") {
		t.Errorf("ExtractZip: got %v, want an error naming z-video.mp4", err)
	}
	if _, err := os.Stat(dest); !os.IsNotExist(err) {
		t.Errorf("ExtractZip created %s before refusing", filepath.Base(dest))
	}
}
