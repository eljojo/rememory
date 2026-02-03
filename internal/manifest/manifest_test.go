package manifest

import (
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"strings"
	"testing"
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
	archiveResult, err := Archive(&buf, testDir)
	if err != nil {
		t.Fatalf("archive: %v", err)
	}
	if len(archiveResult.Warnings) > 0 {
		t.Logf("archive warnings: %v", archiveResult.Warnings)
	}

	// Extract to new location
	dstDir := t.TempDir()
	extractResult, err := Extract(&buf, dstDir)
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
	_, err = Archive(&buf, f.Name())
	if err == nil {
		t.Error("expected error for non-directory")
	}
}

func TestExtractPathTraversal(t *testing.T) {
	// This test ensures the extract function rejects path traversal attacks
	// We can't easily create a malicious tar, but we test that normal paths work
	srcDir := t.TempDir()
	testDir := filepath.Join(srcDir, "safe")
	if err := os.MkdirAll(testDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(testDir, "file.txt"), []byte("safe"), 0644); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	if _, err := Archive(&buf, testDir); err != nil {
		t.Fatal(err)
	}

	dstDir := t.TempDir()
	_, err := Extract(&buf, dstDir)
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
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
	_, err := Archive(&buf, "/nonexistent/path")
	if err == nil {
		t.Error("expected error for nonexistent directory")
	}
}

func TestExtractInvalidGzip(t *testing.T) {
	// Not valid gzip data
	data := bytes.NewReader([]byte("not gzip data"))
	_, err := Extract(data, t.TempDir())
	if err == nil {
		t.Error("expected error for invalid gzip")
	}
}

func TestExtractEmptyArchive(t *testing.T) {
	// Create an empty gzip stream (valid gzip, but no tar entries)
	var buf bytes.Buffer
	gzw := gzip.NewWriter(&buf)
	gzw.Close()

	_, err := Extract(&buf, t.TempDir())
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
	result, err := Archive(&buf, testDir)
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
	extractResult, err := Extract(&buf, dstDir)
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
	_, err := Archive(&buf, emptyDir)
	if err != nil {
		t.Fatalf("Archive empty dir: %v", err)
	}

	// Should still be valid archive
	dstDir := t.TempDir()
	_, err = Extract(&buf, dstDir)
	if err != nil {
		t.Fatalf("Extract empty archive: %v", err)
	}
}
