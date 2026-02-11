package core

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

var generate = flag.Bool("generate", false, "regenerate golden test fixtures (writes to testdata/)")

// --- JSON fixture types ---

type goldenFixture struct {
	Version    int            `json:"version"`
	Passphrase string         `json:"passphrase"`
	Total      int            `json:"total"`
	Threshold  int            `json:"threshold"`
	Created    string         `json:"created"`
	Shares     []goldenShare  `json:"shares"`
	Manifest   goldenManifest `json:"manifest"`
}

type goldenShare struct {
	Index    int    `json:"index"`
	Holder   string `json:"holder"`
	DataHex  string `json:"data_hex"`
	Checksum string `json:"checksum"`
	PEM      string `json:"pem"`
	Compact  string `json:"compact"`
	Words    string `json:"words,omitempty"` // space-separated BIP39 words (v2 only)
}

type goldenManifest struct {
	Files map[string]string `json:"files"`
}

// --- Constants for golden fixtures ---

const (
	// goldenPassphrase is a fixed base64url string (43 chars, represents 32 bytes).
	// This mimics the output of crypto.GeneratePassphrase(32) but is deterministic.
	// Decodes to "this_is_a_test_passphrase_v2_gld" (exactly 32 bytes).
	// Shamir adds a 1-byte coordinate to each share, so 32-byte passphrase → 33-byte shares.
	// 33 bytes = 264 bits → exactly 24 BIP39 words (11 bits each) + 1 index word = 25 words.
	goldenPassphrase = "dGhpc19pc19hX3Rlc3RfcGFzc3BocmFzZV92Ml9nbGQ"

	// goldenCreated is the fixed timestamp for all golden shares.
	goldenCreated = "2025-01-01 00:00"

	// goldenCreatedFormat is the Go time format for parsing goldenCreated.
	goldenCreatedFormat = "2006-01-02 15:04"
)

var goldenHolders = []string{"Alice", "Bob", "Carol", "David", "Eve"}

// goldenManifestFiles are the known files inside the golden test manifest.
var goldenManifestFiles = map[string]string{
	"manifest/README.md":  "# Golden Test Manifest\n\nThis is a test manifest for v1 golden fixtures.\n",
	"manifest/secret.txt": "The secret passphrase is: correct-horse-battery-staple\n",
}

// --- Helpers ---

func loadGoldenJSON(t *testing.T, filename string) goldenFixture {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", filename))
	if err != nil {
		t.Fatalf("reading %s: %v", filename, err)
	}
	var golden goldenFixture
	if err := json.Unmarshal(data, &golden); err != nil {
		t.Fatalf("unmarshaling %s: %v", filename, err)
	}
	return golden
}

func mustDecodeHex(t *testing.T, s string) []byte {
	t.Helper()
	data, err := hex.DecodeString(s)
	if err != nil {
		t.Fatalf("decoding hex: %v", err)
	}
	return data
}

// parseCreatedTime parses a created timestamp, trying short format first then RFC3339.
func parseCreatedTime(t *testing.T, s string) time.Time {
	t.Helper()
	if parsed, err := time.Parse("2006-01-02 15:04", s); err == nil {
		return parsed
	}
	if parsed, err := time.Parse(time.RFC3339, s); err == nil {
		return parsed
	}
	t.Fatalf("cannot parse created time %q", s)
	return time.Time{}
}

// combinations returns all k-element subsets of {0, 1, ..., n-1}.
func combinations(n, k int) [][]int {
	var result [][]int
	combo := make([]int, k)
	var gen func(start, depth int)
	gen = func(start, depth int) {
		if depth == k {
			dup := make([]int, k)
			copy(dup, combo)
			result = append(result, dup)
			return
		}
		for i := start; i < n; i++ {
			combo[depth] = i
			gen(i+1, depth+1)
		}
	}
	gen(0, 0)
	return result
}

// --- Generator ---

// TestGenerateGoldenFixtures generates v2 golden test fixtures.
// V1 fixtures are immutable and checked into the repo — this generator never touches them.
// Run once with: go test -v -run TestGenerateGoldenFixtures -generate ./internal/core/
func TestGenerateGoldenFixtures(t *testing.T) {
	if !*generate {
		t.Skip("skipping fixture generation (use -generate flag to regenerate)")
	}

	createdTime, err := time.Parse(goldenCreatedFormat, goldenCreated)
	if err != nil {
		t.Fatalf("parsing created time: %v", err)
	}

	// Build manifest archive (shared by both v1 and v2)
	archiveData := createTarGz(t, goldenManifestFiles)

	// Decode the passphrase to get the raw 32 bytes (v2 splits these directly)
	rawPassphrase, err := base64.RawURLEncoding.DecodeString(goldenPassphrase)
	if err != nil {
		t.Fatalf("decoding goldenPassphrase: %v", err)
	}

	// Split the raw bytes (32 bytes → 33-byte shares)
	rawSharesV2, err := Split(rawPassphrase, 5, 3)
	if err != nil {
		t.Fatalf("splitting raw passphrase: %v", err)
	}

	// Verify reconstruction: combine → base64url-encode → should match passphrase
	recoveredV2, err := Combine(rawSharesV2[:3])
	if err != nil {
		t.Fatalf("combining v2 shares: %v", err)
	}
	if RecoverPassphrase(recoveredV2, 2) != goldenPassphrase {
		t.Fatal("v2 share reconstruction failed — shares are broken")
	}

	// Build v2 shares with fixed metadata
	sharesV2 := make([]*Share, 5)
	goldenSharesV2 := make([]goldenShare, 5)
	for i := 0; i < 5; i++ {
		share := &Share{
			Version:   2,
			Index:     i + 1,
			Total:     5,
			Threshold: 3,
			Holder:    goldenHolders[i],
			Created:   createdTime,
			Data:      rawSharesV2[i],
			Checksum:  HashBytes(rawSharesV2[i]),
		}
		sharesV2[i] = share
		goldenSharesV2[i] = goldenShare{
			Index:    share.Index,
			Holder:   share.Holder,
			DataHex:  hex.EncodeToString(share.Data),
			Checksum: share.Checksum,
			PEM:      share.Encode(),
			Compact:  share.CompactEncode(),
			Words:    func() string { w, _ := share.Words(); return strings.Join(w, " ") }(),
		}
	}

	// Encrypt manifest for v2
	var encryptedBufV2 bytes.Buffer
	if err := Encrypt(&encryptedBufV2, bytes.NewReader(archiveData), goldenPassphrase); err != nil {
		t.Fatalf("encrypting v2 manifest: %v", err)
	}

	// Build v2 fixture JSON
	fixtureV2 := goldenFixture{
		Version:    2,
		Passphrase: goldenPassphrase,
		Total:      5,
		Threshold:  3,
		Created:    goldenCreated,
		Shares:     goldenSharesV2,
		Manifest: goldenManifest{
			Files: goldenManifestFiles,
		},
	}

	fixtureJSONV2, err := json.MarshalIndent(fixtureV2, "", "  ")
	if err != nil {
		t.Fatalf("marshaling v2 fixture JSON: %v", err)
	}

	// Create v2 directories
	bundleDirV2 := filepath.Join("testdata", "v2-bundle")
	expectedDirV2 := filepath.Join(bundleDirV2, "expected-output")
	if err := os.MkdirAll(expectedDirV2, 0755); err != nil {
		t.Fatalf("creating v2 directories: %v", err)
	}

	// Write v2-golden.json
	jsonPathV2 := filepath.Join("testdata", "v2-golden.json")
	if err := os.WriteFile(jsonPathV2, fixtureJSONV2, 0644); err != nil {
		t.Fatalf("writing %s: %v", jsonPathV2, err)
	}
	t.Logf("wrote %s", jsonPathV2)

	// Write v2 share PEM files
	for _, share := range sharesV2 {
		filename := fmt.Sprintf("SHARE-%s.txt", strings.ToLower(share.Holder))
		sharePath := filepath.Join(bundleDirV2, filename)
		if err := os.WriteFile(sharePath, []byte(share.Encode()), 0644); err != nil {
			t.Fatalf("writing %s: %v", sharePath, err)
		}
		t.Logf("wrote %s", sharePath)
	}

	// Write v2 MANIFEST.age
	manifestPathV2 := filepath.Join(bundleDirV2, "MANIFEST.age")
	if err := os.WriteFile(manifestPathV2, encryptedBufV2.Bytes(), 0644); err != nil {
		t.Fatalf("writing %s: %v", manifestPathV2, err)
	}
	t.Logf("wrote %s (%d bytes)", manifestPathV2, encryptedBufV2.Len())

	// Write v2 expected output files
	for name, content := range goldenManifestFiles {
		outPath := filepath.Join(expectedDirV2, name)
		if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
			t.Fatalf("creating dir for %s: %v", outPath, err)
		}
		if err := os.WriteFile(outPath, []byte(content), 0644); err != nil {
			t.Fatalf("writing %s: %v", outPath, err)
		}
		t.Logf("wrote %s", outPath)
	}

	t.Log("V2 golden fixtures generated successfully.")
	t.Log("Commit the testdata/v2-* files. V1 fixtures are immutable and must not be regenerated.")
}

// --- Golden tests (table-driven across v1 and v2) ---

// goldenVersion defines a fixture version for table-driven golden tests.
type goldenVersion struct {
	name      string // "v1" or "v2"
	fixture   string // JSON fixture filename
	bundleDir string // testdata subdirectory with share PEM files and MANIFEST.age
}

var goldenVersions = []goldenVersion{
	{"v1", "v1-golden.json", "v1-bundle"},
	{"v2", "v2-golden.json", "v2-bundle"},
}

// TestGoldenShareParsing parses each fixture share and verifies all fields match.
func TestGoldenShareParsing(t *testing.T) {
	for _, ver := range goldenVersions {
		t.Run(ver.name, func(t *testing.T) {
			golden := loadGoldenJSON(t, ver.fixture)

			for _, gs := range golden.Shares {
				t.Run(gs.Holder, func(t *testing.T) {
					filename := fmt.Sprintf("SHARE-%s.txt", strings.ToLower(gs.Holder))
					pemData, err := os.ReadFile(filepath.Join("testdata", ver.bundleDir, filename))
					if err != nil {
						t.Fatalf("reading %s: %v", filename, err)
					}

					share, err := ParseShare(pemData)
					if err != nil {
						t.Fatalf("ParseShare: %v", err)
					}

					if share.Version != golden.Version {
						t.Errorf("version: got %d, want %d", share.Version, golden.Version)
					}
					if share.Index != gs.Index {
						t.Errorf("index: got %d, want %d", share.Index, gs.Index)
					}
					if share.Total != golden.Total {
						t.Errorf("total: got %d, want %d", share.Total, golden.Total)
					}
					if share.Threshold != golden.Threshold {
						t.Errorf("threshold: got %d, want %d", share.Threshold, golden.Threshold)
					}
					if share.Holder != gs.Holder {
						t.Errorf("holder: got %q, want %q", share.Holder, gs.Holder)
					}

					expectedCreated := parseCreatedTime(t, golden.Created)
					if !share.Created.Equal(expectedCreated) {
						t.Errorf("created: got %v, want %v", share.Created, expectedCreated)
					}

					expectedData := mustDecodeHex(t, gs.DataHex)
					if !bytes.Equal(share.Data, expectedData) {
						t.Errorf("data mismatch: got %x, want %s", share.Data, gs.DataHex)
					}

					if share.Checksum != gs.Checksum {
						t.Errorf("checksum: got %q, want %q", share.Checksum, gs.Checksum)
					}

					if err := share.Verify(); err != nil {
						t.Errorf("Verify: %v", err)
					}

					reEncoded := share.Encode()
					if reEncoded != gs.PEM {
						t.Errorf("PEM re-encode mismatch:\ngot:\n%s\nwant:\n%s", reEncoded, gs.PEM)
					}

					compact := share.CompactEncode()
					if compact != gs.Compact {
						t.Errorf("compact: got %q, want %q", compact, gs.Compact)
					}

					decoded, err := ParseCompact(compact)
					if err != nil {
						t.Fatalf("ParseCompact: %v", err)
					}
					if !bytes.Equal(decoded.Data, share.Data) {
						t.Errorf("compact round-trip data mismatch")
					}
					if decoded.Version != share.Version {
						t.Errorf("compact round-trip version: got %d, want %d", decoded.Version, share.Version)
					}
				})
			}
		})
	}
}

// TestGoldenCombine combines threshold shares and verifies the passphrase.
func TestGoldenCombine(t *testing.T) {
	for _, ver := range goldenVersions {
		t.Run(ver.name, func(t *testing.T) {
			golden := loadGoldenJSON(t, ver.fixture)

			if len(golden.Shares) < golden.Threshold {
				t.Fatalf("not enough shares in fixture: have %d, need %d", len(golden.Shares), golden.Threshold)
			}

			shareData := make([][]byte, golden.Threshold)
			for i := 0; i < golden.Threshold; i++ {
				shareData[i] = mustDecodeHex(t, golden.Shares[i].DataHex)
			}

			recovered, err := Combine(shareData)
			if err != nil {
				t.Fatalf("Combine: %v", err)
			}

			passphrase := RecoverPassphrase(recovered, golden.Version)
			if passphrase != golden.Passphrase {
				t.Errorf("passphrase: got %q, want %q", passphrase, golden.Passphrase)
			}
		})
	}
}

// TestGoldenCombineAllSubsets tries all valid k-of-n subsets.
func TestGoldenCombineAllSubsets(t *testing.T) {
	for _, ver := range goldenVersions {
		t.Run(ver.name, func(t *testing.T) {
			golden := loadGoldenJSON(t, ver.fixture)

			allData := make([][]byte, len(golden.Shares))
			for i, gs := range golden.Shares {
				allData[i] = mustDecodeHex(t, gs.DataHex)
			}

			subsets := combinations(len(golden.Shares), golden.Threshold)
			expectedSubsets := 10 // C(5,3) = 10
			if len(subsets) != expectedSubsets {
				t.Fatalf("expected %d subsets, got %d", expectedSubsets, len(subsets))
			}

			for _, subset := range subsets {
				indices := make([]string, len(subset))
				for i, idx := range subset {
					indices[i] = fmt.Sprintf("%d", golden.Shares[idx].Index)
				}
				name := strings.Join(indices, ",")

				t.Run(name, func(t *testing.T) {
					shareData := make([][]byte, len(subset))
					for i, idx := range subset {
						shareData[i] = allData[idx]
					}

					recovered, err := Combine(shareData)
					if err != nil {
						t.Fatalf("Combine: %v", err)
					}

					passphrase := RecoverPassphrase(recovered, golden.Version)
					if passphrase != golden.Passphrase {
						t.Errorf("passphrase: got %q, want %q", passphrase, golden.Passphrase)
					}
				})
			}
		})
	}
}

// TestGoldenCombineBelowThreshold verifies that combining fewer than threshold
// shares does not recover the passphrase (Shamir's information-theoretic security).
func TestGoldenCombineBelowThreshold(t *testing.T) {
	for _, ver := range goldenVersions {
		t.Run(ver.name, func(t *testing.T) {
			golden := loadGoldenJSON(t, ver.fixture)

			allData := make([][]byte, len(golden.Shares))
			for i, gs := range golden.Shares {
				allData[i] = mustDecodeHex(t, gs.DataHex)
			}

			for size := 1; size < golden.Threshold; size++ {
				subsets := combinations(len(golden.Shares), size)
				for _, subset := range subsets {
					indices := make([]string, len(subset))
					for i, idx := range subset {
						indices[i] = fmt.Sprintf("%d", golden.Shares[idx].Index)
					}
					name := fmt.Sprintf("%d-of-%d[%s]", size, golden.Threshold, strings.Join(indices, ","))

					t.Run(name, func(t *testing.T) {
						shareData := make([][]byte, len(subset))
						for i, idx := range subset {
							shareData[i] = allData[idx]
						}

						recovered, err := Combine(shareData)
						if err != nil {
							// Combine rejecting below-threshold input is also acceptable
							return
						}

						passphrase := RecoverPassphrase(recovered, golden.Version)
						if passphrase == golden.Passphrase {
							t.Errorf("below-threshold subset recovered the passphrase")
						}
					})
				}
			}
		})
	}
}

// TestGoldenDecrypt combines shares, decrypts the manifest, and verifies output.
func TestGoldenDecrypt(t *testing.T) {
	for _, ver := range goldenVersions {
		t.Run(ver.name, func(t *testing.T) {
			golden := loadGoldenJSON(t, ver.fixture)

			shareNames := []string{"alice", "bob", "carol"}
			shareData := make([][]byte, len(shareNames))
			for i, name := range shareNames {
				filename := fmt.Sprintf("SHARE-%s.txt", name)
				pemData, err := os.ReadFile(filepath.Join("testdata", ver.bundleDir, filename))
				if err != nil {
					t.Fatalf("reading %s: %v", filename, err)
				}

				share, err := ParseShare(pemData)
				if err != nil {
					t.Fatalf("ParseShare(%s): %v", filename, err)
				}

				if err := share.Verify(); err != nil {
					t.Fatalf("Verify(%s): %v", filename, err)
				}

				shareData[i] = share.Data
			}

			recovered, err := Combine(shareData)
			if err != nil {
				t.Fatalf("Combine: %v", err)
			}

			passphrase := RecoverPassphrase(recovered, golden.Version)
			if passphrase != golden.Passphrase {
				t.Fatalf("passphrase mismatch: got %q, want %q", passphrase, golden.Passphrase)
			}

			manifestAge, err := os.ReadFile(filepath.Join("testdata", ver.bundleDir, "MANIFEST.age"))
			if err != nil {
				t.Fatalf("reading MANIFEST.age: %v", err)
			}

			var decrypted bytes.Buffer
			if err := Decrypt(&decrypted, bytes.NewReader(manifestAge), passphrase); err != nil {
				t.Fatalf("Decrypt: %v", err)
			}

			files, err := ExtractTarGz(decrypted.Bytes())
			if err != nil {
				t.Fatalf("ExtractTarGz: %v", err)
			}

			if len(files) == 0 {
				t.Fatal("no files extracted from manifest")
			}

			extracted := make(map[string]string)
			for _, f := range files {
				extracted[f.Name] = string(f.Data)
			}

			if len(extracted) != len(golden.Manifest.Files) {
				t.Errorf("file count mismatch: extracted %d, expected %d", len(extracted), len(golden.Manifest.Files))
			}

			for name, expectedContent := range golden.Manifest.Files {
				got, ok := extracted[name]
				if !ok {
					t.Errorf("missing extracted file %q", name)
					continue
				}
				if got != expectedContent {
					t.Errorf("file %q: got %q, want %q", name, got, expectedContent)
				}
			}

			for _, f := range files {
				diskPath := filepath.Join("testdata", ver.bundleDir, "expected-output", f.Name)
				diskContent, err := os.ReadFile(diskPath)
				if err != nil {
					t.Errorf("reading expected output %s: %v", diskPath, err)
					continue
				}
				if string(f.Data) != string(diskContent) {
					t.Errorf("file %q doesn't match expected-output on disk", f.Name)
				}
			}
		})
	}
}

// TestGoldenV2WordEncoding tests word encoding round-trips against golden fixtures.
// Words are 25 words: 24 data words + 1 index word.
func TestGoldenV2WordEncoding(t *testing.T) {
	golden := loadGoldenJSON(t, "v2-golden.json")

	for _, gs := range golden.Shares {
		t.Run(gs.Holder, func(t *testing.T) {
			if gs.Words == "" {
				t.Skip("no words field in fixture (regenerate with -generate)")
			}

			data := mustDecodeHex(t, gs.DataHex)

			// Build a share to get the full 25-word encoding
			share := &Share{
				Version: golden.Version,
				Index:   gs.Index,
				Data:    data,
			}
			words, err := share.Words()
			if err != nil {
				t.Fatalf("Words() error: %v", err)
			}
			got := strings.Join(words, " ")
			if got != gs.Words {
				t.Errorf("word encoding mismatch:\n  got:  %s\n  want: %s", got, gs.Words)
			}

			// Verify 25 words (24 data + 1 index)
			fixtureWords := strings.Split(gs.Words, " ")
			if len(fixtureWords) != 25 {
				t.Fatalf("expected 25 words in fixture, got %d", len(fixtureWords))
			}

			// Decode fixture words and compare to data bytes + index
			decoded, index, err := DecodeShareWords(fixtureWords)
			if err != nil {
				t.Fatalf("DecodeShareWords: %v", err)
			}
			if !bytes.Equal(decoded, data) {
				t.Errorf("word decoding data mismatch:\n  got:  %x\n  want: %s", decoded, gs.DataHex)
			}
			if index != gs.Index {
				t.Errorf("word decoding index: got %d, want %d", index, gs.Index)
			}
		})
	}
}
