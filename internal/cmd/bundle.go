package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/eljojo/rememory/internal/bundle"
	"github.com/eljojo/rememory/internal/core"
	"github.com/eljojo/rememory/internal/html"
	"github.com/eljojo/rememory/internal/project"
	"github.com/spf13/cobra"
)

var bundleCmd = &cobra.Command{
	Use:   "bundle",
	Short: "Regenerate distribution bundles for all friends",
	Long: `Regenerates ZIP bundles for each friend. This is useful if you:
  - Lost the original bundle files
  - Want to update bundles with a newer version of recover.html

Note: 'rememory seal' automatically generates bundles, so you typically
don't need to run this command separately.

Each bundle contains:
  - README.txt (with embedded share, contacts, instructions)
  - README.pdf (same content, formatted for printing)
  - MANIFEST.age (encrypted payload)
  - recover.html (browser-based recovery tool)`,
	RunE: runBundle,
}

func init() {
	bundleCmd.Flags().String("recovery-url", core.DefaultRecoveryURL, "Base URL for QR code in PDF")
	rootCmd.AddCommand(bundleCmd)
}

func runBundle(cmd *cobra.Command, args []string) error {
	// Find project
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting current directory: %w", err)
	}

	projectDir, err := project.FindProjectDir(cwd)
	if err != nil {
		return fmt.Errorf("no rememory project found (run 'rememory init' first)")
	}

	// Load project
	p, err := project.Load(projectDir)
	if err != nil {
		return fmt.Errorf("loading project: %w", err)
	}

	// Check if sealed
	if p.Sealed == nil {
		return fmt.Errorf("project must be sealed before generating bundles (run 'rememory seal' first)")
	}

	// Get embedded recovery WASM binary (smaller, for bundles)
	wasmBytes := html.GetRecoverWASMBytes()
	if len(wasmBytes) == 0 {
		return fmt.Errorf("recover.wasm not embedded - rebuild with 'make build'")
	}

	// Generate bundles
	fmt.Printf("Generating bundles for %d friends...\n\n", len(p.Friends))

	recoveryURL, _ := cmd.Flags().GetString("recovery-url")

	cfg := bundle.Config{
		Version:          version,
		GitHubReleaseURL: fmt.Sprintf("https://github.com/eljojo/rememory/releases/tag/%s", version),
		WASMBytes:        wasmBytes,
		RecoveryURL:      recoveryURL,
	}

	if err := bundle.GenerateAll(p, cfg); err != nil {
		return fmt.Errorf("generating bundles: %w", err)
	}

	// Print summary
	bundlesDir := filepath.Join(p.OutputPath(), "bundles")
	entries, _ := os.ReadDir(bundlesDir)

	fmt.Println("Created bundles:")
	for _, entry := range entries {
		if !entry.IsDir() {
			info, _ := entry.Info()
			fmt.Printf("  %s %s (%s)\n", green("✓"), entry.Name(), formatSize(info.Size()))
		}
	}

	fmt.Printf("\nBundles saved to: %s\n", bundlesDir)
	fmt.Println("\nNote: Each README contains the friend's share - remind them not to share it!")

	return nil
}
