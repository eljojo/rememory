package core

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

//go:embed wordlists/*.txt
var wordlistFS embed.FS

// Lang identifies a supported BIP39 word list language.
type Lang string

const (
	LangEN    Lang = "en"
	LangES    Lang = "es"
	LangFR    Lang = "fr"
	LangDE    Lang = "de"
	LangSL    Lang = "sl"
	LangPT    Lang = "pt"
	LangZH_TW Lang = "zh-TW"
)

// AllLangs returns all supported word list languages.
func AllLangs() []Lang {
	return []Lang{LangEN, LangES, LangFR, LangDE, LangSL, LangPT, LangZH_TW}
}

// WordListInfo describes a BIP39 word list: its source, expected hash, and words.
type WordListInfo struct {
	Lang         Lang
	SourceURL    string
	ExpectedHash string // SHA-256 of the .txt file contents
	Words        [2048]string
}

// wordListSpec defines the expected properties of a word list before loading.
type wordListSpec struct {
	Lang         Lang
	File         string
	SourceURL    string
	ExpectedHash string
}

// wordListSpecs defines the source and expected hash for each language's word list.
//
// Official BIP39 lists: https://github.com/bitcoin/bips/tree/master/bip-0039
// German: https://github.com/dys2p/wordlists-de (unofficial, widely used)
// Slovenian: https://github.com/StellarStoic/BIP39_Exotica (unofficial)
var wordListSpecs = []wordListSpec{
	{LangEN, "wordlists/english.txt", "https://github.com/bitcoin/bips/blob/ed7af6ae7e80c90bcfc69b3936073505e2fc2503/bip-0039/english.txt", "2f5eed53a4727b4bf8880d8f3f199efc90e58503646d9ff8eff3a2ed3b24dbda"},
	{LangES, "wordlists/spanish.txt", "https://github.com/bitcoin/bips/blob/ed7af6ae7e80c90bcfc69b3936073505e2fc2503/bip-0039/spanish.txt", "46846a5a0139d1e3cb77293e521c2865f7bcdb82c44e8d0a06a2cd0ecba48c0b"},
	{LangFR, "wordlists/french.txt", "https://github.com/bitcoin/bips/blob/ed7af6ae7e80c90bcfc69b3936073505e2fc2503/bip-0039/french.txt", "ebc3959ab7801a1df6bac4fa7d970652f1df76b683cd2f4003c941c63d517e59"},
	{LangDE, "wordlists/german.txt", "https://github.com/dys2p/wordlists-de/blob/43553378b71ac06467e4654372ac249f15e16f4d/de-2048-v1.txt", "7965dc8c6b413ccb635d3021043365e18df0367bf5413a50a069a98addfe4e1d"},
	{LangSL, "wordlists/slovenian.txt", "https://github.com/StellarStoic/BIP39_Exotica/8a5c0d93be825fab837dd293c94c635d6a39aa70/main/WRDL/nonStandard/slovenian.txt", "bdc73f14501843be9ae38fea61d6070298df4a83c67a8710e9755c557880467a"},
	{LangPT, "wordlists/portuguese.txt", "https://github.com/bitcoin/bips/blob/ed7af6ae7e80c90bcfc69b3936073505e2fc2503/bip-0039/portuguese.txt", "2685e9c194c82ae67e10ba59d9ea5345a23dc093e92276fc5361f6667d79cd3f"},
	{LangZH_TW, "wordlists/chinese_traditional.txt", "https://github.com/bitcoin/bips/blob/7d77befd2b14359b9386fc1f9fb15f82d418fb34/bip-0039/chinese_traditional.txt", "417b26b3d8500a4ae3d59717d7011952db6fc2fb84b807f3f94ac734e89c1b5f"},
}

var (
	wordListRegistry map[Lang]*WordListInfo
	registryOnce     sync.Once
)

func initRegistry() {
	registryOnce.Do(func() {
		wordListRegistry = make(map[Lang]*WordListInfo, len(wordListSpecs))
		for _, spec := range wordListSpecs {
			data, err := wordlistFS.ReadFile(spec.File)
			if err != nil {
				panic(fmt.Sprintf("loading word list %s: %v", spec.Lang, err))
			}
			content := strings.TrimSpace(string(data))
			lines := strings.Split(content, "\n")
			if len(lines) != 2048 {
				panic(fmt.Sprintf("word list %s has %d words, expected 2048", spec.Lang, len(lines)))
			}
			info := &WordListInfo{
				Lang:         spec.Lang,
				SourceURL:    spec.SourceURL,
				ExpectedHash: spec.ExpectedHash,
			}
			for i, line := range lines {
				info.Words[i] = strings.TrimSpace(line)
			}
			wordListRegistry[spec.Lang] = info
		}
	})
}

// GetWordList returns the word list for the given language.
// Returns nil if the language is not supported.
func GetWordList(lang Lang) *WordListInfo {
	initRegistry()
	return wordListRegistry[lang]
}

// GetWordListSpecs returns the spec metadata for all word lists (for testing).
func GetWordListSpecs() []wordListSpec {
	return wordListSpecs
}

// --- Normalization ---

// NormalizeWord applies Unicode normalization for tolerant matching:
// lowercase, trim, NFD decompose, and strip combining marks.
// Examples: "Ábaco" → "abaco", "GÜNTHER" → "gunther", "čudež" → "cudez"
func NormalizeWord(word string) string {
	word = strings.ToLower(strings.TrimSpace(word))
	var stripped []rune
	for _, r := range norm.NFD.String(word) {
		if !unicode.Is(unicode.Mn, r) {
			stripped = append(stripped, r)
		}
	}
	return string(stripped)
}

// collapseGermanDigraphs converts German umlaut digraph forms to their
// base vowel, so "guenther" becomes "gunther" (matching NFD-stripped "günther").
// Also handles ß→ss in reverse (ss→s would be lossy, so we expand ß→ss instead).
func collapseGermanDigraphs(word string) string {
	r := strings.NewReplacer("ae", "a", "oe", "o", "ue", "u")
	return r.Replace(word)
}

// --- Per-language word index ---

// langWordIndex maps normalized forms to BIP39 indices for one language.
type langWordIndex struct {
	exact    map[string]int // lowercase canonical → index
	stripped map[string]int // NFD-stripped → index
	digraph  map[string]int // German digraph collapsed → index (DE only)
}

var (
	langIndices     map[Lang]*langWordIndex
	langIndicesOnce sync.Once
)

func initLangIndices() {
	langIndicesOnce.Do(func() {
		initRegistry()
		langIndices = make(map[Lang]*langWordIndex, len(wordListRegistry))
		for lang, info := range wordListRegistry {
			idx := &langWordIndex{
				exact:    make(map[string]int, 2048),
				stripped: make(map[string]int, 2048),
			}
			if lang == LangDE {
				idx.digraph = make(map[string]int, 2048)
			}
			for i, w := range info.Words {
				lower := strings.ToLower(w)
				idx.exact[lower] = i

				normalized := NormalizeWord(w)
				// Only store stripped form if it differs from exact
				// (avoids redundant lookups for ASCII-only lists like English)
				if normalized != lower {
					idx.stripped[normalized] = i
				}

				if lang == LangDE {
					collapsed := collapseGermanDigraphs(normalized)
					if collapsed != normalized {
						idx.digraph[collapsed] = i
					}
				}
			}
			langIndices[lang] = idx
		}
	})
}

// LookupWord finds a word's BIP39 index in the given language.
// Tries exact match first, then NFD-stripped, then German digraph expansion.
// Returns (index, true) if found, (0, false) if not.
func LookupWord(lang Lang, word string) (int, bool) {
	initLangIndices()
	idx := langIndices[lang]
	if idx == nil {
		return 0, false
	}

	lower := strings.ToLower(strings.TrimSpace(word))

	// 1. Exact lowercase match
	if i, ok := idx.exact[lower]; ok {
		return i, true
	}

	// 2. NFD-stripped match (ábaco → abaco)
	normalized := NormalizeWord(word)
	if i, ok := idx.stripped[normalized]; ok {
		return i, true
	}

	// 3. German digraph collapse (guenther → gunther, matching günther)
	if lang == LangDE && idx.digraph != nil {
		collapsed := collapseGermanDigraphs(normalized)
		if i, ok := idx.digraph[collapsed]; ok {
			return i, true
		}
	}

	return 0, false
}

// --- Language detection ---

// DetectWordListLang identifies which language a set of words belongs to.
// Returns the language where the most words match. Requires >50% match.
// Returns empty string if no language matches.
func DetectWordListLang(words []string) Lang {
	initLangIndices()
	bestLang := Lang("")
	bestCount := 0
	for _, lang := range AllLangs() {
		count := 0
		for _, w := range words {
			if _, ok := LookupWord(lang, w); ok {
				count++
			}
		}
		if count > bestCount {
			bestCount = count
			bestLang = lang
		}
	}
	if bestCount <= len(words)/2 {
		return ""
	}
	return bestLang
}

// --- Hash verification (used by tests) ---

// WordListHash computes the SHA-256 hash of a word list's canonical form
// (words joined by newline, with trailing newline).
func WordListHash(lang Lang) string {
	initRegistry()
	info := wordListRegistry[lang]
	if info == nil {
		return ""
	}
	joined := strings.Join(info.Words[:], "\n") + "\n"
	h := sha256.Sum256([]byte(joined))
	return hex.EncodeToString(h[:])
}
