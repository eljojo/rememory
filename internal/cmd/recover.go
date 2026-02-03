package cmd

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/eljojo/rememory/internal/crypto"
	"github.com/eljojo/rememory/internal/manifest"
	"github.com/eljojo/rememory/internal/shamir"
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

	shares := make([]*shamir.Share, len(args))
	for i, path := range args {
		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("reading share %s: %w", path, err)
		}

		share, err := shamir.ParseShare(content)
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
	passphrase, err := shamir.Combine(shareData)
	if err != nil {
		return fmt.Errorf("combining shares: %w", err)
	}

	if recoverPassphrase {
		fmt.Println()
		fmt.Println("Recovered passphrase:")
		fmt.Println(string(passphrase))
		return nil
	}

	// Find manifest file
	manifestPath := recoverManifest
	if manifestPath == "" {
		// Try to find MANIFEST.age in current directory
		manifestPath = "MANIFEST.age"
		if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
			return fmt.Errorf("MANIFEST.age not found in current directory; use --manifest to specify path")
		}
	}

	fmt.Println("Decrypting manifest...")

	// Read and decrypt manifest
	encryptedData, err := os.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("reading manifest: %w", err)
	}

	var decryptedBuf bytes.Buffer
	if err := crypto.Decrypt(&decryptedBuf, bytes.NewReader(encryptedData), string(passphrase)); err != nil {
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
