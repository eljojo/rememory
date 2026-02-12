package cmd

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/eljojo/rememory/internal/core"
	"github.com/eljojo/rememory/internal/html"
	"github.com/eljojo/rememory/internal/manifest"
	"github.com/spf13/cobra"
)

var recoverCmd = &cobra.Command{
	Use:   "recover share1.txt share2.txt ... [--manifest MANIFEST.age]",
	Short: "Recover the manifest from shares",
	Long: `Recover reconstructs the passphrase from shares and decrypts the manifest.

This command can be run from anywhere (doesn't need a project directory).
You need at least the threshold number of shares to recover.

Example:
  rememory recover SHARE-alice.txt SHARE-bob.txt SHARE-carol.txt -m MANIFEST.age`,
	Args: cobra.MinimumNArgs(1),
	RunE: runRecover,
}

var (
	recoverManifest   string
	recoverOutput     string
	recoverPassphrase bool
)

func init() {
	rootCmd.AddCommand(recoverCmd)
	recoverCmd.Flags().StringVarP(&recoverManifest, "manifest", "m", "", "Path to MANIFEST.age file")
	recoverCmd.Flags().StringVarP(&recoverOutput, "output", "o", "", "Output directory (default: recovered-TIMESTAMP)")
	recoverCmd.Flags().BoolVar(&recoverPassphrase, "passphrase-only", false, "Only output the passphrase, don't decrypt")
}

func runRecover(cmd *cobra.Command, args []string) error {
	// Parse all share files
	fmt.Printf("Reading %d share files...\n", len(args))

	shares := make([]*core.Share, len(args))
	for i, path := range args {
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

	if recoverPassphrase {
		fmt.Println()
		fmt.Println("Recovered passphrase:")
		fmt.Println(passphrase)
		return nil
	}

	// Find manifest file
	manifestPath := recoverManifest
	if manifestPath == "" {
		// Try to find MANIFEST.age in current directory, then recover.html
		if _, err := os.Stat("MANIFEST.age"); err == nil {
			manifestPath = "MANIFEST.age"
		} else if _, err := os.Stat("recover.html"); err == nil {
			manifestPath = "recover.html"
		} else {
			return fmt.Errorf("MANIFEST.age not found in current directory; use --manifest to specify path\n  (you can also pass a personalized recover.html file)")
		}
	}

	fmt.Println("Decrypting manifest...")

	// Read manifest data — either directly from .age file or extracted from .html
	var encryptedData []byte
	if strings.HasSuffix(strings.ToLower(manifestPath), ".html") || strings.HasSuffix(strings.ToLower(manifestPath), ".htm") {
		htmlContent, err := os.ReadFile(manifestPath)
		if err != nil {
			return fmt.Errorf("reading %s: %w", manifestPath, err)
		}
		encryptedData, err = html.ExtractManifestFromHTML(htmlContent)
		if err != nil {
			return fmt.Errorf("extracting manifest from %s: %w", manifestPath, err)
		}
		fmt.Printf("Extracted manifest from %s\n", manifestPath)
	} else {
		encryptedData, err = os.ReadFile(manifestPath)
		if err != nil {
			return fmt.Errorf("reading manifest: %w", err)
		}
	}

	var decryptedBuf bytes.Buffer
	if err := core.Decrypt(&decryptedBuf, bytes.NewReader(encryptedData), passphrase); err != nil {
		return fmt.Errorf("decryption failed (shares may be corrupted or from different operation): %w", err)
	}

	// Determine output directory
	outputDir := recoverOutput
	if outputDir == "" {
		outputDir = fmt.Sprintf("recovered-%s", time.Now().Format("2006-01-02"))
	}

	// Extract archive
	extractResult, err := manifest.Extract(&decryptedBuf, outputDir)
	if err != nil {
		return fmt.Errorf("extracting manifest: %w", err)
	}

	// Warn about any skipped files (symlinks, etc.)
	for _, warning := range extractResult.Warnings {
		fmt.Printf("  Warning: %s\n", warning)
	}

	// List recovered files
	fmt.Println()
	fmt.Printf("Recovered to: %s/\n", extractResult.Path)

	err = filepath.Walk(extractResult.Path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if path == extractResult.Path {
			return nil
		}
		relPath, _ := filepath.Rel(extractResult.Path, path)
		if info.IsDir() {
			fmt.Printf("  %s/\n", relPath)
		} else {
			fmt.Printf("  %s\n", relPath)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("listing recovered files: %w", err)
	}

	return nil
}
