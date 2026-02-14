package cmd

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/eljojo/rememory/internal/bundle"
	"github.com/eljojo/rememory/internal/core"
	"github.com/eljojo/rememory/internal/crypto"
	"github.com/eljojo/rememory/internal/html"
	"github.com/eljojo/rememory/internal/manifest"
	"github.com/eljojo/rememory/internal/project"
	"github.com/spf13/cobra"
)

var resealCmd = &cobra.Command{
	Use:   "reseal share1.txt share2.txt ... [--recovery-url URL]",
	Short: "Re-encrypt manifest with the same passphrase (key reuse)",
	Long: `Reseal re-encrypts the manifest directory with the original passphrase,
allowing you to update encrypted data without regenerating shares.

This command:
  1. Loads an existing sealed project
  2. Requires you to provide share files to recover the original passphrase
  3. Verifies the recovered passphrase matches the stored checksum
  4. Archives and encrypts the updated manifest/ directory
  5. Saves versioned MANIFEST-<timestamp>.age file
  6. Regenerates bundles with the new manifest (keeps shares identical)

You must provide at least the threshold number of shares to recover the passphrase.

Example:
  rememory reseal SHARE-alice.txt SHARE-bob.txt SHARE-carol.txt

Run this command inside a project directory (created with 'rememory init').`,
	RunE: runReseal,
}

func init() {
	resealCmd.Flags().String("recovery-url", core.DefaultRecoveryURL, "Base URL for QR code in PDF")
	rootCmd.AddCommand(resealCmd)
}

func runReseal(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("please provide at least one share file")
	}

	// Find and load the project
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting current directory: %w", err)
	}

	projectDir, err := project.FindProjectDir(cwd)
	if err != nil {
		return err
	}

	p, err := project.Load(projectDir)
	if err != nil {
		return fmt.Errorf("loading project: %w", err)
	}

	if err := p.Validate(); err != nil {
		return fmt.Errorf("invalid project: %w", err)
	}

	// Check if project supports reseal
	if !p.SupportsReseal() {
		return fmt.Errorf("this project was sealed with an older version and does not contain a PassphraseChecksum.\nCreate a new project to use the reseal feature")
	}

	recoveryURL, _ := cmd.Flags().GetString("recovery-url")

	if err := resealProject(p, args, recoveryURL); err != nil {
		return err
	}

	bundlesDir := filepath.Join(p.OutputPath(), "bundles")
	fmt.Printf("\nUpdated bundles saved to: %s\n", bundlesDir)

	return nil
}

// resealProject re-encrypts the manifest with a recovered passphrase and regenerates bundles.
func resealProject(p *project.Project, shareFiles []string, recoveryURL string) error {
	// Check manifest directory exists and has content
	manifestDir := p.ManifestPath()
	fileCount, err := manifest.CountFiles(manifestDir)
	if err != nil {
		return fmt.Errorf("checking manifest directory: %w", err)
	}
	if fileCount == 0 {
		return fmt.Errorf("manifest directory is empty: %s", manifestDir)
	}

	dirSize, err := manifest.DirSize(manifestDir)
	if err != nil {
		return fmt.Errorf("calculating manifest size: %w", err)
	}

	fmt.Printf("Archiving manifest/ (%d files, %s)...\n", fileCount, formatSize(dirSize))

	// Parse and validate shares
	fmt.Printf("Reading %d share files...\n", len(shareFiles))

	shares := make([]*core.Share, len(shareFiles))
	for i, path := range shareFiles {
		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("reading share %s: %w", path, err)
		}

		share, err := core.ParseShare(content)
		if err != nil {
			return fmt.Errorf("parsing share %s: %w", path, err)
		}

		// Verify checksum
		if err := share.Verify(); err != nil {
			return fmt.Errorf("share %s: %w", path, err)
		}

		shares[i] = share
	}

	// Validate shares are compatible
	if len(shares) == 0 {
		return fmt.Errorf("no shares provided")
	}

	first := shares[0]
	for i, share := range shares[1:] {
		if share.Version != first.Version {
			return fmt.Errorf("share %d has different version (v%d vs v%d) — all shares must be from the same bundle", i+2, share.Version, first.Version)
		}
		if share.Total != first.Total {
			return fmt.Errorf("share %d has different total (%d vs %d)", i+2, share.Total, first.Total)
		}
		if share.Threshold != first.Threshold {
			return fmt.Errorf("share %d has different threshold (%d vs %d)", i+2, share.Threshold, first.Threshold)
		}
	}

	// Check we have enough shares
	if len(shares) < first.Threshold {
		return fmt.Errorf("need at least %d shares to recover (you provided %d)", first.Threshold, len(shares))
	}

	// Check for duplicate indices
	seen := make(map[int]bool)
	for _, share := range shares {
		if seen[share.Index] {
			return fmt.Errorf("duplicate share index %d", share.Index)
		}
		seen[share.Index] = true
	}

	fmt.Printf("Combining %d shares...\n", len(shares))

	// Extract raw share data
	shareData := make([][]byte, len(shares))
	for i, share := range shares {
		shareData[i] = share.Data
	}

	// Reconstruct passphrase
	recovered, err := core.Combine(shareData)
	if err != nil {
		return fmt.Errorf("combining shares: %w", err)
	}

	passphrase := core.RecoverPassphrase(recovered, first.Version)

	// Verify passphrase matches stored checksum
	fmt.Print("Verifying passphrase... ")
	if !crypto.VerifyPassphrase(passphrase, p.Sealed.PassphraseChecksum) {
		fmt.Println("FAILED")
		return fmt.Errorf("recovered passphrase does not match stored checksum (wrong shares or corrupted project)")
	}
	fmt.Println("OK")

	// Archive the manifest directory
	var archiveBuf bytes.Buffer
	archiveResult, err := manifest.Archive(&archiveBuf, manifestDir)
	if err != nil {
		return fmt.Errorf("archiving manifest: %w", err)
	}

	for _, warning := range archiveResult.Warnings {
		fmt.Printf("  Warning: %s\n", warning)
	}

	fmt.Println("Encrypting with age...")

	// Encrypt the archive with recovered passphrase
	var encryptedBuf bytes.Buffer
	archiveReader := bytes.NewReader(archiveBuf.Bytes())
	if err := core.Encrypt(&encryptedBuf, archiveReader, passphrase); err != nil {
		return fmt.Errorf("encrypting: %w", err)
	}

	// Create versioned manifest file with timestamp
	timestamp := time.Now().UTC().Format("20060102-150405")
	versionedManifestName := fmt.Sprintf("MANIFEST-%s.age", timestamp)
	versionedManifestPath := filepath.Join(p.OutputPath(), versionedManifestName)

	fmt.Printf("Saving %s...\n", versionedManifestName)

	if err := os.WriteFile(versionedManifestPath, encryptedBuf.Bytes(), 0644); err != nil {
		return fmt.Errorf("writing encrypted manifest: %w", err)
	}

	// Also update the standard MANIFEST.age for backwards compatibility with recover
	standardManifestPath := p.ManifestAgePath()
	if err := os.WriteFile(standardManifestPath, encryptedBuf.Bytes(), 0644); err != nil {
		return fmt.Errorf("writing standard manifest: %w", err)
	}

	// Update sealed metadata with new manifest checksum and timestamp, keeping shares/passphrase data
	newChecksum, err := crypto.HashFile(standardManifestPath)
	if err != nil {
		return fmt.Errorf("hashing new manifest: %w", err)
	}

	if p.Sealed != nil {
		p.Sealed.ManifestChecksum = newChecksum
		p.Sealed.At = time.Now().UTC()
	}

	if err := p.Save(); err != nil {
		return fmt.Errorf("saving project metadata: %w", err)
	}
	fmt.Printf("Generating bundles for %d friends...\n", len(p.Friends))

	wasmBytes := html.GetRecoverWASMBytes()
	if len(wasmBytes) == 0 {
		return fmt.Errorf("recover.wasm not embedded - rebuild with 'make build'")
	}

	cfg := bundle.Config{
		Version:          version,
		GitHubReleaseURL: fmt.Sprintf("https://github.com/eljojo/rememory/releases/tag/%s", version),
		WASMBytes:        wasmBytes,
		RecoveryURL:      recoveryURL,
		ResealMode:       true, // Indicate this is a reseal, keep existing shares
	}

	if err := bundle.GenerateAll(p, cfg); err != nil {
		return fmt.Errorf("generating bundles: %w", err)
	}

	// Print reseal summary
	fmt.Println()
	fmt.Println("Resealed:")
	relManifest, _ := filepath.Rel(p.Path, versionedManifestPath)
	fmt.Printf("  %s %s\n", green("✓"), relManifest)
	relStandard, _ := filepath.Rel(p.Path, standardManifestPath)
	fmt.Printf("  %s %s\n", green("✓"), relStandard)

	// Print bundle listing
	bundlesDir := filepath.Join(p.OutputPath(), "bundles")
	entries, _ := os.ReadDir(bundlesDir)

	fmt.Println()
	fmt.Println("Bundles updated:")
	for _, entry := range entries {
		if !entry.IsDir() {
			info, _ := entry.Info()
			fmt.Printf("  %s %s (%s)\n", green("✓"), entry.Name(), formatSize(info.Size()))
		}
	}

	return nil
}


