package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/eljojo/rememory/internal/core"
	"github.com/eljojo/rememory/internal/project"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show project status and summary",
	Long:  `Displays the current state of the rememory project including seal status, friends, and bundle information.`,
	RunE:  runStatus,
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

func runStatus(cmd *cobra.Command, args []string) error {
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

	// Print status
	fmt.Printf("Project: %s\n", p.Name)
	fmt.Printf("Path: %s\n\n", p.Path)

	// Sealed status
	if p.Sealed != nil {
		fmt.Printf("Sealed: %s (%s)\n", green("Yes"), p.Sealed.At.Format("2006-01-02 15:04:05 UTC"))
		fmt.Printf("Manifest Checksum: %s\n", truncateHash(p.Sealed.ManifestChecksum))
	} else {
		fmt.Printf("Sealed: %s\n", yellow("No"))
		fmt.Println("  Run 'rememory seal' to encrypt and split the passphrase")
	}

	// Threshold
	fmt.Printf("\nThreshold: %d of %d\n", p.Threshold, len(p.Friends))

	// Friends
	fmt.Println("\nShare holders:")
	for i, friend := range p.Friends {
		shareExists := checkShareExists(p, friend)
		status := green("✓")
		if !shareExists {
			status = yellow("○")
		}
		contactInfo := friend.Contact
		if contactInfo == "" {
			contactInfo = "no contact info"
		}
		fmt.Printf("  %d. %s %s (%s)\n", i+1, status, friend.Name, contactInfo)
	}

	// Bundles status
	bundlesDir := filepath.Join(p.OutputPath(), "bundles")
	bundleCount := countBundles(bundlesDir)
	fmt.Println()
	if bundleCount > 0 {
		fmt.Printf("Bundles: %s (%d bundles in %s)\n", green("Generated"), bundleCount, bundlesDir)
	} else if p.Sealed != nil {
		fmt.Printf("Bundles: %s\n", yellow("Not yet generated"))
		fmt.Println("  Run 'rememory bundle' to create distribution bundles")
	} else {
		fmt.Printf("Bundles: %s (seal first)\n", yellow("Not available"))
	}

	// Rotation reminder
	if p.Sealed != nil {
		age := time.Since(p.Sealed.At)
		fmt.Println()
		if age > 2*365*24*time.Hour { // 2 years
			fmt.Printf("Rotation: %s\n", yellow("Consider rotating - sealed over 2 years ago"))
		} else if age > 365*24*time.Hour { // 1 year
			fmt.Printf("Rotation: Last sealed %s ago\n", formatDuration(age))
		} else {
			fmt.Printf("Rotation: Last sealed %s ago (consider rotating every 2-3 years)\n", formatDuration(age))
		}
	}

	return nil
}

func checkShareExists(p *project.Project, friend project.Friend) bool {
	sharesDir := p.SharesPath()
	filename := fmt.Sprintf("SHARE-%s.txt", core.SanitizeFilename(friend.Name))
	_, err := os.Stat(filepath.Join(sharesDir, filename))
	return err == nil
}

func countBundles(dir string) int {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0
	}
	count := 0
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".zip" {
			count++
		}
	}
	return count
}

func formatDuration(d time.Duration) string {
	days := int(d.Hours() / 24)
	if days > 365 {
		years := days / 365
		return fmt.Sprintf("%d year%s", years, plural(years))
	}
	if days > 30 {
		months := days / 30
		return fmt.Sprintf("%d month%s", months, plural(months))
	}
	if days > 0 {
		return fmt.Sprintf("%d day%s", days, plural(days))
	}
	hours := int(d.Hours())
	if hours > 0 {
		return fmt.Sprintf("%d hour%s", hours, plural(hours))
	}
	return "just now"
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}
