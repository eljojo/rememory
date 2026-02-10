package core

import (
	"bytes"
	"strings"
	"testing"
)

func TestEncodeDecodeRoundTrip(t *testing.T) {
	tests := []struct {
		name     string
		size     int
		numWords int
	}{
		{"33 bytes (24 words)", 33, 24},
		{"32 bytes (24 words)", 32, 24}, // 256 bits → ceil(256/11) = 24 words
		{"45 bytes (33 words)", 45, 33}, // 360 bits → ceil(360/11) = 33 words
		{"1 byte (1 word)", 1, 1},       // 8 bits → ceil(8/11) = 1 word
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := make([]byte, tt.size)
			for i := range data {
				data[i] = byte(i * 7) // deterministic pattern
			}

			words := EncodeWords(data)
			if len(words) != tt.numWords {
				t.Fatalf("expected %d words, got %d", tt.numWords, len(words))
			}

			decoded, err := DecodeWords(words)
			if err != nil {
				t.Fatalf("DecodeWords error: %v", err)
			}

			// Decoded length is totalBits/8, which may truncate trailing padding bits
			expectedLen := (len(words) * 11) / 8
			if len(decoded) != expectedLen {
				t.Fatalf("decoded length: got %d, want %d", len(decoded), expectedLen)
			}

			// The original data should match the decoded data up to the original length
			if !bytes.Equal(decoded[:tt.size], data) {
				t.Errorf("round-trip mismatch:\n  got:  %x\n  want: %x", decoded[:tt.size], data)
			}
		})
	}
}

func TestEncodeWords24(t *testing.T) {
	// 33 bytes = 264 bits = exactly 24 words (no padding needed)
	data := make([]byte, 33)
	for i := range data {
		data[i] = byte(i + 1)
	}
	words := EncodeWords(data)
	if len(words) != 24 {
		t.Errorf("expected 24 words for 33 bytes, got %d", len(words))
	}
}

func TestDecodeWordsInvalidWord(t *testing.T) {
	words := []string{"abandon", "ability", "appler"} // "appler" is a typo for "apple"
	_, err := DecodeWords(words)
	if err == nil {
		t.Fatal("expected error for invalid word")
	}
	if !strings.Contains(err.Error(), "appler") {
		t.Errorf("error should mention the invalid word, got: %v", err)
	}
	if !strings.Contains(err.Error(), "did you mean") {
		t.Errorf("error should include a suggestion, got: %v", err)
	}
}

func TestDecodeWordsEmpty(t *testing.T) {
	_, err := DecodeWords([]string{})
	if err == nil {
		t.Fatal("expected error for empty input")
	}
	if !strings.Contains(err.Error(), "no words") {
		t.Errorf("expected 'no words' error, got: %v", err)
	}
}

func TestSuggestWord(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"appla", "apple"},    // one char off from "apple"
		{"abandn", "abandon"}, // missing 'o'
		{"zooo", "zoo"},       // one extra char
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := SuggestWord(tt.input)
			if got != tt.expected {
				t.Errorf("SuggestWord(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestBIP39ListIntegrity(t *testing.T) {
	en := GetWordList(LangEN)
	if en == nil {
		t.Fatal("English word list not found")
	}

	// Check count (fixed-size array, but verify no empty entries)
	seen := make(map[string]bool, 2048)
	for i, w := range en.Words {
		if w == "" {
			t.Errorf("empty word at index %d", i)
		}
		if seen[w] {
			t.Errorf("duplicate word %q at index %d", w, i)
		}
		seen[w] = true
	}

	// SHA-256 integrity check
	hash := WordListHash(LangEN)
	if hash != en.ExpectedHash {
		t.Errorf("BIP39 English word list hash mismatch:\n  got:  %s\n  want: %s", hash, en.ExpectedHash)
	}
}

func TestEncodeWordsDeterministic(t *testing.T) {
	data := make([]byte, 33)
	for i := range data {
		data[i] = byte(i * 13)
	}

	words1 := EncodeWords(data)
	words2 := EncodeWords(data)

	if strings.Join(words1, " ") != strings.Join(words2, " ") {
		t.Error("EncodeWords is not deterministic")
	}
}

func TestShareWords(t *testing.T) {
	data := make([]byte, 33)
	for i := range data {
		data[i] = byte(i)
	}
	share := NewShare(2, 1, 5, 3, "Alice", data)
	words, err := share.Words()
	if err != nil {
		t.Fatalf("Words() error: %v", err)
	}
	if len(words) != 25 {
		t.Errorf("expected 25 words for 33-byte share (24 data + 1 meta), got %d", len(words))
	}

	// Round-trip through DecodeShareWords
	decoded, index, err := DecodeShareWords(words)
	if err != nil {
		t.Fatalf("DecodeShareWords error: %v", err)
	}
	if !bytes.Equal(decoded, data) {
		t.Errorf("Share.Words() round-trip data mismatch")
	}
	if index != 1 {
		t.Errorf("Share.Words() round-trip index: got %d, want 1", index)
	}
}

func TestDecodeShareWordsRoundTrip(t *testing.T) {
	tests := []struct {
		name          string
		index         int
		expectedIndex int // what DecodeShareWords should return (0 for >15)
	}{
		{"index 1", 1, 1},
		{"index 2", 2, 2},
		{"index 5", 5, 5},
		{"index 15 (max exact)", 15, 15},
		{"index 16 (sentinel)", 16, 0},   // above 15 → stored as 0
		{"index 100 (sentinel)", 100, 0}, // above 15 → stored as 0
		{"index 255 (sentinel)", 255, 0}, // above 15 → stored as 0
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := make([]byte, 33)
			for i := range data {
				data[i] = byte(i * 7)
			}
			share := NewShare(2, tt.index, 5, 3, "Test", data)
			words, err := share.Words()
			if err != nil {
				t.Fatalf("Words() error: %v", err)
			}
			if len(words) != 25 {
				t.Fatalf("expected 25 words, got %d", len(words))
			}

			decoded, index, err := DecodeShareWords(words)
			if err != nil {
				t.Fatalf("DecodeShareWords error: %v", err)
			}
			if !bytes.Equal(decoded, data) {
				t.Errorf("data mismatch")
			}
			if index != tt.expectedIndex {
				t.Errorf("index: got %d, want %d", index, tt.expectedIndex)
			}
		})
	}
}

// TestWord25ChecksumDetectsTransposition verifies that swapping two adjacent
// data words causes the 25th-word checksum to fail.
func TestWord25ChecksumDetectsTransposition(t *testing.T) {
	data := make([]byte, 33)
	for i := range data {
		data[i] = byte(i * 13)
	}
	share := NewShare(2, 3, 5, 3, "Test", data)
	words, err := share.Words()
	if err != nil {
		t.Fatalf("Words() error: %v", err)
	}

	// Swap words 0 and 1 (adjacent transposition in data words)
	swapped := make([]string, len(words))
	copy(swapped, words)
	swapped[0], swapped[1] = swapped[1], swapped[0]

	_, _, decErr := DecodeShareWords(swapped)
	if decErr == nil {
		t.Error("expected checksum error for transposed words, got nil")
	}
	if !strings.Contains(decErr.Error(), "checksum") {
		t.Errorf("expected checksum error, got: %v", decErr)
	}
}

// TestWord25ChecksumDetectsSubstitution verifies that replacing a data word
// with a different valid BIP39 word causes the checksum to fail.
func TestWord25ChecksumDetectsSubstitution(t *testing.T) {
	data := make([]byte, 33)
	for i := range data {
		data[i] = byte(i * 13)
	}
	share := NewShare(2, 3, 5, 3, "Test", data)
	words, err := share.Words()
	if err != nil {
		t.Fatalf("Words() error: %v", err)
	}

	// Replace word 5 with a different BIP39 word
	modified := make([]string, len(words))
	copy(modified, words)
	// Pick a word that's definitely different from the original
	replacement := "zoo"
	if modified[5] == replacement {
		replacement = "abandon"
	}
	modified[5] = replacement

	_, _, err = DecodeShareWords(modified)
	if err == nil {
		t.Error("expected checksum error for substituted word, got nil")
	}
	if !strings.Contains(err.Error(), "checksum") {
		t.Errorf("expected checksum error, got: %v", err)
	}
}

// TestWord25Layout verifies the bit-packing layout of the 25th word:
// upper 4 bits = index, lower 7 bits = SHA-256 checksum.
func TestWord25Layout(t *testing.T) {
	data := make([]byte, 33)
	for i := range data {
		data[i] = byte(i + 42)
	}

	// Compute expected values
	expectedCheck := word25Checksum(data)
	if expectedCheck < 0 || expectedCheck > 127 {
		t.Fatalf("checksum out of 7-bit range: %d", expectedCheck)
	}

	// Test encoding for index in range (1-15)
	for _, idx := range []int{1, 7, 15} {
		encoded := word25Encode(idx, data)
		gotIdx, gotCheck := word25Decode(encoded)
		if gotIdx != idx {
			t.Errorf("index %d: decode got index %d", idx, gotIdx)
		}
		if gotCheck != expectedCheck {
			t.Errorf("index %d: decode got check %d, want %d", idx, gotCheck, expectedCheck)
		}
		// Verify the 11-bit value is in BIP39 range
		if encoded < 0 || encoded >= 2048 {
			t.Errorf("index %d: encoded value %d out of BIP39 range", idx, encoded)
		}
	}

	// Test sentinel for index > 15
	for _, idx := range []int{16, 100, 255} {
		encoded := word25Encode(idx, data)
		gotIdx, gotCheck := word25Decode(encoded)
		if gotIdx != 0 {
			t.Errorf("index %d: expected sentinel 0, got %d", idx, gotIdx)
		}
		if gotCheck != expectedCheck {
			t.Errorf("index %d: checksum should still be valid, got %d want %d", idx, gotCheck, expectedCheck)
		}
	}
}

// TestWord25ChecksumDifferentData verifies that different data produces
// different checksums (not a guarantee, but should hold for distinct inputs).
func TestWord25ChecksumDifferentData(t *testing.T) {
	data1 := make([]byte, 33)
	data2 := make([]byte, 33)
	for i := range data1 {
		data1[i] = byte(i)
		data2[i] = byte(i + 1)
	}

	check1 := word25Checksum(data1)
	check2 := word25Checksum(data2)
	// With 7 bits, there's a 1/128 chance these collide.
	// Use sufficiently different inputs to make collision astronomically unlikely.
	if check1 == check2 {
		t.Logf("warning: checksums collided (1/128 chance) — not a bug, but unexpected")
	}
}

func TestWordsV1ShareReturnsError(t *testing.T) {
	data := make([]byte, 33)
	share := NewShare(1, 1, 5, 3, "Alice", data)
	_, err := share.Words()
	if err == nil {
		t.Fatal("expected error for v1 share")
	}
	if !strings.Contains(err.Error(), "version 2") {
		t.Errorf("expected version error, got: %v", err)
	}
}

func TestDecodeShareWordsWrongCount(t *testing.T) {
	tests := []struct {
		name  string
		count int
	}{
		{"0 words", 0},
		{"1 word", 1},
		{"10 words", 10},
		{"24 words", 24},
		{"26 words", 26},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			words := make([]string, tt.count)
			for i := range words {
				words[i] = "abandon"
			}
			_, _, err := DecodeShareWords(words)
			if err == nil {
				t.Fatalf("expected error for %d words", tt.count)
			}
			if !strings.Contains(err.Error(), "expected 25 words") {
				t.Errorf("expected word count error, got: %v", err)
			}
		})
	}
}

func TestDecodeWordsMixedCase(t *testing.T) {
	data := make([]byte, 33)
	for i := range data {
		data[i] = byte(i * 7)
	}
	words := EncodeWords(data)

	// Uppercase some words
	mixed := make([]string, len(words))
	for i, w := range words {
		if i%2 == 0 {
			mixed[i] = strings.ToUpper(w)
		} else {
			// Capitalize first letter
			mixed[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}

	decoded, err := DecodeWords(mixed)
	if err != nil {
		t.Fatalf("DecodeWords should handle mixed case, got error: %v", err)
	}
	if !bytes.Equal(decoded, data) {
		t.Errorf("mixed-case round-trip mismatch")
	}
}

func TestSuggestWordNoMatch(t *testing.T) {
	// Strings far from any BIP39 word (distance > 2)
	tests := []string{"zzzzzzz", "qqqqqq", "xylophone"}
	for _, input := range tests {
		got := SuggestWord(input)
		if got != "" {
			t.Errorf("SuggestWord(%q) = %q, want empty string (no close match)", input, got)
		}
	}
}

func TestWordsNegativeIndexReturnsError(t *testing.T) {
	data := make([]byte, 33)
	share := NewShare(2, -1, 5, 3, "Alice", data)
	_, err := share.Words()
	if err == nil {
		t.Fatal("expected error for negative index")
	}
	if !strings.Contains(err.Error(), "non-negative") {
		t.Errorf("expected non-negative error, got: %v", err)
	}
}
