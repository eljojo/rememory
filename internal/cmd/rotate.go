package cmd

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/eljojo/rememory/internal/bundle"
	"github.com/eljojo/rememory/internal/core"
	"github.com/eljojo/rememory/internal/crypto"
	"github.com/eljojo/rememory/internal/html"
	"github.com/eljojo/rememory/internal/manifest"
	"github.com/eljojo/rememory/internal/project"
	"github.com/spf13/cobra"
)

var rotateCmd = &cobra.Command{
	Use:   "rotate",
	Short: "Re-encrypt and re-split with a new passphrase, invalidating all existing shares",
	Long: `Rotate generates a new passphrase, re-encrypts the manifest, and creates new shares for all friends.

After rotation:
  - All old shares become USELESS (they reconstruct the old passphrase, which no longer decrypts anything)
  - You must distribute new bundles to all friends
  - Friends should securely destroy their old bundles

Rotation does NOT change the list of friends or the threshold. It uses whatever is currently
in project.yml. To change friends, edit project.yml before rotating.

When to rotate:
  - Every 2-3 years as a security hygiene practice
  - After a friend loses their share (re-split makes old partial knowledge useless)
  - After you suspect a share may be compromised
  - When you've updated the manifest contents`,
	RunE: runRotate,
}

var rotateForce bool
var rotateThreshold int

func init() {
	rootCmd.AddCommand(rotateCmd)
	rotateCmd.Flags().BoolVarP(&rotateForce, "yes", "y", false, "Skip confirmation prompt")
	rotateCmd.Flags().IntVar(&rotateThreshold, "threshold", 0, "New threshold (default: keep current)")
}

func runRotate(cmd *cobra.Command, args []string) error {
	// Find and load the project
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting current directory: %w", err)
	}

	projectDir, err := project.FindProjectDir(cwd)
	if err != nil {
		return fmt.Errorf("no rememory project found (run 'rememory init' first)")
	}

	p, err := project.Load(projectDir)
	if err != nil {
		return fmt.Errorf("loading project: %w", err)
	}

	if err := p.Validate(); err != nil {
		return fmt.Errorf("invalid project: %w", err)
	}

	// Require an existing sealed manifest to rotate from
	if p.Sealed == nil {
		return fmt.Errorf("project is not sealed yet — run 'rememory seal' first")
	}

	// Apply threshold override if requested
	if rotateThreshold > 0 {
		if rotateThreshold < 2 {
			return fmt.Errorf("threshold must be at least 2")
		}
		if rotateThreshold > len(p.Friends) {
			return fmt.Errorf("threshold (%d) cannot exceed number of friends (%d)", rotateThreshold, len(p.Friends))
		}
		p.Threshold = rotateThreshold
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

	// Print what's about to happen
	age := time.Since(p.Sealed.At)
	fmt.Println()
	fmt.Printf("  Project:   %s\n", p.Name)
	fmt.Printf("  Sealed:    %s ago (%s)\n", formatDuration(age), p.Sealed.At.Format("2006-01-02"))
	fmt.Printf("  Friends:   %d\n", len(p.Friends))
	fmt.Printf("  Threshold: %d of %d\n", p.Threshold, len(p.Friends))
	fmt.Println()
	fmt.Println(yellow("WARNING: Rotation invalidates ALL existing shares and bundles."))
	fmt.Println(yellow("You must distribute new bundles and ask friends to destroy their old ones."))
	fmt.Println()

	// Confirm unless --yes
	if !rotateForce {
		fmt.Print("Type 'rotate' to confirm: ")
		scanner := bufio.NewScanner(os.Stdin)
		if !scanner.Scan() {
			return fmt.Errorf("confirmation cancelled")
		}
		input := strings.TrimSpace(scanner.Text())
		if input != "rotate" {
			return fmt.Errorf("rotation cancelled")
		}
		fmt.Println()
	}

	// Archive old output before overwriting
	oldOutputDir := p.OutputPath()
	if _, err := os.Stat(oldOutputDir); err == nil {
		timestamp := time.Now().UTC().Format("20060102-150405")
		archiveDir := filepath.Join(p.Path, fmt.Sprintf("output-archived-%s", timestamp))
		if err := os.Rename(oldOutputDir, archiveDir); err != nil {
			return fmt.Errorf("archiving old output: %w", err)
		}
		fmt.Printf("Old shares/bundles archived to: %s\n", filepath.Base(archiveDir))
		fmt.Println("(You should securely delete this directory after distributing new bundles.)")
		fmt.Println()
	}

	// Archive the manifest directory
	dirSize, _ := manifest.DirSize(manifestDir)
	fmt.Printf("Archiving manifest/ (%d files, %s)...\n", fileCount, formatSize(dirSize))

	var archiveBuf bytes.Buffer
	archiveResult, err := manifest.Archive(&archiveBuf, manifestDir)
	if err != nil {
		return fmt.Errorf("archiving manifest: %w", err)
	}

	for _, warning := range archiveResult.Warnings {
		fmt.Printf("  Warning: %s\n", warning)
	}

	// Generate new passphrase
	passphrase, err := crypto.GeneratePassphrase(crypto.DefaultPassphraseBytes)
	if err != nil {
		return fmt.Errorf("generating passphrase: %w", err)
	}

	fmt.Println("Encrypting with age...")

	// Encrypt the archive
	var encryptedBuf bytes.Buffer
	archiveReader := bytes.NewReader(archiveBuf.Bytes())
	if err := core.Encrypt(&encryptedBuf, archiveReader, passphrase); err != nil {
		return fmt.Errorf("encrypting: %w", err)
	}

	// Create output directories
	sharesDir := p.SharesPath()
	if err := os.MkdirAll(sharesDir, 0755); err != nil {
		return fmt.Errorf("creating output directories: %w", err)
	}

	// Write new encrypted manifest
	manifestAgePath := p.ManifestAgePath()
	if err := os.WriteFile(manifestAgePath, encryptedBuf.Bytes(), 0644); err != nil {
		return fmt.Errorf("writing encrypted manifest: %w", err)
	}

	fmt.Printf("Splitting into %d shares (threshold: %d)...\n", len(p.Friends), p.Threshold)

	// Split the new passphrase
	shares, err := core.Split([]byte(passphrase), len(p.Friends), p.Threshold)
	if err != nil {
		return fmt.Errorf("splitting passphrase: %w", err)
	}

	// Write share files
	shareInfos := make([]project.ShareInfo, len(shares))
	for i, shareData := range shares {
		friend := p.Friends[i]
		share := core.NewShare(i+1, len(p.Friends), p.Threshold, friend.Name, shareData)

		filename := share.Filename()
		sharePath := filepath.Join(sharesDir, filename)

		if err := os.WriteFile(sharePath, []byte(share.Encode()), 0600); err != nil {
			return fmt.Errorf("writing share for %s: %w", friend.Name, err)
		}

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
	recovered, err := core.Combine(testShares)
	if err != nil {
		fmt.Println("FAILED")
		return fmt.Errorf("verification failed: %w", err)
	}
	if string(recovered) != passphrase {
		fmt.Println("FAILED")
		return fmt.Errorf("verification failed: reconstructed passphrase doesn't match")
	}
	fmt.Println("OK")

	// Update project with new seal information
	manifestChecksum, err := crypto.HashFile(manifestAgePath)
	if err != nil {
		return fmt.Errorf("computing manifest checksum: %w", err)
	}

	p.Sealed = &project.Sealed{
		At:               time.Now().UTC(),
		ManifestChecksum: manifestChecksum,
		VerificationHash: core.HashString(passphrase),
		Shares:           shareInfos,
	}

	if err := p.Save(); err != nil {
		return fmt.Errorf("saving project: %w", err)
	}

	// Print seal summary
	fmt.Println()
	fmt.Println("New seal:")
	relManifest, _ := filepath.Rel(p.Path, manifestAgePath)
	fmt.Printf("  %s %s\n", green("✓"), relManifest)
	for _, si := range shareInfos {
		fmt.Printf("  %s %s\n", green("✓"), si.File)
	}

	// Generate new bundles
	fmt.Println()
	fmt.Printf("Generating new bundles for %d friends...\n", len(p.Friends))

	wasmBytes := html.GetRecoverWASMBytes()
	if len(wasmBytes) == 0 {
		return fmt.Errorf("recover.wasm not embedded - rebuild with 'make build'")
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
	fmt.Println("New bundles ready to distribute:")
	for _, entry := range entries {
		if !entry.IsDir() {
			info, _ := entry.Info()
			fmt.Printf("  %s %s (%s)\n", green("✓"), entry.Name(), formatSize(info.Size()))
		}
	}
	fmt.Printf("\nSaved to: %s\n", bundlesDir)

	// Final checklist
	fmt.Println()
	fmt.Println("─────────────────────────────────────────────────")
	fmt.Println(green("Rotation complete. Next steps:"))
	fmt.Println()
	fmt.Println("  1. Distribute the new bundles to each friend")
	fmt.Println("  2. Ask friends to securely delete their old bundle")
	fmt.Printf("  3. Securely delete the archived old output:\n")
	oldArchiveGlob := filepath.Join(p.Path, "output-archived-*")
	oldArchives, _ := filepath.Glob(oldArchiveGlob)
	for _, a := range oldArchives {
		fmt.Printf("       rm -rf %s\n", a)
	}
	fmt.Println("  4. Verify the new bundles: rememory verify")
	fmt.Println("─────────────────────────────────────────────────")

	return nil
}
