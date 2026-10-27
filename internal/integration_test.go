package integration_test

import (
	"archive/zip"
	"bytes"
	cryptorand "crypto/rand"
	"encoding/base64"
	"encoding/json"
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
	"github.com/eljojo/rememory/internal/html"
	"github.com/eljojo/rememory/internal/manifest"
	"github.com/eljojo/rememory/internal/project"
	"github.com/eljojo/rememory/internal/translations"
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
	if _, err := manifest.ArchiveZip(&archiveBuf, p.ManifestPath()); err != nil {
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
		shareObjects[i] = core.NewShare(1, i+1, len(friends), threshold, friends[i].Name, data)
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
			extractResult, err := manifest.ExtractAuto(bytes.NewReader(decryptedBuf.Bytes()), extractDir)
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
	share := core.NewShare(1, 1, 5, 3, "Alice", []byte("test-data"))
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
	if _, err := manifest.ArchiveZip(&archiveBuf, manifestDir); err != nil {
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
	extractResult, err := manifest.ExtractAuto(bytes.NewReader(decrypted.Bytes()), extractDir)
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
					shareObjs[i] = core.NewShare(1, i+1, n, k, "", data)
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
	if _, err := manifest.ArchiveZip(&archiveBuf, p.ManifestPath()); err != nil {
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
		share := core.NewShare(1, i+1, len(friends), threshold, friends[i].Name, data)
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

	// Generate bundles (recover.html uses native JavaScript crypto, no WASM)
	cfg := bundle.Config{
		Version: "v1.0.0-test",
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
	var foundReadmeTxt, foundReadmePdf, foundRecover bool

	var readmeContent string
	var recoverContent string

	for _, f := range r.File {
		rc, err := f.Open()
		if err != nil {
			t.Fatalf("opening %s: %v", f.Name, err)
		}
		data, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			t.Fatalf("reading %s: %v", f.Name, err)
		}

		switch {
		case translations.IsReadmeFile(f.Name, ".txt"):
			foundReadmeTxt = true
			readmeContent = string(data)
		case translations.IsReadmeFile(f.Name, ".pdf"):
			foundReadmePdf = true
		case f.Name == "recover.html":
			foundRecover = true
			recoverContent = string(data)
		}
	}

	if !foundReadmeTxt {
		t.Error("missing README .txt file")
	}
	if !foundReadmePdf {
		t.Error("missing README .pdf file")
	}
	// MANIFEST.age is only in the ZIP when NOT embedded in recover.html.
	// Small manifests are embedded, so they won't be in the ZIP.
	// We still parse it if present, but don't require it.
	if !foundRecover {
		t.Error("missing file: recover.html")
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
	if !strings.Contains(recoverContent, "🧠 ReMemory") {
		t.Error("recover.html missing title")
	}
	if !strings.Contains(recoverContent, "v1.0.0-test") {
		t.Error("recover.html missing version")
	}
	// Note: recover.html now uses native JavaScript crypto, no WASM needed
}

// TestPagesGeneration tests that static pages output contains the right files and config.
func TestPagesGeneration(t *testing.T) {
	// Setup: create a sealed project (reuses the pattern from TestBundleGeneration)
	baseDir := t.TempDir()
	projectDir := filepath.Join(baseDir, "test-pages-project")

	friends := []project.Friend{
		{Name: "Alice", Contact: "alice@example.com"},
		{Name: "Bob", Contact: "bob@example.com"},
		{Name: "Carol", Contact: "carol@example.com"},
	}
	threshold := 2

	p, err := project.New(projectDir, "test-pages-project", threshold, friends)
	if err != nil {
		t.Fatalf("creating project: %v", err)
	}

	// Add secret content
	secretFile := filepath.Join(p.ManifestPath(), "secrets.txt")
	if err := os.WriteFile(secretFile, []byte("pages test secret"), 0644); err != nil {
		t.Fatalf("writing secret: %v", err)
	}

	// Seal the project
	var archiveBuf bytes.Buffer
	if _, err := manifest.ArchiveZip(&archiveBuf, p.ManifestPath()); err != nil {
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
		share := core.NewShare(1, i+1, len(friends), threshold, friends[i].Name, data)
		sharePath := filepath.Join(p.SharesPath(), share.Filename())
		os.WriteFile(sharePath, []byte(share.Encode()), 0644)
		shareInfos[i] = project.ShareInfo{
			Friend:   friends[i].Name,
			File:     share.Filename(),
			Checksum: share.Checksum,
		}
	}

	manifestData, _ := os.ReadFile(p.ManifestAgePath())
	manifestChecksum := core.HashBytes(manifestData)

	p.Sealed = &project.Sealed{
		At:               time.Now(),
		ManifestChecksum: manifestChecksum,
		VerificationHash: core.HashString(passphrase),
		Shares:           shareInfos,
	}
	p.Save()

	// Generate pages output
	pagesDir := p.PagesPath()
	if err := os.MkdirAll(pagesDir, 0755); err != nil {
		t.Fatalf("creating pages dir: %v", err)
	}

	// Copy MANIFEST.age
	if err := os.WriteFile(filepath.Join(pagesDir, "MANIFEST.age"), manifestData, 0644); err != nil {
		t.Fatalf("writing manifest: %v", err)
	}

	// Generate static-hosted recover.html
	html.SetVersion("v1.0.0-test")
	recoverContent := html.GenerateRecoverHTML(nil, html.RecoverHTMLOptions{
		StaticHosted: true,
	})
	if err := os.WriteFile(filepath.Join(pagesDir, "recover.html"), []byte(recoverContent), 0644); err != nil {
		t.Fatalf("writing recover.html: %v", err)
	}

	// Assert: output/pages/recover.html exists
	recoverPath := filepath.Join(pagesDir, "recover.html")
	if _, err := os.Stat(recoverPath); os.IsNotExist(err) {
		t.Fatal("output/pages/recover.html not found")
	}

	// Assert: output/pages/MANIFEST.age exists and matches sealed manifest
	pagesManifest, err := os.ReadFile(filepath.Join(pagesDir, "MANIFEST.age"))
	if err != nil {
		t.Fatalf("reading pages manifest: %v", err)
	}
	pagesManifestChecksum := core.HashBytes(pagesManifest)
	if pagesManifestChecksum != manifestChecksum {
		t.Error("pages MANIFEST.age checksum does not match sealed manifest")
	}

	// Assert: recover.html contains manifestURL config
	if !strings.Contains(recoverContent, `"manifestURL":"./MANIFEST.age"`) {
		t.Error("pages recover.html missing manifestURL config")
	}

	// Assert: recover.html does NOT contain /api/bundle
	if strings.Contains(recoverContent, "/api/bundle") {
		t.Error("pages recover.html contains /api/bundle server URLs")
	}

	// Assert: nav links use filenames, not server routes
	if strings.Contains(recoverContent, `href="/create"`) {
		t.Error("pages recover.html has server nav rewrites")
	}
	if !strings.Contains(recoverContent, `href="about.html"`) {
		t.Error("pages recover.html missing standard nav links")
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
	if _, err := manifest.ArchiveZip(&archiveBuf, p.ManifestPath()); err != nil {
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
		share := core.NewShare(1, i+1, len(friends), threshold, friends[i].Name, data)
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
	cfg := bundle.Config{
		Version: "v1.0.0",
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
	extractResult, err := manifest.ExtractAuto(bytes.NewReader(decrypted.Bytes()), extractDir)
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
	share, err := bundle.ExtractShareFromZip(bundlePath)
	if err != nil {
		t.Fatalf("extracting share from %s: %v", bundlePath, err)
	}
	return share
}

func extractManifestFromBundle(t *testing.T, bundlePath string) []byte {
	t.Helper()
	data, err := bundle.ExtractManifestFromZip(bundlePath)
	if err != nil {
		t.Fatalf("extracting manifest from %s: %v", bundlePath, err)
	}
	return data
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
	if _, err := manifest.ArchiveZip(&archiveBuf, p.ManifestPath()); err != nil {
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
		share := core.NewShare(1, i+1, len(p.Friends), p.Threshold, p.Friends[i].Name, data)
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
	cfg := bundle.Config{
		Version: "v1.0.0-test",
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
		if translations.IsReadmeFile(f.Name, ".txt") {
			rc, _ := f.Open()
			data, _ := io.ReadAll(rc)
			rc.Close()
			readmeContent = string(data)
			break
		}
	}

	if readmeContent == "" {
		t.Fatal("README file not found")
	}

	// Anonymous READMEs should NOT contain "OTHER SHARE HOLDERS" section
	if strings.Contains(readmeContent, "OTHER SHARE HOLDERS") {
		t.Error("anonymous README should not contain OTHER SHARE HOLDERS section")
	}

	// Should contain anonymous-specific warning text
	if !strings.Contains(readmeContent, "combine it with other pieces") {
		t.Error("anonymous README should mention combining with other pieces")
	}

	// Should NOT contain "friends listed below"
	if strings.Contains(readmeContent, "friends listed below") {
		t.Error("anonymous README should not mention friends listed below")
	}

	// Should contain correct threshold info
	thresholdText := fmt.Sprintf("At least %d of you must come together", threshold)
	if !strings.Contains(readmeContent, thresholdText) {
		t.Errorf("README should contain threshold info: %s", thresholdText)
	}

	totalText := fmt.Sprintf("one of %d people", total)
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

	// recover.html looks pieces up in otherFriends; it must be an array.
	for _, f := range r.File {
		if f.Name == "recover.html" {
			rc, _ := f.Open()
			data, _ := io.ReadAll(rc)
			rc.Close()
			if !strings.Contains(string(data), `"otherFriends":[]`) {
				t.Error(`anonymous recover.html should embed "otherFriends":[]`)
			}
		}
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
	manifest.ArchiveZip(&archiveBuf, p.ManifestPath())
	passphrase, _ := crypto.GeneratePassphrase(crypto.DefaultPassphraseBytes)

	os.MkdirAll(p.OutputPath(), 0755)
	os.MkdirAll(p.SharesPath(), 0755)

	manifestFile, _ := os.Create(p.ManifestAgePath())
	core.Encrypt(manifestFile, bytes.NewReader(archiveBuf.Bytes()), passphrase)
	manifestFile.Close()

	shares, _ := core.Split([]byte(passphrase), len(p.Friends), p.Threshold)
	shareInfos := make([]project.ShareInfo, len(p.Friends))
	for i, data := range shares {
		share := core.NewShare(1, i+1, len(p.Friends), p.Threshold, p.Friends[i].Name, data)
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
	cfg := bundle.Config{
		Version: "v1.0.0",
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
	extractResult, err := manifest.ExtractAuto(bytes.NewReader(decrypted.Bytes()), extractDir)
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

// TestManifestEmbedding verifies that small manifests are embedded in recover.html
// and that the NoEmbedManifest flag disables embedding.
func TestManifestEmbedding(t *testing.T) {
	// Helper to create a sealed project and generate bundles
	setup := func(t *testing.T, secretSize int, noEmbed bool) (string, []byte) {
		t.Helper()
		baseDir := t.TempDir()
		projectDir := filepath.Join(baseDir, "test-embed-project")

		friends := []project.Friend{
			{Name: "Alice", Contact: "alice@example.com"},
			{Name: "Bob", Contact: "bob@example.com"},
		}

		p, err := project.New(projectDir, "test-embed", 2, friends)
		if err != nil {
			t.Fatalf("creating project: %v", err)
		}

		// Use random data so it doesn't compress well (important for large manifest tests)
		secretData := make([]byte, secretSize)
		if _, err := cryptorand.Read(secretData); err != nil {
			t.Fatalf("generating random data: %v", err)
		}
		if err := os.WriteFile(filepath.Join(p.ManifestPath(), "data.bin"), secretData, 0644); err != nil {
			t.Fatalf("writing secret: %v", err)
		}

		var archiveBuf bytes.Buffer
		if _, err := manifest.ArchiveZip(&archiveBuf, p.ManifestPath()); err != nil {
			t.Fatalf("archiving: %v", err)
		}

		passphrase, _ := crypto.GeneratePassphrase(crypto.DefaultPassphraseBytes)
		os.MkdirAll(p.OutputPath(), 0755)
		os.MkdirAll(p.SharesPath(), 0755)

		manifestFile, _ := os.Create(p.ManifestAgePath())
		core.Encrypt(manifestFile, bytes.NewReader(archiveBuf.Bytes()), passphrase)
		manifestFile.Close()

		shares, _ := core.Split([]byte(passphrase), len(friends), 2)
		shareInfos := make([]project.ShareInfo, len(friends))
		for i, data := range shares {
			share := core.NewShare(1, i+1, len(friends), 2, friends[i].Name, data)
			sharePath := filepath.Join(p.SharesPath(), share.Filename())
			os.WriteFile(sharePath, []byte(share.Encode()), 0644)
			shareInfos[i] = project.ShareInfo{
				Friend:   friends[i].Name,
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

		cfg := bundle.Config{
			Version:         "v1.0.0",
			NoEmbedManifest: noEmbed,
		}
		if err := bundle.GenerateAll(p, cfg); err != nil {
			t.Fatalf("generating bundles: %v", err)
		}

		bundlesDir := filepath.Join(p.OutputPath(), "bundles")
		return bundlesDir, manifestData
	}

	extractPersonalization := func(t *testing.T, bundlePath string) *html.PersonalizationData {
		t.Helper()
		r, err := zip.OpenReader(bundlePath)
		if err != nil {
			t.Fatalf("opening bundle: %v", err)
		}
		defer r.Close()

		for _, f := range r.File {
			if f.Name != "recover.html" {
				continue
			}
			rc, _ := f.Open()
			data, _ := io.ReadAll(rc)
			rc.Close()

			content := string(data)
			start := strings.Index(content, "window.PERSONALIZATION = ")
			if start == -1 {
				t.Fatal("PERSONALIZATION not found in recover.html")
			}
			start += len("window.PERSONALIZATION = ")
			end := strings.Index(content[start:], ";\n")
			if end == -1 {
				t.Fatal("PERSONALIZATION end not found")
			}
			jsonStr := content[start : start+end]

			var pd html.PersonalizationData
			if err := json.Unmarshal([]byte(jsonStr), &pd); err != nil {
				t.Fatalf("parsing personalization JSON: %v", err)
			}
			return &pd
		}
		t.Fatal("recover.html not found in bundle")
		return nil
	}

	t.Run("small manifest is embedded", func(t *testing.T) {
		bundlesDir, manifestData := setup(t, 100, false)
		bundlePath := filepath.Join(bundlesDir, "bundle-alice.zip")

		pd := extractPersonalization(t, bundlePath)
		if pd.ManifestB64 == "" {
			t.Fatal("expected ManifestB64 to be set for small manifest")
		}

		decoded, err := base64.StdEncoding.DecodeString(pd.ManifestB64)
		if err != nil {
			t.Fatalf("decoding ManifestB64: %v", err)
		}
		if !bytes.Equal(decoded, manifestData) {
			t.Error("decoded ManifestB64 does not match original manifest data")
		}
	})

	t.Run("NoEmbedManifest flag prevents embedding", func(t *testing.T) {
		bundlesDir, _ := setup(t, 100, true)
		bundlePath := filepath.Join(bundlesDir, "bundle-alice.zip")

		pd := extractPersonalization(t, bundlePath)
		if pd.ManifestB64 != "" {
			t.Error("expected ManifestB64 to be empty when NoEmbedManifest is true")
		}
	})

	t.Run("large manifest is not embedded", func(t *testing.T) {
		if testing.Short() {
			t.Skip("skipping large manifest test in short mode")
		}
		// 11MB secret -> encrypted manifest will exceed 10MB threshold
		bundlesDir, _ := setup(t, 11*1024*1024, false)
		bundlePath := filepath.Join(bundlesDir, "bundle-alice.zip")

		pd := extractPersonalization(t, bundlePath)
		if pd.ManifestB64 != "" {
			t.Error("expected ManifestB64 to be empty for large manifest")
		}
	})

	t.Run("MANIFEST.age excluded from ZIP when embedded", func(t *testing.T) {
		bundlesDir, _ := setup(t, 100, false)
		bundlePath := filepath.Join(bundlesDir, "bundle-alice.zip")

		r, err := zip.OpenReader(bundlePath)
		if err != nil {
			t.Fatalf("opening bundle: %v", err)
		}
		defer r.Close()

		for _, f := range r.File {
			if f.Name == "MANIFEST.age" {
				t.Error("MANIFEST.age should NOT be in ZIP when embedded in recover.html")
			}
		}
	})

	t.Run("MANIFEST.age included in ZIP when not embedded", func(t *testing.T) {
		// NoEmbedManifest=true means manifest is NOT embedded, so MANIFEST.age must be in ZIP
		bundlesDir, _ := setup(t, 100, true)
		bundlePath := filepath.Join(bundlesDir, "bundle-alice.zip")

		r, err := zip.OpenReader(bundlePath)
		if err != nil {
			t.Fatalf("opening bundle: %v", err)
		}
		defer r.Close()

		found := false
		for _, f := range r.File {
			if f.Name == "MANIFEST.age" {
				found = true
				break
			}
		}
		if !found {
			t.Error("MANIFEST.age should be in ZIP when not embedded in recover.html")
		}
	})
}

// TestTlockFullWorkflow tests the complete seal-with-timelock and recover pipeline.
// Gated by REMEMORY_TEST_TLOCK=1 because it requires internet access (drand network).
func TestTlockFullWorkflow(t *testing.T) {
	if os.Getenv("REMEMORY_TEST_TLOCK") != "1" {
		t.Skip("set REMEMORY_TEST_TLOCK=1 to run tlock integration tests (requires internet)")
	}

	baseDir := t.TempDir()
	projectDir := filepath.Join(baseDir, "test-tlock-project")

	friends := []project.Friend{
		{Name: "Alice", Contact: "alice@example.com"},
		{Name: "Bob", Contact: "bob@example.com"},
		{Name: "Carol", Contact: "carol@example.com"},
	}
	threshold := 2

	p, err := project.New(projectDir, "test-tlock", threshold, friends)
	if err != nil {
		t.Fatalf("creating project: %v", err)
	}

	// Add secret content
	secretContent := "Tlock integration test secret: the answer is 42"
	if err := os.WriteFile(filepath.Join(p.ManifestPath(), "secrets.txt"), []byte(secretContent), 0644); err != nil {
		t.Fatalf("writing secret: %v", err)
	}

	// Archive manifest
	var archiveBuf bytes.Buffer
	if _, err := manifest.ArchiveZip(&archiveBuf, p.ManifestPath()); err != nil {
		t.Fatalf("archiving: %v", err)
	}

	// Tlock-encrypt to a recently-past round (inner layer)
	pastTime := time.Now().Add(-1 * time.Minute)
	tlockRound := core.RoundForTime(pastTime)
	unlockTime := core.TimeForRound(tlockRound)

	var tlockBuf bytes.Buffer
	if err := core.TlockEncrypt(&tlockBuf, bytes.NewReader(archiveBuf.Bytes()), tlockRound); err != nil {
		t.Fatalf("tlock encrypt: %v", err)
	}

	// Build tlock container (ZIP with tlock.json + manifest.tlock.age)
	meta := &core.TlockMeta{
		V:      core.TlockContainerVersion,
		Method: core.TlockMethodQuicknet,
		Round:  tlockRound,
		Unlock: unlockTime.UTC().Format(time.RFC3339),
		Chain:  core.QuicknetChainHash,
	}
	container, err := core.BuildTlockContainer(meta, tlockBuf.Bytes())
	if err != nil {
		t.Fatalf("building tlock container: %v", err)
	}

	// Age-encrypt the container (outer layer)
	raw, passphrase, err := crypto.GenerateRawPassphrase(crypto.DefaultPassphraseBytes)
	if err != nil {
		t.Fatalf("generating passphrase: %v", err)
	}

	var encryptedBuf bytes.Buffer
	if err := core.Encrypt(&encryptedBuf, bytes.NewReader(container), passphrase); err != nil {
		t.Fatalf("encrypting: %v", err)
	}
	manifestData := encryptedBuf.Bytes()

	// MANIFEST.age should be a plain age file (no JSON envelope)
	if manifestData[0] != 'a' { // 'a' from "age-encryption.org"
		t.Error("MANIFEST.age should start with age header, not JSON")
	}

	// Recovery: combine shares → age-decrypt → detect tlock container → tlock-decrypt → extract

	// Split passphrase
	shares, err := core.Split(raw, len(friends), threshold)
	if err != nil {
		t.Fatalf("splitting: %v", err)
	}

	// Combine threshold shares
	recoveredRaw, err := core.Combine(shares[:threshold])
	if err != nil {
		t.Fatalf("combining: %v", err)
	}
	recoveredPass := base64.RawURLEncoding.EncodeToString(recoveredRaw)

	// Age-decrypt (outer layer)
	var ageDecrypted bytes.Buffer
	if err := core.Decrypt(&ageDecrypted, bytes.NewReader(manifestData), recoveredPass); err != nil {
		t.Fatalf("age decrypting: %v", err)
	}

	// Detect tlock container
	decryptedData := ageDecrypted.Bytes()
	if !core.IsTlockContainer(decryptedData) {
		t.Fatal("expected tlock container after age-decryption")
	}

	// Open container and verify metadata
	parsedMeta, tlockCiphertext, err := core.OpenTlockContainer(decryptedData)
	if err != nil {
		t.Fatalf("opening tlock container: %v", err)
	}
	if parsedMeta.Round != tlockRound {
		t.Errorf("tlock round: got %d, want %d", parsedMeta.Round, tlockRound)
	}
	if parsedMeta.Method != core.TlockMethodQuicknet {
		t.Errorf("tlock method: got %q, want %q", parsedMeta.Method, core.TlockMethodQuicknet)
	}

	// Tlock-decrypt (inner layer — the round is in the past, so this should work)
	var tlockDecrypted bytes.Buffer
	if err := core.TlockDecrypt(&tlockDecrypted, bytes.NewReader(tlockCiphertext)); err != nil {
		t.Fatalf("tlock decrypting: %v", err)
	}

	// Extract archive
	extractDir := t.TempDir()
	extractResult, err := manifest.ExtractAuto(bytes.NewReader(tlockDecrypted.Bytes()), extractDir)
	if err != nil {
		t.Fatalf("extracting: %v", err)
	}

	// Verify content
	recovered, err := os.ReadFile(filepath.Join(extractResult.Path, "secrets.txt"))
	if err != nil {
		t.Fatalf("reading recovered: %v", err)
	}
	if string(recovered) != secretContent {
		t.Errorf("content mismatch: got %q, want %q", recovered, secretContent)
	}
}

// TestTlockFutureRoundCannotDecrypt encrypts to a far-future drand round and
// confirms decryption fails with ErrTooEarly.
// Gated by REMEMORY_TEST_TLOCK=1 because it requires internet access (drand network).
func TestTlockFutureRoundCannotDecrypt(t *testing.T) {
	if os.Getenv("REMEMORY_TEST_TLOCK") != "1" {
		t.Skip("set REMEMORY_TEST_TLOCK=1 to run tlock integration tests (requires internet)")
	}

	futureRound := core.RoundForTime(time.Now().Add(365 * 24 * time.Hour))

	var encrypted bytes.Buffer
	if err := core.TlockEncrypt(&encrypted, strings.NewReader("not yet"), futureRound); err != nil {
		t.Fatalf("tlock encrypt: %v", err)
	}

	var decrypted bytes.Buffer
	err := core.TlockDecrypt(&decrypted, bytes.NewReader(encrypted.Bytes()))
	if err == nil {
		t.Fatal("tlock decrypt should have failed for a future round")
	}
	if !core.IsTlockTooEarly(err) {
		t.Errorf("expected tlock.ErrTooEarly, got: %v", err)
	}
}

// TestTlockManifestWithoutTlock verifies that regular archives (no tlock container)
// are not detected as tlock.
func TestTlockManifestWithoutTlock(t *testing.T) {
	// A regular age-encrypted manifest starts with "age-encryption.org"
	data := []byte("age-encryption.org/v1\n-> scrypt...")
	if core.IsTlockContainer(data) {
		t.Error("regular age data should not be detected as tlock container")
	}

	// A ZIP that isn't a tlock container (no tlock.json)
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	f, _ := w.Create("hello.txt")
	f.Write([]byte("world"))
	w.Close()
	if core.IsTlockContainer(buf.Bytes()) {
		t.Error("regular ZIP should not be detected as tlock container")
	}
}

// TestTarGzBackwardCompatibility verifies that manifests created with the old
// tar.gz format can still be recovered. This is critical for bundles that may
// have been created with earlier versions of rememory.
func TestTarGzBackwardCompatibility(t *testing.T) {
	// Create a temp directory with test files
	baseDir := t.TempDir()
	manifestDir := filepath.Join(baseDir, "manifest")
	if err := os.MkdirAll(manifestDir, 0755); err != nil {
		t.Fatal(err)
	}

	secretContent := "backward compat secret"
	if err := os.WriteFile(filepath.Join(manifestDir, "secret.txt"), []byte(secretContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Archive with the old tar.gz format
	var archiveBuf bytes.Buffer
	if _, err := manifest.ArchiveTarGz(&archiveBuf, manifestDir); err != nil {
		t.Fatalf("archiving (tar.gz): %v", err)
	}

	// Encrypt
	passphrase, err := crypto.GeneratePassphrase(crypto.DefaultPassphraseBytes)
	if err != nil {
		t.Fatal(err)
	}
	var encryptedBuf bytes.Buffer
	if err := core.Encrypt(&encryptedBuf, bytes.NewReader(archiveBuf.Bytes()), passphrase); err != nil {
		t.Fatal(err)
	}

	// Decrypt
	var decrypted bytes.Buffer
	if err := core.Decrypt(&decrypted, bytes.NewReader(encryptedBuf.Bytes()), passphrase); err != nil {
		t.Fatal(err)
	}

	// Recover with ExtractAuto — should auto-detect tar.gz
	extractDir := t.TempDir()
	result, err := manifest.ExtractAuto(bytes.NewReader(decrypted.Bytes()), extractDir)
	if err != nil {
		t.Fatalf("ExtractAuto (tar.gz backward compat): %v", err)
	}

	got, err := os.ReadFile(filepath.Join(result.Path, "secret.txt"))
	if err != nil {
		t.Fatalf("reading extracted file: %v", err)
	}
	if string(got) != secretContent {
		t.Errorf("got %q, want %q", got, secretContent)
	}
}
