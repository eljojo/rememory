package integration_test

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/eljojo/rememory/internal/bundle"
	"github.com/eljojo/rememory/internal/core"
	"github.com/eljojo/rememory/internal/crypto"
	"github.com/eljojo/rememory/internal/manifest"
	"github.com/eljojo/rememory/internal/project"
)

// TestFullWorkflow tests the complete init -> seal -> recover pipeline
func TestFullWorkflow(t *testing.T) {
	// Setup: create a temp directory for the project
	baseDir := t.TempDir()
	projectDir := filepath.Join(baseDir, "test-project")

	// Step 1: Create project (simulating 'rememory init')
	friends := []project.Friend{
		{Name: "Alice", Contact: "alice@example.com"},
		{Name: "Bob", Contact: "bob@example.com"},
		{Name: "Carol", Contact: "carol@example.com"},
		{Name: "David", Contact: "david@example.com"},
		{Name: "Eve", Contact: "eve@example.com"},
	}
	threshold := 3

	p, err := project.New(projectDir, "test-project", threshold, friends)
	if err != nil {
		t.Fatalf("creating project: %v", err)
	}

	// Add secret content to manifest
	secretContent := "This is my super secret password: hunter2"
	secretFile := filepath.Join(p.ManifestPath(), "secrets.txt")
	if err := os.WriteFile(secretFile, []byte(secretContent), 0644); err != nil {
		t.Fatalf("writing secret: %v", err)
	}

	// Step 2: Seal (simulating 'rememory seal')
	// Archive manifest
	var archiveBuf bytes.Buffer
	if _, err := manifest.Archive(&archiveBuf, p.ManifestPath()); err != nil {
		t.Fatalf("archiving: %v", err)
	}

	// Generate passphrase
	passphrase, err := crypto.GeneratePassphrase(crypto.DefaultPassphraseBytes)
	if err != nil {
		t.Fatalf("generating passphrase: %v", err)
	}

	// Encrypt
	var encryptedBuf bytes.Buffer
	if err := core.Encrypt(&encryptedBuf, bytes.NewReader(archiveBuf.Bytes()), passphrase); err != nil {
		t.Fatalf("encrypting: %v", err)
	}

	// Split passphrase
	shares, err := core.Split([]byte(passphrase), len(friends), threshold)
	if err != nil {
		t.Fatalf("splitting: %v", err)
	}

	// Create share objects with metadata
	shareObjects := make([]*core.Share, len(shares))
	for i, data := range shares {
		shareObjects[i] = core.NewShare(i+1, len(friends), threshold, friends[i].Name, data)
	}

	// Verify immediate reconstruction
	testShares := make([][]byte, threshold)
	for i := 0; i < threshold; i++ {
		testShares[i] = shares[i]
	}
	recovered, err := core.Combine(testShares)
	if err != nil {
		t.Fatalf("verification combine: %v", err)
	}
	if string(recovered) != passphrase {
		t.Fatal("verification failed: passphrase mismatch")
	}

	// Step 3: Simulate share distribution and recovery
	// Test with different share combinations
	combinations := [][]int{
		{0, 1, 2},       // First three
		{2, 3, 4},       // Last three
		{0, 2, 4},       // Alternating
		{1, 2, 3},       // Middle three
		{0, 1, 2, 3, 4}, // All five (should also work)
	}

	for _, combo := range combinations {
		t.Run("", func(t *testing.T) {
			// Simulate encoding/decoding shares (as if read from files)
			recoveryShares := make([]*core.Share, len(combo))
			for i, idx := range combo {
				encoded := shareObjects[idx].Encode()
				parsed, err := core.ParseShare([]byte(encoded))
				if err != nil {
					t.Fatalf("parsing share %d: %v", idx, err)
				}
				if err := parsed.Verify(); err != nil {
					t.Fatalf("verifying share %d: %v", idx, err)
				}
				recoveryShares[i] = parsed
			}

			// Combine shares
			shareData := make([][]byte, len(recoveryShares))
			for i, s := range recoveryShares {
				shareData[i] = s.Data
			}
			recoveredPass, err := core.Combine(shareData)
			if err != nil {
				t.Fatalf("combining: %v", err)
			}

			if string(recoveredPass) != passphrase {
				t.Fatal("recovered passphrase doesn't match")
			}

			// Decrypt
			var decryptedBuf bytes.Buffer
			if err := core.Decrypt(&decryptedBuf, bytes.NewReader(encryptedBuf.Bytes()), string(recoveredPass)); err != nil {
				t.Fatalf("decrypting: %v", err)
			}

			// Extract
			extractDir := t.TempDir()
			extractResult, err := manifest.Extract(&decryptedBuf, extractDir)
			if err != nil {
				t.Fatalf("extracting: %v", err)
			}

			// Verify content
			recoveredSecret, err := os.ReadFile(filepath.Join(extractResult.Path, "secrets.txt"))
			if err != nil {
				t.Fatalf("reading recovered secret: %v", err)
			}
			if string(recoveredSecret) != secretContent {
				t.Errorf("content mismatch: got %q, want %q", recoveredSecret, secretContent)
			}
		})
	}
}

// TestInsufficientShares verifies that fewer than threshold shares fail
func TestInsufficientShares(t *testing.T) {
	passphrase := "test-passphrase"
	n, k := 5, 3

	shares, err := core.Split([]byte(passphrase), n, k)
	if err != nil {
		t.Fatal(err)
	}

	// Try with k-1 shares - should produce wrong result
	insufficient := shares[:k-1]
	recovered, err := core.Combine(insufficient)
	if err != nil {
		// Some implementations error, which is fine
		return
	}

	// If no error, the result should be wrong
	if string(recovered) == passphrase {
		t.Error("recovered correct passphrase with insufficient shares - this shouldn't happen")
	}
}

// TestCorruptedShare verifies that corrupted shares are detected
func TestCorruptedShare(t *testing.T) {
	share := core.NewShare(1, 5, 3, "Alice", []byte("test-data"))
	encoded := share.Encode()

	// Parse and verify - should work
	parsed, err := core.ParseShare([]byte(encoded))
	if err != nil {
		t.Fatal(err)
	}
	if err := parsed.Verify(); err != nil {
		t.Fatal(err)
	}

	// Corrupt the data
	parsed.Data[0] ^= 0xFF

	// Verify should now fail
	if err := parsed.Verify(); err == nil {
		t.Error("corrupted share should fail verification")
	}
}

// TestWrongPassphrase verifies decryption fails with wrong passphrase
func TestWrongPassphrase(t *testing.T) {
	data := []byte("secret data")
	correctPass := "correct-passphrase"
	wrongPass := "wrong-passphrase"

	var encrypted bytes.Buffer
	if err := core.Encrypt(&encrypted, bytes.NewReader(data), correctPass); err != nil {
		t.Fatal(err)
	}

	var decrypted bytes.Buffer
	err := core.Decrypt(&decrypted, bytes.NewReader(encrypted.Bytes()), wrongPass)
	if err == nil {
		t.Error("decryption should fail with wrong passphrase")
	}
}

// TestLargeManifest tests with a larger payload (approaching 10MB limit)
func TestLargeManifest(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping large manifest test in short mode")
	}

	baseDir := t.TempDir()
	manifestDir := filepath.Join(baseDir, "manifest")
	if err := os.MkdirAll(manifestDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create a ~1MB file
	largeData := bytes.Repeat([]byte("x"), 1024*1024)
	if err := os.WriteFile(filepath.Join(manifestDir, "large.bin"), largeData, 0644); err != nil {
		t.Fatal(err)
	}

	// Archive
	var archiveBuf bytes.Buffer
	if _, err := manifest.Archive(&archiveBuf, manifestDir); err != nil {
		t.Fatal(err)
	}

	// Encrypt
	passphrase := "test-passphrase"
	var encrypted bytes.Buffer
	if err := core.Encrypt(&encrypted, &archiveBuf, passphrase); err != nil {
		t.Fatal(err)
	}

	// Decrypt
	var decrypted bytes.Buffer
	if err := core.Decrypt(&decrypted, &encrypted, passphrase); err != nil {
		t.Fatal(err)
	}

	// Extract and verify
	extractDir := t.TempDir()
	extractResult, err := manifest.Extract(&decrypted, extractDir)
	if err != nil {
		t.Fatal(err)
	}

	recovered, err := os.ReadFile(filepath.Join(extractResult.Path, "large.bin"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(recovered, largeData) {
		t.Error("large file content mismatch")
	}
}

// TestAllThresholdCombinations tests all valid (N,K) from 2-of-2 to 7-of-7
func TestAllThresholdCombinations(t *testing.T) {
	secret := []byte("test-secret-for-threshold-combinations")

	for n := 2; n <= 7; n++ {
		for k := 2; k <= n; k++ {
			t.Run("", func(t *testing.T) {
				// Split
				shares, err := core.Split(secret, n, k)
				if err != nil {
					t.Fatalf("%d-of-%d split: %v", k, n, err)
				}

				// Create share objects and encode/decode them
				shareObjs := make([]*core.Share, n)
				for i, data := range shares {
					shareObjs[i] = core.NewShare(i+1, n, k, "", data)
					encoded := shareObjs[i].Encode()
					parsed, err := core.ParseShare([]byte(encoded))
					if err != nil {
						t.Fatalf("parse share %d: %v", i, err)
					}
					if err := parsed.Verify(); err != nil {
						t.Fatalf("verify share %d: %v", i, err)
					}
					shareObjs[i] = parsed
				}

				// Recover with exactly k shares
				recoverData := make([][]byte, k)
				for i := 0; i < k; i++ {
					recoverData[i] = shareObjs[i].Data
				}
				recovered, err := core.Combine(recoverData)
				if err != nil {
					t.Fatalf("%d-of-%d combine: %v", k, n, err)
				}
				if string(recovered) != string(secret) {
					t.Errorf("%d-of-%d: secret mismatch", k, n)
				}
			})
		}
	}
}

// TestBundleGeneration tests the complete bundle generation workflow
func TestBundleGeneration(t *testing.T) {
	// Setup: create a sealed project
	baseDir := t.TempDir()
	projectDir := filepath.Join(baseDir, "test-bundle-project")

	friends := []project.Friend{
		{Name: "Alice", Contact: "alice@example.com"},
		{Name: "Bob", Contact: "bob@example.com"},
		{Name: "Carol", Contact: "carol@example.com"},
	}
	threshold := 2

	p, err := project.New(projectDir, "test-bundle-project", threshold, friends)
	if err != nil {
		t.Fatalf("creating project: %v", err)
	}

	// Add secret content
	secretContent := "My super secret: treasure is under the oak tree"
	secretFile := filepath.Join(p.ManifestPath(), "secrets.txt")
	if err := os.WriteFile(secretFile, []byte(secretContent), 0644); err != nil {
		t.Fatalf("writing secret: %v", err)
	}

	// Seal the project
	var archiveBuf bytes.Buffer
	if _, err := manifest.Archive(&archiveBuf, p.ManifestPath()); err != nil {
		t.Fatalf("archiving: %v", err)
	}

	passphrase, err := crypto.GeneratePassphrase(crypto.DefaultPassphraseBytes)
	if err != nil {
		t.Fatalf("generating passphrase: %v", err)
	}

	// Create output directories
	if err := os.MkdirAll(p.OutputPath(), 0755); err != nil {
		t.Fatalf("creating output dir: %v", err)
	}
	if err := os.MkdirAll(p.SharesPath(), 0755); err != nil {
		t.Fatalf("creating shares dir: %v", err)
	}

	// Encrypt manifest
	manifestFile, err := os.Create(p.ManifestAgePath())
	if err != nil {
		t.Fatalf("creating manifest file: %v", err)
	}
	if err := core.Encrypt(manifestFile, bytes.NewReader(archiveBuf.Bytes()), passphrase); err != nil {
		manifestFile.Close()
		t.Fatalf("encrypting: %v", err)
	}
	manifestFile.Close()

	// Split passphrase and write shares
	shares, err := core.Split([]byte(passphrase), len(friends), threshold)
	if err != nil {
		t.Fatalf("splitting: %v", err)
	}

	shareInfos := make([]project.ShareInfo, len(friends))
	for i, data := range shares {
		share := core.NewShare(i+1, len(friends), threshold, friends[i].Name, data)
		sharePath := filepath.Join(p.SharesPath(), share.Filename())
		if err := os.WriteFile(sharePath, []byte(share.Encode()), 0644); err != nil {
			t.Fatalf("writing share: %v", err)
		}
		shareInfos[i] = project.ShareInfo{
			Friend:   friends[i].Name,
			File:     share.Filename(),
			Checksum: share.Checksum,
		}
	}

	// Read manifest for checksum
	manifestData, _ := os.ReadFile(p.ManifestAgePath())
	manifestChecksum := core.HashBytes(manifestData)

	// Mark project as sealed
	p.Sealed = &project.Sealed{
		At:               time.Now(),
		ManifestChecksum: manifestChecksum,
		VerificationHash: core.HashString(passphrase),
		Shares:           shareInfos,
	}
	if err := p.Save(); err != nil {
		t.Fatalf("saving project: %v", err)
	}

	// Generate bundles
	// Use minimal WASM bytes for testing (just needs to be non-empty)
	fakeWASM := []byte("fake-wasm-for-testing")

	cfg := bundle.Config{
		Version:          "v1.0.0-test",
		GitHubReleaseURL: "https://github.com/eljojo/rememory/releases/tag/v1.0.0-test",
		WASMBytes:        fakeWASM,
	}

	if err := bundle.GenerateAll(p, cfg); err != nil {
		t.Fatalf("generating bundles: %v", err)
	}

	// Verify bundles were created
	bundlesDir := filepath.Join(p.OutputPath(), "bundles")
	entries, err := os.ReadDir(bundlesDir)
	if err != nil {
		t.Fatalf("reading bundles dir: %v", err)
	}

	if len(entries) != len(friends) {
		t.Errorf("expected %d bundles, got %d", len(friends), len(entries))
	}

	// Verify each bundle
	for _, friend := range friends {
		bundlePath := filepath.Join(bundlesDir, "bundle-"+strings.ToLower(friend.Name)+".zip")
		t.Run("Bundle-"+friend.Name, func(t *testing.T) {
			verifyBundle(t, bundlePath, friend, friends, threshold)
		})
	}
}

func verifyBundle(t *testing.T, bundlePath string, friend project.Friend, allFriends []project.Friend, threshold int) {
	t.Helper()

	// Open ZIP
	r, err := zip.OpenReader(bundlePath)
	if err != nil {
		t.Fatalf("opening bundle: %v", err)
	}
	defer r.Close()

	// Check expected files exist
	expectedFiles := map[string]bool{
		"README.txt":   false,
		"README.pdf":   false,
		"MANIFEST.age": false,
		"recover.html": false,
	}

	var readmeContent string
	var recoverContent string

	for _, f := range r.File {
		if _, ok := expectedFiles[f.Name]; ok {
			expectedFiles[f.Name] = true
		}

		rc, err := f.Open()
		if err != nil {
			t.Fatalf("opening %s: %v", f.Name, err)
		}
		data, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			t.Fatalf("reading %s: %v", f.Name, err)
		}

		switch f.Name {
		case "README.txt":
			readmeContent = string(data)
		case "recover.html":
			recoverContent = string(data)
		}
	}

	for name, found := range expectedFiles {
		if !found {
			t.Errorf("missing file: %s", name)
		}
	}

	// Verify README.txt contains the share
	if !strings.Contains(readmeContent, "-----BEGIN REMEMORY SHARE-----") {
		t.Error("README.txt missing share block")
	}
	if !strings.Contains(readmeContent, "-----END REMEMORY SHARE-----") {
		t.Error("README.txt missing share end block")
	}

	// Verify share can be parsed from README
	share, err := core.ParseShare([]byte(readmeContent))
	if err != nil {
		t.Fatalf("parsing share from README: %v", err)
	}
	if share.Holder != friend.Name {
		t.Errorf("share holder: got %q, want %q", share.Holder, friend.Name)
	}
	if share.Threshold != threshold {
		t.Errorf("share threshold: got %d, want %d", share.Threshold, threshold)
	}
	if share.Total != len(allFriends) {
		t.Errorf("share total: got %d, want %d", share.Total, len(allFriends))
	}

	// Verify share checksum
	if err := share.Verify(); err != nil {
		t.Errorf("share verification failed: %v", err)
	}

	// Verify README contains other friends (not this one)
	for _, f := range allFriends {
		if f.Name == friend.Name {
			// Should NOT contain own contact in contacts section
			// (but will contain name in header, so just check contact)
			continue
		}
		if f.Contact != "" && !strings.Contains(readmeContent, f.Contact) {
			t.Errorf("README missing contact for %s", f.Name)
		}
	}

	// Verify README contains metadata footer
	if !strings.Contains(readmeContent, "METADATA FOOTER") {
		t.Error("README missing metadata footer")
	}
	if !strings.Contains(readmeContent, "rememory-version:") {
		t.Error("README missing version in footer")
	}
	if !strings.Contains(readmeContent, "checksum-manifest:") {
		t.Error("README missing manifest checksum in footer")
	}

	// Verify recover.html contains expected elements
	if !strings.Contains(recoverContent, "🧠 ReMemory Recovery") {
		t.Error("recover.html missing title")
	}
	if !strings.Contains(recoverContent, "v1.0.0-test") {
		t.Error("recover.html missing version")
	}
	if !strings.Contains(recoverContent, "WASM_BINARY") {
		t.Error("recover.html missing embedded WASM")
	}
}

// TestBundleRecovery tests recovering from bundle contents
func TestBundleRecovery(t *testing.T) {
	// Setup: create and seal a project
	baseDir := t.TempDir()
	projectDir := filepath.Join(baseDir, "test-recovery-project")

	friends := []project.Friend{
		{Name: "Alice", Contact: "alice@example.com"},
		{Name: "Bob", Contact: "bob@example.com"},
		{Name: "Carol", Contact: "carol@example.com"},
	}
	threshold := 2

	p, err := project.New(projectDir, "test-recovery", threshold, friends)
	if err != nil {
		t.Fatalf("creating project: %v", err)
	}

	// Add secret content
	secretContent := "Recovery test secret: the password is correct-horse-battery-staple"
	secretFile := filepath.Join(p.ManifestPath(), "secret.txt")
	if err := os.WriteFile(secretFile, []byte(secretContent), 0644); err != nil {
		t.Fatalf("writing secret: %v", err)
	}

	// Seal
	var archiveBuf bytes.Buffer
	if _, err := manifest.Archive(&archiveBuf, p.ManifestPath()); err != nil {
		t.Fatalf("archiving: %v", err)
	}

	passphrase, _ := crypto.GeneratePassphrase(crypto.DefaultPassphraseBytes)

	os.MkdirAll(p.OutputPath(), 0755)
	os.MkdirAll(p.SharesPath(), 0755)

	manifestFile, _ := os.Create(p.ManifestAgePath())
	core.Encrypt(manifestFile, bytes.NewReader(archiveBuf.Bytes()), passphrase)
	manifestFile.Close()

	shares, _ := core.Split([]byte(passphrase), len(friends), threshold)
	shareInfos := make([]project.ShareInfo, len(friends))
	for i, data := range shares {
		share := core.NewShare(i+1, len(friends), threshold, friends[i].Name, data)
		sharePath := filepath.Join(p.SharesPath(), share.Filename())
		os.WriteFile(sharePath, []byte(share.Encode()), 0644)
		shareInfos[i] = project.ShareInfo{
			Friend:   friends[i].Name,
			File:     share.Filename(),
			Checksum: share.Checksum,
		}
	}

	// Mark project as sealed
	manifestData, _ := os.ReadFile(p.ManifestAgePath())
	p.Sealed = &project.Sealed{
		At:               time.Now(),
		ManifestChecksum: core.HashBytes(manifestData),
		VerificationHash: core.HashString(passphrase),
		Shares:           shareInfos,
	}
	p.Save()

	// Generate bundles
	fakeWASM := []byte("fake-wasm")
	cfg := bundle.Config{
		Version:          "v1.0.0",
		GitHubReleaseURL: "https://example.com",
		WASMBytes:        fakeWASM,
	}
	bundle.GenerateAll(p, cfg)

	// Now simulate recovery using bundles
	bundlesDir := filepath.Join(p.OutputPath(), "bundles")

	// Extract shares from Alice's and Bob's bundles (threshold = 2)
	aliceBundle := filepath.Join(bundlesDir, "bundle-alice.zip")
	bobBundle := filepath.Join(bundlesDir, "bundle-bob.zip")

	aliceShare := extractShareFromBundle(t, aliceBundle)
	bobShare := extractShareFromBundle(t, bobBundle)
	bundleManifestData := extractManifestFromBundle(t, aliceBundle) // Same in all bundles

	// Combine shares
	recoveredPass, err := core.Combine([][]byte{aliceShare.Data, bobShare.Data})
	if err != nil {
		t.Fatalf("combining shares: %v", err)
	}

	// Decrypt manifest
	var decrypted bytes.Buffer
	if err := core.Decrypt(&decrypted, bytes.NewReader(bundleManifestData), string(recoveredPass)); err != nil {
		t.Fatalf("decrypting: %v", err)
	}

	// Extract
	extractDir := t.TempDir()
	extractResult, err := manifest.Extract(&decrypted, extractDir)
	if err != nil {
		t.Fatalf("extracting: %v", err)
	}

	// Verify content
	recovered, err := os.ReadFile(filepath.Join(extractResult.Path, "secret.txt"))
	if err != nil {
		t.Fatalf("reading recovered: %v", err)
	}
	if string(recovered) != secretContent {
		t.Errorf("content mismatch: got %q, want %q", recovered, secretContent)
	}
}

func extractShareFromBundle(t *testing.T, bundlePath string) *core.Share {
	t.Helper()

	r, err := zip.OpenReader(bundlePath)
	if err != nil {
		t.Fatalf("opening bundle: %v", err)
	}
	defer r.Close()

	for _, f := range r.File {
		if f.Name == "README.txt" {
			rc, _ := f.Open()
			data := make([]byte, f.UncompressedSize64)
			rc.Read(data)
			rc.Close()

			share, err := core.ParseShare(data)
			if err != nil {
				t.Fatalf("parsing share: %v", err)
			}
			return share
		}
	}
	t.Fatal("README.txt not found in bundle")
	return nil
}

func extractManifestFromBundle(t *testing.T, bundlePath string) []byte {
	t.Helper()

	r, err := zip.OpenReader(bundlePath)
	if err != nil {
		t.Fatalf("opening bundle: %v", err)
	}
	defer r.Close()

	for _, f := range r.File {
		if f.Name == "MANIFEST.age" {
			rc, _ := f.Open()
			data := make([]byte, f.UncompressedSize64)
			rc.Read(data)
			rc.Close()
			return data
		}
	}
	t.Fatal("MANIFEST.age not found in bundle")
	return nil
}

// TestAnonymousBundleGeneration tests bundle generation for anonymous projects
func TestAnonymousBundleGeneration(t *testing.T) {
	baseDir := t.TempDir()
	projectDir := filepath.Join(baseDir, "test-anon-project")

	// Create anonymous project with 5 shares, threshold 3
	p, err := project.NewAnonymous(projectDir, "test-anon", 3, 5)
	if err != nil {
		t.Fatalf("creating anonymous project: %v", err)
	}

	// Verify project is anonymous
	if !p.Anonymous {
		t.Fatal("project should be anonymous")
	}

	// Add secret content
	secretContent := "Anonymous mode secret: the treasure is hidden"
	secretFile := filepath.Join(p.ManifestPath(), "secrets.txt")
	if err := os.WriteFile(secretFile, []byte(secretContent), 0644); err != nil {
		t.Fatalf("writing secret: %v", err)
	}

	// Seal the project
	var archiveBuf bytes.Buffer
	if _, err := manifest.Archive(&archiveBuf, p.ManifestPath()); err != nil {
		t.Fatalf("archiving: %v", err)
	}

	passphrase, err := crypto.GeneratePassphrase(crypto.DefaultPassphraseBytes)
	if err != nil {
		t.Fatalf("generating passphrase: %v", err)
	}

	// Create output directories
	os.MkdirAll(p.OutputPath(), 0755)
	os.MkdirAll(p.SharesPath(), 0755)

	// Encrypt manifest
	manifestFile, _ := os.Create(p.ManifestAgePath())
	core.Encrypt(manifestFile, bytes.NewReader(archiveBuf.Bytes()), passphrase)
	manifestFile.Close()

	// Split passphrase and write shares
	shares, _ := core.Split([]byte(passphrase), len(p.Friends), p.Threshold)
	shareInfos := make([]project.ShareInfo, len(p.Friends))
	for i, data := range shares {
		share := core.NewShare(i+1, len(p.Friends), p.Threshold, p.Friends[i].Name, data)
		sharePath := filepath.Join(p.SharesPath(), share.Filename())
		os.WriteFile(sharePath, []byte(share.Encode()), 0644)
		shareInfos[i] = project.ShareInfo{
			Friend:   p.Friends[i].Name,
			File:     share.Filename(),
			Checksum: share.Checksum,
		}
	}

	// Mark project as sealed
	manifestData, _ := os.ReadFile(p.ManifestAgePath())
	p.Sealed = &project.Sealed{
		At:               time.Now(),
		ManifestChecksum: core.HashBytes(manifestData),
		VerificationHash: core.HashString(passphrase),
		Shares:           shareInfos,
	}
	p.Save()

	// Generate bundles
	fakeWASM := []byte("fake-wasm")
	cfg := bundle.Config{
		Version:          "v1.0.0-test",
		GitHubReleaseURL: "https://example.com",
		WASMBytes:        fakeWASM,
	}
	if err := bundle.GenerateAll(p, cfg); err != nil {
		t.Fatalf("generating bundles: %v", err)
	}

	// Verify bundles were created with correct names
	bundlesDir := filepath.Join(p.OutputPath(), "bundles")
	for i := 1; i <= 5; i++ {
		bundleName := fmt.Sprintf("bundle-share-%d.zip", i)
		bundlePath := filepath.Join(bundlesDir, bundleName)

		if _, err := os.Stat(bundlePath); os.IsNotExist(err) {
			t.Errorf("bundle %s not found", bundleName)
			continue
		}

		// Verify bundle contents
		t.Run(bundleName, func(t *testing.T) {
			verifyAnonymousBundle(t, bundlePath, i, 5, 3)
		})
	}
}

func verifyAnonymousBundle(t *testing.T, bundlePath string, shareNum, total, threshold int) {
	t.Helper()

	r, err := zip.OpenReader(bundlePath)
	if err != nil {
		t.Fatalf("opening bundle: %v", err)
	}
	defer r.Close()

	var readmeContent string
	for _, f := range r.File {
		if f.Name == "README.txt" {
			rc, _ := f.Open()
			data, _ := io.ReadAll(rc)
			rc.Close()
			readmeContent = string(data)
			break
		}
	}

	if readmeContent == "" {
		t.Fatal("README.txt not found")
	}

	// Anonymous READMEs should NOT contain "OTHER SHARE HOLDERS" section
	if strings.Contains(readmeContent, "OTHER SHARE HOLDERS") {
		t.Error("anonymous README should not contain OTHER SHARE HOLDERS section")
	}

	// Should contain anonymous-specific warning text
	if !strings.Contains(readmeContent, "combine this with other shares") {
		t.Error("anonymous README should mention combining with other shares")
	}

	// Should NOT contain "friends listed below"
	if strings.Contains(readmeContent, "friends listed below") {
		t.Error("anonymous README should not mention friends listed below")
	}

	// Should contain correct threshold info
	thresholdText := fmt.Sprintf("At least %d of you must cooperate", threshold)
	if !strings.Contains(readmeContent, thresholdText) {
		t.Errorf("README should contain threshold info: %s", thresholdText)
	}

	totalText := fmt.Sprintf("one of %d trusted friends", total)
	if !strings.Contains(readmeContent, totalText) {
		t.Errorf("README should contain total info: %s", totalText)
	}

	// Parse and verify share
	share, err := core.ParseShare([]byte(readmeContent))
	if err != nil {
		t.Fatalf("parsing share: %v", err)
	}

	expectedHolder := fmt.Sprintf("Share %d", shareNum)
	if share.Holder != expectedHolder {
		t.Errorf("holder: got %q, want %q", share.Holder, expectedHolder)
	}

	if err := share.Verify(); err != nil {
		t.Errorf("share verification failed: %v", err)
	}
}

// TestAnonymousBundleRecovery tests recovery from anonymous bundles
func TestAnonymousBundleRecovery(t *testing.T) {
	baseDir := t.TempDir()
	projectDir := filepath.Join(baseDir, "test-anon-recovery")

	// Create anonymous project
	p, err := project.NewAnonymous(projectDir, "test-recovery", 2, 3)
	if err != nil {
		t.Fatalf("creating project: %v", err)
	}

	// Add secret content
	secretContent := "The secret is: anonymous-mode-works"
	secretFile := filepath.Join(p.ManifestPath(), "secret.txt")
	os.WriteFile(secretFile, []byte(secretContent), 0644)

	// Seal
	var archiveBuf bytes.Buffer
	manifest.Archive(&archiveBuf, p.ManifestPath())
	passphrase, _ := crypto.GeneratePassphrase(crypto.DefaultPassphraseBytes)

	os.MkdirAll(p.OutputPath(), 0755)
	os.MkdirAll(p.SharesPath(), 0755)

	manifestFile, _ := os.Create(p.ManifestAgePath())
	core.Encrypt(manifestFile, bytes.NewReader(archiveBuf.Bytes()), passphrase)
	manifestFile.Close()

	shares, _ := core.Split([]byte(passphrase), len(p.Friends), p.Threshold)
	shareInfos := make([]project.ShareInfo, len(p.Friends))
	for i, data := range shares {
		share := core.NewShare(i+1, len(p.Friends), p.Threshold, p.Friends[i].Name, data)
		sharePath := filepath.Join(p.SharesPath(), share.Filename())
		os.WriteFile(sharePath, []byte(share.Encode()), 0644)
		shareInfos[i] = project.ShareInfo{
			Friend:   p.Friends[i].Name,
			File:     share.Filename(),
			Checksum: share.Checksum,
		}
	}

	manifestData, _ := os.ReadFile(p.ManifestAgePath())
	p.Sealed = &project.Sealed{
		At:               time.Now(),
		ManifestChecksum: core.HashBytes(manifestData),
		VerificationHash: core.HashString(passphrase),
		Shares:           shareInfos,
	}
	p.Save()

	// Generate bundles
	fakeWASM := []byte("fake-wasm")
	cfg := bundle.Config{
		Version:          "v1.0.0",
		GitHubReleaseURL: "https://example.com",
		WASMBytes:        fakeWASM,
	}
	bundle.GenerateAll(p, cfg)

	// Recover using bundles
	bundlesDir := filepath.Join(p.OutputPath(), "bundles")
	bundle1 := filepath.Join(bundlesDir, "bundle-share-1.zip")
	bundle2 := filepath.Join(bundlesDir, "bundle-share-2.zip")

	share1 := extractShareFromBundle(t, bundle1)
	share2 := extractShareFromBundle(t, bundle2)
	bundleManifest := extractManifestFromBundle(t, bundle1)

	// Combine shares
	recoveredPass, err := core.Combine([][]byte{share1.Data, share2.Data})
	if err != nil {
		t.Fatalf("combining shares: %v", err)
	}

	// Decrypt
	var decrypted bytes.Buffer
	if err := core.Decrypt(&decrypted, bytes.NewReader(bundleManifest), string(recoveredPass)); err != nil {
		t.Fatalf("decrypting: %v", err)
	}

	// Extract and verify
	extractDir := t.TempDir()
	extractResult, err := manifest.Extract(&decrypted, extractDir)
	if err != nil {
		t.Fatalf("extracting: %v", err)
	}

	recovered, err := os.ReadFile(filepath.Join(extractResult.Path, "secret.txt"))
	if err != nil {
		t.Fatalf("reading recovered: %v", err)
	}
	if string(recovered) != secretContent {
		t.Errorf("content mismatch: got %q, want %q", recovered, secretContent)
	}
}
