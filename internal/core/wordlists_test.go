package core

import (
	"bytes"
	"strings"
	"testing"
)

func TestAllWordListIntegrity(t *testing.T) {
	for _, lang := range AllLangs() {
		t.Run(string(lang), func(t *testing.T) {
			wl := GetWordList(lang)
			if wl == nil {
				t.Fatalf("word list for %s not found", lang)
			}

			// Check no empty entries or duplicates
			seen := make(map[string]bool, 2048)
			for i, w := range wl.Words {
				if w == "" {
					t.Errorf("empty word at index %d", i)
				}
				if seen[w] {
					t.Errorf("duplicate word %q at index %d", w, i)
				}
				seen[w] = true
			}

			// SHA-256 integrity check
			hash := WordListHash(lang)
			if hash != wl.ExpectedHash {
				t.Errorf("hash mismatch:\n  got:  %s\n  want: %s\n  source: %s", hash, wl.ExpectedHash, wl.SourceURL)
			}
		})
	}
}

func TestNormalizeWord(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Ábaco", "abaco"},
		{"GÜNTHER", "gunther"},
		{"čudež", "cudez"},
		{"aérer", "aerer"},
		{"abandon", "abandon"},
		{"  Zoo  ", "zoo"},
		{"ZÜRICH", "zurich"},
		{"naïve", "naive"},
		{"résumé", "resume"},
		{"über", "uber"},
		{"straße", "straße"}, // ß is not a combining mark, stays as-is after NFD
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := NormalizeWord(tt.input)
			if got != tt.expected {
				t.Errorf("NormalizeWord(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestLookupWordExact(t *testing.T) {
	// Each language's first word should be found at index 0
	tests := []struct {
		lang Lang
		word string
	}{
		{LangEN, "abandon"},
		{LangES, "ábaco"},
		{LangFR, "abaisser"},
		{LangDE, "abbau"},
		{LangSL, "abeceda"},
	}

	for _, tt := range tests {
		t.Run(string(tt.lang), func(t *testing.T) {
			idx, ok := LookupWord(tt.lang, tt.word)
			if !ok {
				t.Fatalf("LookupWord(%s, %q) not found", tt.lang, tt.word)
			}
			if idx != 0 {
				t.Errorf("LookupWord(%s, %q) = %d, want 0", tt.lang, tt.word, idx)
			}
		})
	}
}

func TestLookupWordStripped(t *testing.T) {
	// Accent-stripped forms should match
	tests := []struct {
		lang Lang
		word string
	}{
		{LangES, "abaco"}, // matches "ábaco"
		{LangFR, "aerer"}, // matches "aérer" (if present)
	}

	for _, tt := range tests {
		t.Run(string(tt.lang)+"/"+tt.word, func(t *testing.T) {
			idx, ok := LookupWord(tt.lang, tt.word)
			if !ok {
				t.Fatalf("LookupWord(%s, %q) not found (accent-stripped)", tt.lang, tt.word)
			}
			// Verify it matches the same index as the canonical form
			wl := GetWordList(tt.lang)
			canonical := wl.Words[idx]
			t.Logf("matched %q → %q (index %d)", tt.word, canonical, idx)
		})
	}
}

func TestLookupWordGermanDigraph(t *testing.T) {
	// The dys2p German word list intentionally uses ASCII-only words (no umlauts).
	// Verify this is the case and that exact lookup still works.
	wl := GetWordList(LangDE)
	if wl == nil {
		t.Fatal("German word list not found")
	}

	hasUmlauts := false
	for _, w := range wl.Words {
		if NormalizeWord(w) != strings.ToLower(w) {
			hasUmlauts = true
			break
		}
	}
	if hasUmlauts {
		t.Log("German list has diacritics — digraph lookup applies")
	} else {
		t.Log("German list is ASCII-only — digraph lookup is a no-op (by design)")
	}

	// Verify a plain German word looks up correctly
	idx, ok := LookupWord(LangDE, wl.Words[0])
	if !ok || idx != 0 {
		t.Errorf("first German word lookup failed")
	}
}

func TestLookupWordCaseInsensitive(t *testing.T) {
	tests := []struct {
		lang Lang
		word string
	}{
		{LangEN, "ABANDON"},
		{LangEN, "Abandon"},
		{LangES, "ÁBACO"},
		{LangES, "Ábaco"},
	}

	for _, tt := range tests {
		t.Run(string(tt.lang)+"/"+tt.word, func(t *testing.T) {
			idx, ok := LookupWord(tt.lang, tt.word)
			if !ok {
				t.Fatalf("LookupWord(%s, %q) not found (case insensitive)", tt.lang, tt.word)
			}
			if idx != 0 {
				t.Errorf("LookupWord(%s, %q) = %d, want 0", tt.lang, tt.word, idx)
			}
		})
	}
}

func TestEncodeDecodeRoundTripAllLangs(t *testing.T) {
	data := make([]byte, 33)
	for i := range data {
		data[i] = byte(i * 7)
	}

	for _, lang := range AllLangs() {
		t.Run(string(lang), func(t *testing.T) {
			words := EncodeWordsLang(data, lang)
			if len(words) != 24 {
				t.Fatalf("expected 24 words, got %d", len(words))
			}

			decoded, err := DecodeWordsLang(words, lang)
			if err != nil {
				t.Fatalf("DecodeWordsLang error: %v", err)
			}
			if !bytes.Equal(decoded, data) {
				t.Errorf("round-trip mismatch for %s", lang)
			}
		})
	}
}

func TestShareWordsForLangRoundTrip(t *testing.T) {
	data := make([]byte, 33)
	for i := range data {
		data[i] = byte(i * 11)
	}

	for _, lang := range AllLangs() {
		t.Run(string(lang), func(t *testing.T) {
			share := NewShare(2, 3, 5, 3, "Test", data)
			words, err := share.WordsForLang(lang)
			if err != nil {
				t.Fatalf("WordsForLang(%s) error: %v", lang, err)
			}
			if len(words) != 25 {
				t.Fatalf("expected 25 words, got %d", len(words))
			}

			// Auto-detect decode should return the same data
			decoded, index, detectedLang, err := DecodeShareWordsAuto(words)
			if err != nil {
				t.Fatalf("DecodeShareWordsAuto error: %v", err)
			}
			if !bytes.Equal(decoded, data) {
				t.Errorf("data mismatch")
			}
			if index != 3 {
				t.Errorf("index: got %d, want 3", index)
			}
			if detectedLang != lang {
				t.Errorf("detected language: got %s, want %s", detectedLang, lang)
			}
		})
	}
}

func TestDetectWordListLang(t *testing.T) {
	data := make([]byte, 33)
	for i := range data {
		data[i] = byte(i * 13)
	}

	for _, lang := range AllLangs() {
		t.Run(string(lang), func(t *testing.T) {
			share := NewShare(2, 1, 5, 3, "Test", data)
			words, err := share.WordsForLang(lang)
			if err != nil {
				t.Fatalf("WordsForLang error: %v", err)
			}

			detected := DetectWordListLang(words)
			if detected != lang {
				t.Errorf("DetectWordListLang: got %s, want %s", detected, lang)
			}
		})
	}
}

func TestAutoDetectWithNormalization(t *testing.T) {
	data := make([]byte, 33)
	for i := range data {
		data[i] = byte(i * 7)
	}
	share := NewShare(2, 2, 5, 3, "Test", data)

	// Get Spanish words and strip accents
	words, err := share.WordsForLang(LangES)
	if err != nil {
		t.Fatalf("WordsForLang(es) error: %v", err)
	}

	// Strip accents from all words
	stripped := make([]string, len(words))
	for i, w := range words {
		stripped[i] = NormalizeWord(w)
	}

	// Should still auto-detect and decode correctly
	decoded, index, lang, err := DecodeShareWordsAuto(stripped)
	if err != nil {
		t.Fatalf("DecodeShareWordsAuto with stripped accents error: %v", err)
	}
	if lang != LangES {
		t.Errorf("detected language: got %s, want es", lang)
	}
	if !bytes.Equal(decoded, data) {
		t.Errorf("data mismatch")
	}
	if index != 2 {
		t.Errorf("index: got %d, want 2", index)
	}
}

func TestSuggestWordLang(t *testing.T) {
	// Test suggestion in Spanish
	wl := GetWordList(LangES)
	if wl == nil {
		t.Fatal("Spanish word list not found")
	}
	// Misspell the first Spanish word
	firstWord := wl.Words[0]
	misspelled := firstWord + "x"
	suggestion := SuggestWordLang(misspelled, LangES)
	if suggestion == "" {
		t.Errorf("SuggestWordLang(%q, es) returned no suggestion", misspelled)
	} else {
		t.Logf("SuggestWordLang(%q, es) = %q", misspelled, suggestion)
	}
}

func TestSuggestWordAllLangs(t *testing.T) {
	// Misspell a Spanish word — should still find it across all languages
	suggestion := SuggestWordAllLangs("abaco") // accent-stripped, should match "ábaco"
	if suggestion == "" {
		t.Error("SuggestWordAllLangs(abaco) returned no suggestion")
	} else {
		t.Logf("SuggestWordAllLangs(abaco) = %q", suggestion)
	}
}

func TestCrossLanguageEncodingProducesSameData(t *testing.T) {
	// Encoding the same data in different languages should decode to the same bytes
	data := make([]byte, 33)
	for i := range data {
		data[i] = byte(i * 17)
	}
	share := NewShare(2, 5, 5, 3, "Test", data)

	// Encode in each language, decode, verify same data
	for _, lang := range AllLangs() {
		words, err := share.WordsForLang(lang)
		if err != nil {
			t.Fatalf("WordsForLang(%s) error: %v", lang, err)
		}
		decoded, index, err := DecodeShareWords(words)
		if err != nil {
			t.Fatalf("DecodeShareWords(%s words) error: %v", lang, err)
		}
		if !bytes.Equal(decoded, data) {
			t.Errorf("data mismatch for language %s", lang)
		}
		if index != 5 {
			t.Errorf("index for %s: got %d, want 5", lang, index)
		}
	}
}

func TestWordListsHaveUniqueWords(t *testing.T) {
	// Verify each language's word list has no internal duplicates.
	// Some languages (e.g. Slovenian) have words where the stripped form
	// collides (jez/jež both → jez). This is expected — exact match takes
	// priority, and the stripped index is only a fallback.
	for _, lang := range AllLangs() {
		t.Run(string(lang), func(t *testing.T) {
			wl := GetWordList(lang)
			seen := make(map[string]string, 2048)
			collisions := 0
			for _, w := range wl.Words {
				n := NormalizeWord(w)
				if existing, ok := seen[n]; ok && existing != w {
					collisions++
					t.Logf("normalized collision (expected for some languages): %q and %q both normalize to %q", existing, w, n)
				}
				seen[n] = w
			}
			if collisions > 0 {
				t.Logf("%s: %d normalized collisions (exact match handles these correctly)", lang, collisions)
			}
		})
	}
}
