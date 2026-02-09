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

var demoCmd = &cobra.Command{
	Use:   "demo [directory]",
	Short: "Create a demo project with sample data",
	Long: `Create a complete demo project with sample friends and secret files.

This is useful for testing the recovery workflow or demonstrating ReMemory.

The demo project includes:
  - 3 friends: Alice, Bob, and Carol
  - Threshold of 2 (any 2 friends can recover)
  - Sample secret files in the manifest
  - Fully sealed and bundled, ready to test

Example:
  rememory demo
  rememory demo my-demo-project`,
	Args: cobra.MaximumNArgs(1),
	RunE: runDemo,
}

func init() {
	rootCmd.AddCommand(demoCmd)
}

func runDemo(cmd *cobra.Command, args []string) error {
	// Determine project directory
	dirName := "demo-recovery"
	if len(args) > 0 {
		dirName = args[0]
	}

	dir, err := filepath.Abs(dirName)
	if err != nil {
		return fmt.Errorf("resolving path: %w", err)
	}

	// Check if directory already exists
	if _, err := os.Stat(dir); err == nil {
		return fmt.Errorf("directory already exists: %s", dir)
	}

	fmt.Printf("Creating demo project: %s/\n\n", dirName)

	// Demo friends
	friends := []project.Friend{
		{Name: "Alice", Contact: "alice@example.com"},
		{Name: "Bob", Contact: "bob@example.com"},
		{Name: "Carol", Contact: "carol@example.com"},
	}
	threshold := 2

	fmt.Printf("Friends: %s\n", friendNames(friends))
	fmt.Printf("Threshold: %d of %d\n\n", threshold, len(friends))

	// Create the project
	p, err := project.New(dir, "Demo Project", threshold, friends)
	if err != nil {
		return fmt.Errorf("creating project: %w", err)
	}

	// Write the manifest README template
	templateData := project.TemplateData{
		ProjectName: "Demo Project",
		Friends:     friends,
		Threshold:   threshold,
	}
	if err := project.WriteManifestReadme(p.ManifestPath(), templateData); err != nil {
		return fmt.Errorf("creating manifest README: %w", err)
	}

	// Add demo secret files
	manifestDir := p.ManifestPath()

	demoSecretContent := `# Demo Secret File

This is a demonstration of ReMemory's secret recovery system.

In a real scenario, this file might contain:
- Password manager recovery codes
- Cryptocurrency seed phrases
- Important account credentials
- Instructions for loved ones

Remember: This file will be encrypted and can only be recovered
when enough friends combine their shares.
`
	if err := os.WriteFile(filepath.Join(manifestDir, "demo-secret.txt"), []byte(demoSecretContent), 0600); err != nil {
		return fmt.Errorf("writing demo secret: %w", err)
	}

	passwordsContent := `# Example Passwords (DEMO ONLY)

Email: demo@example.com
Password: correct-horse-battery-staple

Bank PIN: 1234

WiFi Password: DemoNetwork2024

Note: In a real project, these would be your actual sensitive credentials.
`
	if err := os.WriteFile(filepath.Join(manifestDir, "passwords.txt"), []byte(passwordsContent), 0600); err != nil {
		return fmt.Errorf("writing passwords file: %w", err)
	}

	fmt.Println("Created demo files:")
	fmt.Printf("  %s manifest/demo-secret.txt\n", green("✓"))
	fmt.Printf("  %s manifest/passwords.txt\n", green("✓"))
	fmt.Println()

	// Now seal the project (inline from seal.go logic)
	fileCount, err := manifest.CountFiles(manifestDir)
	if err != nil {
		return fmt.Errorf("checking manifest directory: %w", err)
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
	if err := core.Encrypt(&encryptedBuf, archiveReader, passphrase); err != nil {
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
	shares, err := core.Split([]byte(passphrase), len(p.Friends), p.Threshold)
	if err != nil {
		return fmt.Errorf("splitting passphrase: %w", err)
	}

	// Create share files
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

	// Update project with seal information
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
	fmt.Println("Sealed:")
	relManifest, _ := filepath.Rel(p.Path, manifestAgePath)
	fmt.Printf("  %s %s\n", green("✓"), relManifest)
	for _, si := range shareInfos {
		fmt.Printf("  %s %s\n", green("✓"), si.File)
	}

	// Generate bundles
	fmt.Println()
	fmt.Printf("Generating bundles for %d friends...\n", len(p.Friends))

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
	fmt.Println("Bundles ready:")
	for _, entry := range entries {
		if !entry.IsDir() {
			info, _ := entry.Info()
			fmt.Printf("  %s %s (%s)\n", green("✓"), entry.Name(), formatSize(info.Size()))
		}
	}

	fmt.Println()
	fmt.Println("Demo project created successfully!")
	fmt.Println()
	fmt.Println("To test recovery:")
	fmt.Printf("  1. Open %s/output/bundles/bundle-alice.zip\n", dirName)
	fmt.Println("  2. Extract and open recover.html in a browser")
	fmt.Println("  3. Alice's share is pre-loaded, add Bob's or Carol's README.txt")
	fmt.Println("  4. Recovery will happen automatically!")

	return nil
}
