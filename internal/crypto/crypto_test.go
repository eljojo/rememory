package crypto

import (
	"os"
	"strings"
	"testing"

	"github.com/eljojo/rememory/internal/core"
)

func TestGeneratePassphrase(t *testing.T) {
	tests := []struct {
		name    string
		bytes   int
		wantErr bool
	}{
		{"default", DefaultPassphraseBytes, false},
		{"minimum", 16, false},
		{"large", 64, false},
		{"too small", 8, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pass, err := GeneratePassphrase(tt.bytes)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if pass == "" {
				t.Error("empty passphrase")
			}
			// Check it's valid base64
			if strings.ContainsAny(pass, "+/=") {
				t.Error("passphrase should be URL-safe base64")
			}
		})
	}

	// Test uniqueness
	t.Run("unique", func(t *testing.T) {
		p1, _ := GeneratePassphrase(32)
		p2, _ := GeneratePassphrase(32)
		if p1 == p2 {
			t.Error("passphrases should be unique")
		}
	})
}

func TestHashFile(t *testing.T) {
	// Create a temp file
	dir := t.TempDir()
	path := dir + "/test.txt"
	content := []byte("hello world")
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}

	h, err := HashFile(path)
	if err != nil {
		t.Fatalf("HashFile: %v", err)
	}

	if !strings.HasPrefix(h, "sha256:") {
		t.Errorf("hash should have sha256: prefix, got %s", h)
	}

	// Should match core.HashBytes of the same content
	expected := core.HashBytes(content)
	if h != expected {
		t.Errorf("HashFile != core.HashBytes: got %s, want %s", h, expected)
	}
}

func TestHashFileNotFound(t *testing.T) {
	_, err := HashFile("/nonexistent/path/file.txt")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}
