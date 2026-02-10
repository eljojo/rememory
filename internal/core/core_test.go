package core

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"strings"
	"testing"
)

func TestHashString(t *testing.T) {
	h := HashString("hello")
	if !strings.HasPrefix(h, "sha256:") {
		t.Errorf("hash should have sha256: prefix, got %s", h)
	}
	if len(h) != 7+64 { // "sha256:" + 64 hex chars
		t.Errorf("unexpected hash length: %d", len(h))
	}

	// Same input should produce same hash
	h2 := HashString("hello")
	if h != h2 {
		t.Error("same input should produce same hash")
	}

	// Different input should produce different hash
	h3 := HashString("world")
	if h == h3 {
		t.Error("different input should produce different hash")
	}
}

func TestHashBytes(t *testing.T) {
	h := HashBytes([]byte{1, 2, 3})
	if !strings.HasPrefix(h, "sha256:") {
		t.Errorf("hash should have sha256: prefix, got %s", h)
	}

	// HashBytes and HashString should produce same result for same content
	h2 := HashString("hello")
	h3 := HashBytes([]byte("hello"))
	if h2 != h3 {
		t.Error("HashString and HashBytes should produce same result")
	}
}

func TestVerifyHash(t *testing.T) {
	hash := HashString("test")

	if !VerifyHash(hash, hash) {
		t.Error("identical hashes should verify")
	}

	if VerifyHash(hash, "sha256:wrong") {
		t.Error("different hashes should not verify")
	}
}

func TestEncryptDecrypt(t *testing.T) {
	tests := []struct {
		name string
		data string
	}{
		{"small", "hello world"},
		{"empty", ""},
		{"large", strings.Repeat("x", 10000)},
		{"unicode", "Hello 世界 🌍"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			passphrase := "test-passphrase-12345"

			// Encrypt
			var encrypted bytes.Buffer
			err := Encrypt(&encrypted, strings.NewReader(tt.data), passphrase)
			if err != nil {
				t.Fatalf("encrypt: %v", err)
			}

			// Decrypt
			var decrypted bytes.Buffer
			err = Decrypt(&decrypted, &encrypted, passphrase)
			if err != nil {
				t.Fatalf("decrypt: %v", err)
			}

			if decrypted.String() != tt.data {
				t.Errorf("got %q, want %q", decrypted.String(), tt.data)
			}
		})
	}
}

func TestDecryptBytes(t *testing.T) {
	data := []byte("secret data")
	passphrase := "test-passphrase"

	var encrypted bytes.Buffer
	if err := Encrypt(&encrypted, bytes.NewReader(data), passphrase); err != nil {
		t.Fatalf("encrypt: %v", err)
	}

	decrypted, err := DecryptBytes(encrypted.Bytes(), passphrase)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}

	if !bytes.Equal(decrypted, data) {
		t.Errorf("got %q, want %q", decrypted, data)
	}
}

func TestDecryptWrongPassphrase(t *testing.T) {
	data := []byte("secret data")
	correctPass := "correct-passphrase"
	wrongPass := "wrong-passphrase"

	var encrypted bytes.Buffer
	if err := Encrypt(&encrypted, bytes.NewReader(data), correctPass); err != nil {
		t.Fatalf("encrypt: %v", err)
	}

	_, err := DecryptBytes(encrypted.Bytes(), wrongPass)
	if err == nil {
		t.Error("expected error with wrong passphrase")
	}
}

func TestSplitCombine(t *testing.T) {
	secret := []byte("my-super-secret-passphrase")

	tests := []struct {
		name string
		n    int // total shares
		k    int // threshold
	}{
		{"2-of-2", 2, 2},
		{"2-of-3", 3, 2},
		{"3-of-5", 5, 3},
		{"5-of-5", 5, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shares, err := Split(secret, tt.n, tt.k)
			if err != nil {
				t.Fatalf("split: %v", err)
			}

			if len(shares) != tt.n {
				t.Errorf("got %d shares, want %d", len(shares), tt.n)
			}

			// Test with exactly threshold shares
			recovered, err := Combine(shares[:tt.k])
			if err != nil {
				t.Fatalf("combine: %v", err)
			}

			if string(recovered) != string(secret) {
				t.Errorf("got %q, want %q", recovered, secret)
			}
		})
	}
}

func TestValidateShamirParams(t *testing.T) {
	tests := []struct {
		name    string
		n       int
		k       int
		wantErr bool
	}{
		{"valid 3-of-5", 5, 3, false},
		{"k=1", 3, 1, true},
		{"k>n", 3, 5, true},
		{"n>255", 300, 3, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateShamirParams(tt.n, tt.k)
			if tt.wantErr && err == nil {
				t.Error("expected error")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestShareEncodeDecode(t *testing.T) {
	original := NewShare(1, 5, 3, "Alice", []byte("test-share-data"))

	encoded := original.Encode()

	decoded, err := ParseShare([]byte(encoded))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	if decoded.Version != original.Version {
		t.Errorf("version: got %d, want %d", decoded.Version, original.Version)
	}
	if decoded.Index != original.Index {
		t.Errorf("index: got %d, want %d", decoded.Index, original.Index)
	}
	if decoded.Total != original.Total {
		t.Errorf("total: got %d, want %d", decoded.Total, original.Total)
	}
	if decoded.Threshold != original.Threshold {
		t.Errorf("threshold: got %d, want %d", decoded.Threshold, original.Threshold)
	}
	if decoded.Holder != original.Holder {
		t.Errorf("holder: got %q, want %q", decoded.Holder, original.Holder)
	}
	if string(decoded.Data) != string(original.Data) {
		t.Errorf("data: got %q, want %q", decoded.Data, original.Data)
	}
	if decoded.Checksum != original.Checksum {
		t.Errorf("checksum: got %q, want %q", decoded.Checksum, original.Checksum)
	}
}

func TestShareVerify(t *testing.T) {
	share := NewShare(1, 5, 3, "Alice", []byte("test-data"))

	// Valid checksum
	if err := share.Verify(); err != nil {
		t.Errorf("valid share failed verify: %v", err)
	}

	// Corrupted checksum
	share.Checksum = "sha256:wrong"
	if err := share.Verify(); err == nil {
		t.Error("corrupted share should fail verify")
	}
}

func TestShareFilename(t *testing.T) {
	tests := []struct {
		holder   string
		expected string
	}{
		{"Alice", "SHARE-alice.txt"},
		{"Bob Smith", "SHARE-bob-smith.txt"},
		{"Carol!", "SHARE-carol.txt"},
		{"", "SHARE-1.txt"},
	}

	for _, tt := range tests {
		share := NewShare(1, 3, 2, tt.holder, []byte("data"))
		got := share.Filename()
		if got != tt.expected {
			t.Errorf("holder %q: got %q, want %q", tt.holder, got, tt.expected)
		}
	}
}

func TestCompactEncodeRoundTrip(t *testing.T) {
	original := NewShare(1, 5, 3, "Alice", []byte("test-share-data-1234567890"))

	compact := original.CompactEncode()

	decoded, err := ParseCompact(compact)
	if err != nil {
		t.Fatalf("ParseCompact: %v", err)
	}

	if decoded.Version != original.Version {
		t.Errorf("version: got %d, want %d", decoded.Version, original.Version)
	}
	if decoded.Index != original.Index {
		t.Errorf("index: got %d, want %d", decoded.Index, original.Index)
	}
	if decoded.Total != original.Total {
		t.Errorf("total: got %d, want %d", decoded.Total, original.Total)
	}
	if decoded.Threshold != original.Threshold {
		t.Errorf("threshold: got %d, want %d", decoded.Threshold, original.Threshold)
	}
	if !bytes.Equal(decoded.Data, original.Data) {
		t.Errorf("data mismatch")
	}
	if decoded.Checksum != original.Checksum {
		t.Errorf("checksum: got %q, want %q", decoded.Checksum, original.Checksum)
	}
}

func TestCompactEncodeWithRealShares(t *testing.T) {
	secret := []byte("a]Zp9kR-mN2xB7qL_YwF4vC8hD6sE0jT")
	shares, err := Split(secret, 5, 3)
	if err != nil {
		t.Fatalf("Split: %v", err)
	}

	for i, shareData := range shares {
		share := NewShare(i+1, 5, 3, "", shareData)
		compact := share.CompactEncode()

		decoded, err := ParseCompact(compact)
		if err != nil {
			t.Fatalf("share %d: ParseCompact(%q): %v", i+1, compact, err)
		}
		if !bytes.Equal(decoded.Data, shareData) {
			t.Errorf("share %d: data mismatch after round-trip", i+1)
		}
	}
}

func TestCompactEncodeFormat(t *testing.T) {
	share := NewShare(2, 5, 3, "Bob", []byte{0xDE, 0xAD, 0xBE, 0xEF})
	compact := share.CompactEncode()

	if !strings.HasPrefix(compact, "RM1:") {
		t.Errorf("should start with RM1:, got %q", compact)
	}

	parts := strings.Split(compact, ":")
	if len(parts) != 6 {
		t.Fatalf("expected 6 parts, got %d: %q", len(parts), compact)
	}
	if parts[0] != "RM1" {
		t.Errorf("version prefix: got %q, want RM1", parts[0])
	}
	if parts[1] != "2" {
		t.Errorf("index: got %q, want 2", parts[1])
	}
	if parts[2] != "5" {
		t.Errorf("total: got %q, want 5", parts[2])
	}
	if parts[3] != "3" {
		t.Errorf("threshold: got %q, want 3", parts[3])
	}
	// base64url of 0xDEADBEEF with no padding
	if parts[4] != "3q2-7w" {
		t.Errorf("data: got %q, want 3q2-7w", parts[4])
	}
	// checksum should be 4 hex chars
	if len(parts[5]) != 4 {
		t.Errorf("checksum length: got %d, want 4", len(parts[5]))
	}
}

func TestParseCompactRejectsBadInput(t *testing.T) {
	// Build a valid compact string to use as a base
	share := NewShare(1, 5, 3, "Alice", []byte("valid-data"))
	valid := share.CompactEncode()

	tests := []struct {
		name  string
		input string
	}{
		{"empty string", ""},
		{"too few fields", "RM1:1:5:3:data"},
		{"too many fields", "RM1:1:5:3:data:check:extra"},
		{"wrong prefix", "XX1:1:5:3:AAAA:0000"},
		{"bad version", "RMx:1:5:3:AAAA:0000"},
		{"zero version", "RM0:1:5:3:AAAA:0000"},
		{"negative index", "RM1:-1:5:3:AAAA:0000"},
		{"zero index", "RM1:0:5:3:AAAA:0000"},
		{"zero total", "RM1:1:0:3:AAAA:0000"},
		{"zero threshold", "RM1:1:5:0:AAAA:0000"},
		{"bad base64", "RM1:1:5:3:!!!invalid!!!:0000"},
		{"wrong checksum", valid[:len(valid)-4] + "ffff"},
		{"truncated", valid[:len(valid)/2]},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseCompact(tt.input)
			if err == nil {
				t.Errorf("expected error for %q, got nil", tt.input)
			}
		})
	}
}

func TestCompactEncodeNoHolderOrCreated(t *testing.T) {
	// Compact format intentionally omits Holder and Created metadata
	// to keep the string short for QR codes
	share := NewShare(3, 7, 4, "Carol with spaces", []byte("some-share-data"))
	compact := share.CompactEncode()
	decoded, err := ParseCompact(compact)
	if err != nil {
		t.Fatalf("ParseCompact: %v", err)
	}
	if decoded.Holder != "" {
		t.Errorf("holder should be empty in compact format, got %q", decoded.Holder)
	}
	if !decoded.Created.IsZero() {
		t.Errorf("created should be zero in compact format, got %v", decoded.Created)
	}
}

// createTarGz builds a tar.gz archive in memory with arbitrary entry names.
// This allows crafting malicious archives for security testing.
func createTarGz(t *testing.T, entries map[string]string) []byte {
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

func TestExtractTarGzPathTraversal(t *testing.T) {
	t.Run("rejected paths", func(t *testing.T) {
		tests := []struct {
			name  string
			entry string
		}{
			{"classic traversal", "../etc/passwd"},
			{"mid-path traversal", "foo/../../etc/passwd"},
			{"deep traversal", "foo/bar/../../../etc/shadow"},
			{"bare dotdot", ".."},
			{"trailing dotdot", "foo/.."},
			// foo/../bar is also rejected by the regex because it matches
			// `..` between slashes. This is intentionally conservative for
			// in-memory extraction where paths cannot be resolved.
			{"non-escaping dotdot", "foo/../bar"},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				data := createTarGz(t, map[string]string{tt.entry: "malicious"})
				_, err := ExtractTarGz(data)
				if err == nil {
					t.Errorf("expected error for path %q, got nil", tt.entry)
				}
				if err != nil && !strings.Contains(err.Error(), "invalid path") {
					t.Errorf("expected 'invalid path' error for %q, got: %v", tt.entry, err)
				}
			})
		}
	})

	t.Run("accepted paths", func(t *testing.T) {
		entries := map[string]string{
			"safe/file.txt":        "hello",
			"safe/nested/deep.txt": "world",
		}
		data := createTarGz(t, entries)
		files, err := ExtractTarGz(data)
		if err != nil {
			t.Fatalf("unexpected error for safe paths: %v", err)
		}

		extracted := make(map[string]string)
		for _, f := range files {
			extracted[f.Name] = string(f.Data)
		}

		for name, want := range entries {
			got, ok := extracted[name]
			if !ok {
				t.Errorf("missing extracted file %q", name)
				continue
			}
			if got != want {
				t.Errorf("file %q: got %q, want %q", name, got, want)
			}
		}
	})

	t.Run("empty input", func(t *testing.T) {
		_, err := ExtractTarGz([]byte{})
		if err == nil {
			t.Error("expected error for empty input")
		}
	})
}

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Alice", "alice"},
		{"Bob Smith", "bob-smith"},
		{"Carol!", "carol"},
		{"test@user.com", "testusercom"},
		{"file/path", "filepath"},
		// NFD transliteration: accented chars → ASCII base
		{"José", "jose"},
		{"Ñoño", "nono"},
		{"Müller", "muller"},
		// Underscore → hyphen
		{"bob_smith", "bob-smith"},
		// Hyphen trimming and collapsing
		{"---hyphens---", "hyphens"},
		{"a--b", "a-b"},
		// Leading/trailing spaces
		{"  Alice  ", "alice"},
		// Path traversal chars stripped
		{"../etc/passwd", "etcpasswd"},
		// Non-ASCII input that sanitizes to empty
		{"日本語", ""},
		{"!!!???", ""},
		// Empty input
		{"", ""},
	}

	for _, tt := range tests {
		got := SanitizeFilename(tt.input)
		if got != tt.expected {
			t.Errorf("SanitizeFilename(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}
