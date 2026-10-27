package cmd

import (
	"fmt"

	"github.com/eljojo/rememory/internal/bundle"
	"github.com/spf13/cobra"
)

var verifyBundleCmd = &cobra.Command{
	Use:   "verify-bundle <bundle.zip>",
	Short: "Verify the integrity of a bundle ZIP file",
	Long: `Verify-bundle checks that a distribution bundle is valid and intact.

This command verifies:
  - All required files are present (README.txt, README.pdf, MANIFEST.age, recover.html)
  - Checksums match the values embedded in README.txt
  - The embedded share is valid and parseable

Use this to verify bundles before distributing them, or to check bundles
you've received from others for damage. The checksums travel inside the
same ZIP, so they catch accidental corruption, not deliberate changes.`,
	Args: cobra.ExactArgs(1),
	RunE: runVerifyBundle,
}

func init() {
	rootCmd.AddCommand(verifyBundleCmd)
}

func runVerifyBundle(cmd *cobra.Command, args []string) error {
	bundlePath := args[0]

	fmt.Printf("Verifying bundle: %s\n", bundlePath)

	if err := bundle.VerifyBundle(bundlePath); err != nil {
		return fmt.Errorf("verification failed: %w", err)
	}

	fmt.Println("Bundle verified successfully.")
	return nil
}
