package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/eljojo/rememory/internal/bundle"
	"github.com/eljojo/rememory/internal/html"
	"github.com/eljojo/rememory/internal/project"
	"github.com/spf13/cobra"
)

var bundleCmd = &cobra.Command{
	Use:   "bundle",
	Short: "Generate distribution bundles for all friends",
	Long: `Creates a ZIP bundle for each friend containing:
  - README.txt (with embedded share, contacts, instructions)
  - MANIFEST.age (encrypted payload)
  - recover.html (browser-based recovery tool)

Each bundle is self-contained and can be distributed to the respective friend.`,
	RunE: runBundle,
}

func init() {
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

	// Get embedded WASM binary
	wasmBytes := html.GetWASMBytes()
	if len(wasmBytes) == 0 {
		return fmt.Errorf("WASM binary not embedded - rebuild with 'make build'")
	}

	// Generate bundles
	fmt.Printf("Generating bundles for %d friends...\n\n", len(p.Friends))

	cfg := bundle.Config{
		Version:          version,
		GitHubReleaseURL: fmt.Sprintf("https://github.com/eljojo/rememory/releases/tag/%s", version),
		WASMBytes:        wasmBytes,
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
