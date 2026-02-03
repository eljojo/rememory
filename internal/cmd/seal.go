package cmd

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/eljojo/rememory/internal/bundle"
	"github.com/eljojo/rememory/internal/crypto"
	"github.com/eljojo/rememory/internal/html"
	"github.com/eljojo/rememory/internal/manifest"
	"github.com/eljojo/rememory/internal/project"
	"github.com/eljojo/rememory/internal/shamir"
	"github.com/spf13/cobra"
)

var sealCmd = &cobra.Command{
	Use:   "seal",
	Short: "Encrypt the manifest, create shares, and generate bundles",
	Long: `Seal encrypts the manifest directory, splits the passphrase into shares,
and generates distribution bundles for each friend.

This command:
  1. Archives the manifest/ directory
  2. Encrypts it with a generated passphrase
  3. Splits the passphrase into shares (one per friend)
  4. Verifies the shares can reconstruct the passphrase
  5. Generates ZIP bundles for distribution
  6. Writes checksums to project.yml

Run this command inside a project directory (created with 'rememory init').`,
	RunE: runSeal,
}

func init() {
	rootCmd.AddCommand(sealCmd)
}

func runSeal(cmd *cobra.Command, args []string) error {
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

	// Archive the manifest directory
	var archiveBuf bytes.Buffer
	archiveResult, err := manifest.Archive(&archiveBuf, manifestDir)
	if err != nil {
		return fmt.Errorf("archiving manifest: %w", err)
	}

	// Warn about any skipped files (symlinks, etc.)
	for _, warning := range archiveResult.Warnings {
		fmt.Printf("  Warning: %s\n", warning)
	}

	// Generate passphrase
	passphrase, err := crypto.GeneratePassphrase(crypto.DefaultPassphraseBytes)
	if err != nil {
		return fmt.Errorf("generating passphrase: %w", err)
	}

	fmt.Println("Encrypting with age...")

	// Encrypt the archive
	var encryptedBuf bytes.Buffer
	archiveReader := bytes.NewReader(archiveBuf.Bytes())
	if err := crypto.Encrypt(&encryptedBuf, archiveReader, passphrase); err != nil {
		return fmt.Errorf("encrypting: %w", err)
	}

	// Create output directories
	sharesDir := p.SharesPath()
	if err := os.MkdirAll(sharesDir, 0755); err != nil {
		return fmt.Errorf("creating output directories: %w", err)
	}

	// Write encrypted manifest
	manifestAgePath := p.ManifestAgePath()
	if err := os.WriteFile(manifestAgePath, encryptedBuf.Bytes(), 0644); err != nil {
		return fmt.Errorf("writing encrypted manifest: %w", err)
	}

	fmt.Printf("Splitting into %d shares (threshold: %d)...\n", len(p.Friends), p.Threshold)

	// Split the passphrase
	shares, err := shamir.Split([]byte(passphrase), len(p.Friends), p.Threshold)
	if err != nil {
		return fmt.Errorf("splitting passphrase: %w", err)
	}

	// Create share files
	shareInfos := make([]project.ShareInfo, len(shares))
	for i, shareData := range shares {
		friend := p.Friends[i]
		share := shamir.NewShare(i+1, len(p.Friends), p.Threshold, friend.Name, shareData)

		filename := share.Filename()
		sharePath := filepath.Join(sharesDir, filename)

		if err := os.WriteFile(sharePath, []byte(share.Encode()), 0644); err != nil {
			return fmt.Errorf("writing share for %s: %w", friend.Name, err)
		}

		// Compute checksum of the share file
		fileChecksum, err := crypto.HashFile(sharePath)
		if err != nil {
			return fmt.Errorf("computing checksum: %w", err)
		}

		relPath, _ := filepath.Rel(p.Path, sharePath)
		shareInfos[i] = project.ShareInfo{
			Friend:   friend.Name,
			File:     relPath,
			Checksum: fileChecksum,
		}
	}

	// Verify reconstruction
	fmt.Print("Verifying reconstruction... ")
	testShares := make([][]byte, p.Threshold)
	for i := 0; i < p.Threshold; i++ {
		testShares[i] = shares[i]
	}
	recovered, err := shamir.Combine(testShares)
	if err != nil {
		fmt.Println("FAILED")
		return fmt.Errorf("verification failed: %w", err)
	}
	if string(recovered) != passphrase {
		fmt.Println("FAILED")
		return fmt.Errorf("verification failed: reconstructed passphrase doesn't match")
	}
	fmt.Println("OK")

	// Update project with seal information
	manifestChecksum, err := crypto.HashFile(manifestAgePath)
	if err != nil {
		return fmt.Errorf("computing manifest checksum: %w", err)
	}

	p.Sealed = &project.Sealed{
		At:               time.Now().UTC(),
		ManifestChecksum: manifestChecksum,
		VerificationHash: crypto.HashString(passphrase),
		Shares:           shareInfos,
	}

	if err := p.Save(); err != nil {
		return fmt.Errorf("saving project: %w", err)
	}

	// Print seal summary
	fmt.Println()
	fmt.Println("Sealed:")
	relManifest, _ := filepath.Rel(p.Path, manifestAgePath)
	fmt.Printf("  %s %s\n", green("✓"), relManifest)
	for _, si := range shareInfos {
		fmt.Printf("  %s %s\n", green("✓"), si.File)
	}

	// Generate bundles automatically
	fmt.Println()
	fmt.Printf("Generating bundles for %d friends...\n", len(p.Friends))

	wasmBytes := html.GetWASMBytes()
	if len(wasmBytes) == 0 {
		return fmt.Errorf("WASM binary not embedded - rebuild with 'make build'")
	}

	cfg := bundle.Config{
		Version:          version,
		GitHubReleaseURL: fmt.Sprintf("https://github.com/eljojo/rememory/releases/tag/%s", version),
		WASMBytes:        wasmBytes,
	}

	if err := bundle.GenerateAll(p, cfg); err != nil {
		return fmt.Errorf("generating bundles: %w", err)
	}

	// Print bundle summary
	bundlesDir := filepath.Join(p.OutputPath(), "bundles")
	entries, _ := os.ReadDir(bundlesDir)

	fmt.Println()
	fmt.Println("Bundles ready to distribute:")
	for _, entry := range entries {
		if !entry.IsDir() {
			info, _ := entry.Info()
			fmt.Printf("  %s %s (%s)\n", green("✓"), entry.Name(), formatSize(info.Size()))
		}
	}

	fmt.Printf("\nSaved to: %s\n", bundlesDir)

	return nil
}

func formatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func truncateHash(hash string) string {
	// sha256:abc123... -> sha256:abc123...
	if len(hash) > 20 {
		return hash[:20] + "..."
	}
	return hash
}
