package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/eljojo/rememory/internal/core"
	"github.com/eljojo/rememory/internal/project"
	"github.com/spf13/cobra"
)

var messageCmd = &cobra.Command{
	Use:   "message",
	Short: "Generate messages to send with each friend's bundle",
	Long: `Generate personalized messages for each friend explaining their
bundle and what to do with it.

Use after sealing to help distribute bundles. The messages are designed
to be copied into any messaging app, email, or read aloud.

Use --rotation when distributing updated bundles after re-sealing.

Example:
  rememory message
  rememory message --rotation
  rememory message --friend Alice`,
	RunE: runMessage,
}

func init() {
	messageCmd.Flags().Bool("rotation", false, "Generate shorter messages for updated bundles (friends already know what this is)")
	messageCmd.Flags().String("friend", "", "Generate message for a single friend")
	rootCmd.AddCommand(messageCmd)
}

func runMessage(cmd *cobra.Command, args []string) error {
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

	if p.Sealed == nil {
		return fmt.Errorf("project is not sealed yet — run 'rememory seal' first")
	}

	rotation, _ := cmd.Flags().GetBool("rotation")
	friendFilter, _ := cmd.Flags().GetString("friend")

	return printMessages(p, rotation, friendFilter)
}

func printMessages(p *project.Project, rotation bool, friendFilter string) error {
	bundlesDir := filepath.Join(p.OutputPath(), "bundles")

	for _, friend := range p.Friends {
		if friendFilter != "" && !strings.EqualFold(friend.Name, friendFilter) {
			continue
		}

		bundleFilename := fmt.Sprintf("bundle-%s.zip", core.SanitizeFilename(friend.Name))
		bundlePath := filepath.Join(bundlesDir, bundleFilename)

		// Check bundle exists
		bundleInfo := ""
		if info, err := os.Stat(bundlePath); err == nil {
			bundleInfo = fmt.Sprintf(" (%s)", formatSize(info.Size()))
		}

		// Header
		fmt.Println(strings.Repeat("\u2500", 56))
		contactLine := ""
		if friend.Contact != "" {
			contactLine = fmt.Sprintf(" (%s)", friend.Contact)
		}
		fmt.Printf("  %s%s\n", friend.Name, contactLine)

		label := bundleFilename
		if rotation {
			label += " (updated)"
		}
		fmt.Printf("  Bundle: %s%s\n", label, bundleInfo)
		fmt.Println(strings.Repeat("\u2500", 56))
		fmt.Println()

		if rotation {
			printRotationMessage(friend, bundleFilename)
		} else {
			printInitialMessage(friend, bundleFilename, p.Threshold, len(p.Friends))
		}

		fmt.Println()
	}

	if friendFilter != "" {
		found := false
		for _, f := range p.Friends {
			if strings.EqualFold(f.Name, friendFilter) {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("no friend named %q in this project", friendFilter)
		}
	}

	return nil
}

func printInitialMessage(friend project.Friend, bundleFilename string, threshold, total int) {
	name := friend.Name

	fmt.Printf("Hi %s,\n\n", name)

	fmt.Printf("I'm sending you a file called %s as part of\n", bundleFilename)
	fmt.Println("something I've set up to protect my important digital files.")
	fmt.Println()

	fmt.Printf("You're one of %d people I trust with a piece of the key. No\n", total)
	fmt.Printf("single person can open my files alone. If something happens\n")
	fmt.Printf("to me, any %d of you can work together to unlock them.\n", threshold)
	fmt.Println()

	fmt.Println("All you need to do is keep the ZIP file somewhere you won't")
	fmt.Println("lose it. A folder on your computer, a USB drive, or even")
	fmt.Println("printed out. You don't need to install anything or create")
	fmt.Println("an account.")
	fmt.Println()

	fmt.Println("If recovery is ever needed, open the file called recover.html")
	fmt.Println("inside the ZIP. It works in any browser and has instructions")
	fmt.Println("for what to do next, including how to reach the others.")
	fmt.Println()

	fmt.Println("Thank you for holding onto this.")
}

func printRotationMessage(friend project.Friend, bundleFilename string) {
	name := friend.Name

	fmt.Printf("Hi %s,\n\n", name)

	fmt.Printf("I've updated my recovery bundles. Please replace your old\n")
	fmt.Printf("%s with the new one I'm sending.\n\n", bundleFilename)

	fmt.Println("The old bundle no longer works for recovery. Everything")
	fmt.Println("else stays the same.")
}
