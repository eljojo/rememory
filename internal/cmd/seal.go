package cmd

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/eljojo/rememory/internal/bundle"
	"github.com/eljojo/rememory/internal/core"
	"github.com/eljojo/rememory/internal/crypto"
	"github.com/eljojo/rememory/internal/manifest"
	"github.com/eljojo/rememory/internal/project"
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
	sealCmd.Flags().String("recovery-url", core.DefaultRecoveryURL, "Base URL for QR code in PDF")
	sealCmd.Flags().Bool("no-embed-manifest", false, "Do not embed MANIFEST.age in recover.html (it is embedded by default when 10 MB or less)")
	sealCmd.Flags().String("timelock", "", "Time-lock duration or date (e.g., 5min, 30d, 6m, 1y, 2027-06-15T00:00:00Z)")
	sealCmd.Flags().Bool("pages", false, "Generate a static pages directory (recover.html + MANIFEST.age) for hosting")
	sealCmd.Flags().Bool("message", false, "Print distribution messages for each friend after bundling")
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

	recoveryURL, _ := cmd.Flags().GetString("recovery-url")
	noEmbedManifest, _ := cmd.Flags().GetBool("no-embed-manifest")
	timelockStr, _ := cmd.Flags().GetString("timelock")
	pages, _ := cmd.Flags().GetBool("pages")

	if err := sealProject(p, recoveryURL, noEmbedManifest, timelockStr); err != nil {
		return err
	}

	bundlesDir := filepath.Join(p.OutputPath(), "bundles")
	fmt.Printf("\nSaved to: %s\n", bundlesDir)

	if pages {
		if err := generatePages(p); err != nil {
			return err
		}
	}

	showMessage, _ := cmd.Flags().GetBool("message")
	if showMessage {
		fmt.Println()
		if err := printMessages(p, false, ""); err != nil {
			return fmt.Errorf("generating messages: %w", err)
		}
	}

	return nil
}

// sealProject archives, encrypts, splits, verifies, saves, and generates bundles
// for an already-loaded project. Both runSeal and runDemo share this logic.
// recoveryURL is the base URL for QR codes in the PDF. If empty, the PDF defaults to the production URL.
// noEmbedManifest controls whether MANIFEST.age is embedded in recover.html.
// timelockStr is an optional time-lock duration or date (e.g., "5min", "30d", "1y", "2027-06-15T00:00:00Z").
func sealProject(p *project.Project, recoveryURL string, noEmbedManifest bool, timelockStr string) error {
	// Check manifest directory exists and has content
	manifestDir := p.ManifestPath()
	fileCount, err := manifest.CountFiles(manifestDir)
	if err != nil {
		return fmt.Errorf("checking manifest directory: %w", err)
	}
	if fileCount == 0 {
		return fmt.Errorf("manifest directory is empty: %s", manifestDir)
	}

	// Detect untouched template: exactly one file in the manifest tree,
	// a README.md exists at the root, and its contents match the rendered template.
	if fileCount == 1 {
		readmePath := filepath.Join(manifestDir, "README.md")
		actual, err := os.ReadFile(readmePath)
		if err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("reading manifest file: %w", err)
		}
		if err == nil {
			td := project.TemplateDataFromProject(p)
			expected, err := project.RenderManifestReadme(td)
			if err != nil {
				return fmt.Errorf("rendering manifest template: %w", err)
			}
			if bytes.Equal(actual, expected) {
				return fmt.Errorf("manifest contains only the template README — add your files to %s before sealing", manifestDir)
			}
		}
	}

	dirSize, err := manifest.DirSize(manifestDir)
	if err != nil {
		return fmt.Errorf("calculating manifest size: %w", err)
	}

	fmt.Printf("Archiving manifest/ (%d files, %s)...\n", fileCount, formatSize(dirSize))

	// Archive the manifest directory
	var archiveBuf bytes.Buffer
	archiveResult, err := manifest.ArchiveZip(&archiveBuf, manifestDir)
	if err != nil {
		return fmt.Errorf("archiving manifest: %w", err)
	}

	for _, warning := range archiveResult.Warnings {
		fmt.Printf("  Warning: %s\n", warning)
	}

	// Parse timelock if specified
	var tlockEnabled bool
	var tlockRound uint64
	var tlockUnlockTime time.Time
	if timelockStr != "" {
		var err error
		tlockUnlockTime, err = core.ParseTimelockValue(timelockStr)
		if err != nil {
			return fmt.Errorf("parsing timelock: %w", err)
		}
		tlockRound = core.RoundForTime(tlockUnlockTime)
		tlockEnabled = true
	}

	// Generate passphrase (v2: split raw bytes, not the base64 string)
	raw, passphrase, err := crypto.GenerateRawPassphrase(crypto.DefaultPassphraseBytes)
	if err != nil {
		return fmt.Errorf("generating passphrase: %w", err)
	}

	// The encryption order: tlock first (inner), then age (outer).
	// This means age protects tlock — if tlock's pairing math weakens,
	// age's conservative primitives still guard the data.
	dataToEncrypt := archiveBuf.Bytes()

	if tlockEnabled {
		fmt.Printf("Time-lock encrypting (unlocks %s, drand round %d)...\n",
			tlockUnlockTime.Format("2006-01-02"), tlockRound)

		var tlockBuf bytes.Buffer
		if err := core.TlockEncrypt(&tlockBuf, bytes.NewReader(dataToEncrypt), tlockRound); err != nil {
			return fmt.Errorf("tlock encrypting: %w", err)
		}

		// Build tlock container: ZIP with tlock.json + manifest.tlock.age
		meta := &core.TlockMeta{
			V:      core.TlockContainerVersion,
			Method: core.TlockMethodQuicknet,
			Round:  tlockRound,
			Unlock: tlockUnlockTime.Format(time.RFC3339),
			Chain:  core.QuicknetChainHash,
		}
		container, err := core.BuildTlockContainer(meta, tlockBuf.Bytes())
		if err != nil {
			return fmt.Errorf("building tlock container: %w", err)
		}
		dataToEncrypt = container
	}

	fmt.Println("Encrypting with age...")

	// Encrypt with age (outer layer) — always a plain age file
	var encryptedBuf bytes.Buffer
	if err := core.Encrypt(&encryptedBuf, bytes.NewReader(dataToEncrypt), passphrase); err != nil {
		return fmt.Errorf("encrypting: %w", err)
	}

	manifestData := encryptedBuf.Bytes()

	// Create output directories
	sharesDir := p.SharesPath()
	if err := os.MkdirAll(sharesDir, 0755); err != nil {
		return fmt.Errorf("creating output directories: %w", err)
	}

	// Write encrypted manifest
	manifestAgePath := p.ManifestAgePath()
	if err := os.WriteFile(manifestAgePath, manifestData, 0644); err != nil {
		return fmt.Errorf("writing encrypted manifest: %w", err)
	}

	fmt.Printf("Splitting into %d shares (threshold: %d)...\n", len(p.Friends), p.Threshold)

	// Split the raw bytes (v2: 32 bytes instead of 43-byte base64 string)
	shares, err := core.Split(raw, len(p.Friends), p.Threshold)
	if err != nil {
		return fmt.Errorf("splitting passphrase: %w", err)
	}

	// Create share files
	shareInfos := make([]project.ShareInfo, len(shares))
	for i, shareData := range shares {
		friend := p.Friends[i]
		share := core.NewShare(2, i+1, len(p.Friends), p.Threshold, friend.Name, shareData)

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
	if base64.RawURLEncoding.EncodeToString(recovered) != passphrase {
		fmt.Println("FAILED")
		return fmt.Errorf("verification failed: reconstructed passphrase doesn't match")
	}
	fmt.Println("OK")

	// Update project with seal information
	manifestChecksum, err := crypto.HashFile(manifestAgePath)
	if err != nil {
		return fmt.Errorf("computing manifest checksum: %w", err)
	}

	sealed := &project.Sealed{
		At:               time.Now().UTC(),
		ManifestChecksum: manifestChecksum,
		VerificationHash: core.HashString(passphrase),
		Shares:           shareInfos,
	}
	if tlockEnabled {
		sealed.TlockEnabled = true
		sealed.TlockRound = tlockRound
		unlockUTC := tlockUnlockTime.UTC()
		sealed.TlockUnlockTime = &unlockUTC
	}
	p.Sealed = sealed

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

	// Generate bundles
	fmt.Println()
	fmt.Printf("Generating bundles for %d friends...\n", len(p.Friends))

	cfg := bundle.Config{
		Version:         version,
		RecoveryURL:     recoveryURL,
		NoEmbedManifest: noEmbedManifest,
		TlockEnabled:    tlockEnabled,
	}

	if err := bundle.GenerateAll(p, cfg); err != nil {
		return fmt.Errorf("generating bundles: %w", err)
	}

	// Print bundle listing
	bundlesDir := filepath.Join(p.OutputPath(), "bundles")
	entries, _ := os.ReadDir(bundlesDir)

	fmt.Println()
	fmt.Println("Bundles ready:")
	for _, entry := range entries {
		if !entry.IsDir() {
			info, _ := entry.Info()
			fmt.Printf("  %s %s (%s)\n", green("✓"), entry.Name(), formatSize(info.Size()))
		}
	}

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
