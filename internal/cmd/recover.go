package cmd

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/eljojo/rememory/internal/bundle"
	"github.com/eljojo/rememory/internal/core"
	"github.com/eljojo/rememory/internal/html"
	"github.com/eljojo/rememory/internal/manifest"
	"github.com/spf13/cobra"
)

var recoverCmd = &cobra.Command{
	Use:   "recover [share1.txt|bundle1.zip] [share2.txt|bundle2.zip] ... [--manifest MANIFEST.age]",
	Short: "Recover the manifest from shares",
	Long: `Recover reconstructs the passphrase from shares and decrypts the manifest.

This command can be run from anywhere (doesn't need a project directory).
You need at least the threshold number of shares to recover.

Shares can be plain text files, bundle ZIP files, or personalized
recover.html files. The manifest is extracted from the first ZIP or
HTML that contains one, unless --manifest is set.

Example:
  rememory recover bundle-alice.zip bundle-bob.zip
  rememory recover alice/recover.html bob/recover.html carol/recover.html
  rememory recover SHARE-alice.txt SHARE-bob.txt -m MANIFEST.age`,
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
	recoverCmd.Flags().StringVarP(&recoverManifest, "manifest", "m", "", "Path to MANIFEST.age file (or bundle .zip)")
	recoverCmd.Flags().StringVarP(&recoverOutput, "output", "o", "", "Output directory (default: recovered-TIMESTAMP)")
	recoverCmd.Flags().BoolVar(&recoverPassphrase, "passphrase-only", false, "Only output the passphrase, don't decrypt")
}

func runRecover(cmd *cobra.Command, args []string) error {
	// Parse all share files
	fmt.Printf("Reading %d share files...\n", len(args))

	shares := make([]*core.Share, len(args))
	for i, path := range args {
		var share *core.Share
		var err error

		if isZipFile(path) {
			share, err = bundle.ExtractShareFromZip(path)
			if err != nil {
				return fmt.Errorf("extracting share from %s: %w", path, err)
			}
		} else if isHTMLFile(path) {
			content, err := os.ReadFile(path)
			if err != nil {
				return fmt.Errorf("reading %s: %w", path, err)
			}
			share, err = html.ExtractShareFromHTML(content)
			if err != nil {
				return fmt.Errorf("extracting share from %s: %w", path, err)
			}
		} else {
			content, err := os.ReadFile(path)
			if err != nil {
				return fmt.Errorf("reading share %s: %w", path, err)
			}

			share, err = core.ParseShare(content)
			if err != nil {
				return fmt.Errorf("parsing share %s: %w", path, err)
			}
		}

		// Verify checksum
		if err := share.Verify(); err != nil {
			return fmt.Errorf("share %s: %w", path, err)
		}

		shares[i] = share
	}

	// Validate shares are compatible
	if err := validateSharesCompatible(shares); err != nil {
		return err
	}
	if warning := createdTimeWarning(shares); warning != "" {
		fmt.Printf("Warning: %s\n", warning)
	}

	first := shares[0]

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

	// Find manifest
	var encryptedData []byte
	manifestPath := recoverManifest

	if manifestPath != "" && isZipFile(manifestPath) {
		// --manifest points to a ZIP — extract from it
		encryptedData, err = bundle.ExtractManifestFromZip(manifestPath)
		if err != nil {
			return fmt.Errorf("extracting manifest from %s: %w", manifestPath, err)
		}
		fmt.Printf("Extracted manifest from %s\n", manifestPath)
	} else if manifestPath == "" {
		// No --manifest flag — try extracting from the provided args
		for _, path := range args {
			if isZipFile(path) {
				data, err := bundle.ExtractManifestFromZip(path)
				if err == nil {
					encryptedData = data
					fmt.Printf("Extracted manifest from %s\n", path)
					break
				}
			} else if isHTMLFile(path) {
				content, err := os.ReadFile(path)
				if err == nil {
					data, err := html.ExtractManifestFromHTML(content)
					if err == nil {
						encryptedData = data
						fmt.Printf("Extracted manifest from %s\n", path)
						break
					}
				}
			}
		}

		// Fall back to searching the current directory
		if encryptedData == nil {
			if _, err := os.Stat("MANIFEST.age"); err == nil {
				manifestPath = "MANIFEST.age"
			} else if _, err := os.Stat("recover.html"); err == nil {
				manifestPath = "recover.html"
			} else {
				return fmt.Errorf("no manifest found — pass --manifest or include a bundle zip that contains one")
			}
		}
	}

	fmt.Println("Decrypting manifest...")

	// Read manifest from file if not already extracted from a ZIP
	if encryptedData == nil {
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
	}

	var decryptedBuf bytes.Buffer
	if err := core.Decrypt(&decryptedBuf, bytes.NewReader(encryptedData), passphrase); err != nil {
		return fmt.Errorf("decryption failed (shares may be corrupted or from different operation): %w", err)
	}

	decryptedData := decryptedBuf.Bytes()

	// Check for tlock container (time-lock encrypted archive)
	if core.IsTlockContainer(decryptedData) {
		meta, tlockCiphertext, err := core.OpenTlockContainer(decryptedData)
		if err != nil {
			return fmt.Errorf("reading tlock container: %w", err)
		}

		unlockTime, _ := meta.UnlockTime()
		if time.Now().Before(unlockTime) {
			return fmt.Errorf("this archive is time-locked until %s — try again after that date", unlockTime.Format("2006-01-02 15:04"))
		}

		fmt.Println("Opening time lock...")
		var tlockBuf bytes.Buffer
		if err := core.TlockDecrypt(&tlockBuf, bytes.NewReader(tlockCiphertext)); err != nil {
			return fmt.Errorf("time-lock decryption failed (may need internet connection): %w", err)
		}
		decryptedData = tlockBuf.Bytes()
	}

	// Determine output directory
	outputDir := recoverOutput
	if outputDir == "" {
		outputDir = fmt.Sprintf("recovered-%s", time.Now().Format("2006-01-02"))
	}

	// Extract archive (auto-detects ZIP or tar.gz)
	extractResult, err := manifest.ExtractAuto(bytes.NewReader(decryptedData), outputDir)
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

func validateSharesCompatible(shares []*core.Share) error {
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

	return nil
}

// createdTimeWarning describes pieces whose creation times differ, or
// returns "". It's a warning, not an error: older versions of seal stamped
// each piece separately, so pieces of one set could differ by a minute.
// Pieces from different seals fail to unlock anyway.
func createdTimeWarning(shares []*core.Share) string {
	for i, share := range shares {
		if share.Created.IsZero() {
			continue
		}
		for _, prev := range shares[:i] {
			if !prev.Created.IsZero() && !share.Created.Equal(prev.Created) {
				return fmt.Sprintf("share %d was created at a different time (%s, not %s); if these pieces are from different seals, unlocking will fail",
					i+1, share.Created.Format("2006-01-02 15:04"), prev.Created.Format("2006-01-02 15:04"))
			}
		}
	}
	return ""
}

func isZipFile(path string) bool {
	return strings.HasSuffix(strings.ToLower(path), ".zip")
}

func isHTMLFile(path string) bool {
	lower := strings.ToLower(path)
	return strings.HasSuffix(lower, ".html") || strings.HasSuffix(lower, ".htm")
}
