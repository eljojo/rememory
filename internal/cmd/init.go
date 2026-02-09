package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/eljojo/rememory/internal/project"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init [name]",
	Short: "Create a new rememory project",
	Long: `Create a new rememory project with a manifest directory and configuration.

The project will contain:
  - project.yml: Configuration with friends' contact information
  - manifest/: Directory for your secret files

Example:
  rememory init my-recovery-2026
  rememory init my-recovery --from ../old-project`,
	Args: cobra.MaximumNArgs(1),
	RunE: runInit,
}

var (
	initFrom      string
	initName      string
	initThreshold int
	initFriends   []string
	initAnonymous bool
	initShares    int
)

const (
	// MaxNameLength is the maximum allowed length for friend names
	MaxNameLength = 200
	// MaxContactLength is the maximum allowed length for contact info
	MaxContactLength = 500
)

func init() {
	rootCmd.AddCommand(initCmd)
	initCmd.Flags().StringVar(&initFrom, "from", "", "Base new project on existing project (copies friends)")
	initCmd.Flags().StringVar(&initName, "name", "", "Project name (defaults to directory name)")
	initCmd.Flags().IntVar(&initThreshold, "threshold", 0, "Number of shares needed to recover")
	initCmd.Flags().StringArrayVar(&initFriends, "friend", nil, "Friend in format 'Name' or 'Name,contact info' (repeatable)")
	initCmd.Flags().BoolVar(&initAnonymous, "anonymous", false, "Anonymous mode (no contact info for shareholders)")
	initCmd.Flags().IntVar(&initShares, "shares", 0, "Number of shares (for anonymous mode)")
}

func runInit(cmd *cobra.Command, args []string) error {
	// Determine project directory from args
	dirName := "recovery"
	if len(args) > 0 {
		dirName = args[0]
	}

	dir, err := filepath.Abs(dirName)
	if err != nil {
		return fmt.Errorf("resolving path: %w", err)
	}

	// Determine project name
	name := initName
	if name == "" {
		name = filepath.Base(dir)
	}

	// Check if directory already exists
	if _, err := os.Stat(dir); err == nil {
		return fmt.Errorf("directory already exists: %s", dir)
	}

	fmt.Printf("Creating new rememory project: %s/\n\n", dirName)

	var friends []project.Friend
	var threshold int
	var anonymous bool

	// Anonymous mode
	if initAnonymous {
		anonymous = true
		reader := bufio.NewReader(os.Stdin)

		numShares := initShares
		if numShares == 0 {
			fmt.Print("How many shares? [5]: ")
			numStr, _ := reader.ReadString('\n')
			numStr = strings.TrimSpace(numStr)
			numShares = 5
			if numStr != "" {
				n, err := strconv.Atoi(numStr)
				if err != nil || n < 2 {
					return fmt.Errorf("invalid number of shares (minimum 2)")
				}
				numShares = n
			}
		}

		threshold = initThreshold
		if threshold == 0 {
			defaultThreshold := (numShares + 1) / 2
			if defaultThreshold < 2 {
				defaultThreshold = 2
			}
			fmt.Printf("How many shares needed to recover? [%d]: ", defaultThreshold)
			threshStr, _ := reader.ReadString('\n')
			threshStr = strings.TrimSpace(threshStr)
			threshold = defaultThreshold
			if threshStr != "" {
				t, err := strconv.Atoi(threshStr)
				if err != nil || t < 2 || t > numShares {
					return fmt.Errorf("invalid threshold (must be 2-%d)", numShares)
				}
				threshold = t
			}
		}

		if threshold < 2 || threshold > numShares {
			return fmt.Errorf("invalid threshold: must be between 2 and %d", numShares)
		}

		// Generate synthetic friends
		friends = make([]project.Friend, numShares)
		for i := 0; i < numShares; i++ {
			friends[i] = project.Friend{Name: fmt.Sprintf("Share %d", i+1)}
		}

		fmt.Printf("\nAnonymous mode: %d shares, threshold %d of %d\n\n", numShares, threshold, numShares)
	} else if len(initFriends) > 0 {
		// Non-interactive mode: use flags
		friends, err = parseFriendFlags(initFriends)
		if err != nil {
			return err
		}

		threshold = initThreshold
		if threshold == 0 {
			threshold = (len(friends) + 1) / 2 // Default to majority
			if threshold < 2 {
				threshold = 2
			}
		}

		if threshold < 2 || threshold > len(friends) {
			return fmt.Errorf("invalid threshold: must be between 2 and %d", len(friends))
		}

		fmt.Printf("Friends: %s\n", friendNames(friends))
		fmt.Printf("Threshold: %d of %d\n\n", threshold, len(friends))
	} else if initFrom != "" {
		fromDir, err := filepath.Abs(initFrom)
		if err != nil {
			return fmt.Errorf("resolving --from path: %w", err)
		}

		existing, err := project.Load(fromDir)
		if err != nil {
			return fmt.Errorf("loading existing project: %w", err)
		}

		friends = existing.Friends
		threshold = existing.Threshold
		fmt.Printf("Copying configuration from: %s\n", initFrom)
		fmt.Printf("  Friends: %s\n", friendNames(friends))
		fmt.Printf("  Threshold: %d of %d\n\n", threshold, len(friends))
	} else {
		// Interactive prompts
		reader := bufio.NewReader(os.Stdin)

		// Number of friends
		fmt.Print("How many friends will hold shares? [5]: ")
		numStr, _ := reader.ReadString('\n')
		numStr = strings.TrimSpace(numStr)
		numFriends := 5
		if numStr != "" {
			n, err := strconv.Atoi(numStr)
			if err != nil || n < 2 {
				return fmt.Errorf("invalid number of friends (minimum 2)")
			}
			numFriends = n
		}

		// Threshold
		defaultThreshold := (numFriends + 1) / 2 // Majority by default
		if defaultThreshold < 2 {
			defaultThreshold = 2
		}
		fmt.Printf("How many shares needed to recover? [%d]: ", defaultThreshold)
		threshStr, _ := reader.ReadString('\n')
		threshStr = strings.TrimSpace(threshStr)
		threshold = defaultThreshold
		if threshStr != "" {
			t, err := strconv.Atoi(threshStr)
			if err != nil || t < 2 || t > numFriends {
				return fmt.Errorf("invalid threshold (must be 2-%d)", numFriends)
			}
			threshold = t
		}

		fmt.Println()

		// Collect friend information
		friends = make([]project.Friend, numFriends)
		for i := 0; i < numFriends; i++ {
			fmt.Printf("Friend %d:\n", i+1)

			fmt.Print("  Name: ")
			nameStr, _ := reader.ReadString('\n')
			nameStr = strings.TrimSpace(nameStr)
			if nameStr == "" {
				return fmt.Errorf("name is required")
			}
			if len(nameStr) > MaxNameLength {
				return fmt.Errorf("name too long (max %d characters)", MaxNameLength)
			}
			friends[i].Name = nameStr

			fmt.Print("  Contact info (optional): ")
			contactStr, _ := reader.ReadString('\n')
			contactStr = strings.TrimSpace(contactStr)
			if len(contactStr) > MaxContactLength {
				return fmt.Errorf("contact info too long (max %d characters)", MaxContactLength)
			}
			friends[i].Contact = contactStr

			fmt.Println()
		}
	}

	// Create the project
	p, err := project.NewWithOptions(dir, name, threshold, friends, anonymous)
	if err != nil {
		return fmt.Errorf("creating project: %w", err)
	}

	// Write the manifest README template
	templateData := project.TemplateData{
		ProjectName: name,
		Friends:     friends,
		Threshold:   threshold,
	}
	if err := project.WriteManifestReadme(p.ManifestPath(), templateData); err != nil {
		return fmt.Errorf("creating manifest README: %w", err)
	}

	fmt.Printf("Created %s/\n", name)
	fmt.Printf("  - project.yml (edit to update friends)\n")
	fmt.Printf("  - manifest/README.md (add your secrets here)\n")
	fmt.Println()
	fmt.Println("Next: Add files to manifest/, then run `rememory seal`")

	return nil
}

func friendNames(friends []project.Friend) string {
	names := make([]string, len(friends))
	for i, f := range friends {
		names[i] = f.Name
	}
	return strings.Join(names, ", ")
}

// parseFriendFlags parses --friend flags in format "Name" or "Name,contact info"
func parseFriendFlags(flags []string) ([]project.Friend, error) {
	friends := make([]project.Friend, len(flags))
	for i, f := range flags {
		// Split on the first comma only — everything after it is free-form contact
		name := f
		contact := ""
		if idx := strings.Index(f, ","); idx >= 0 {
			name = f[:idx]
			contact = strings.TrimSpace(f[idx+1:])
		}

		friends[i] = project.Friend{
			Name:    strings.TrimSpace(name),
			Contact: contact,
		}

		if friends[i].Name == "" {
			return nil, fmt.Errorf("friend name cannot be empty")
		}
		if len(friends[i].Name) > MaxNameLength {
			return nil, fmt.Errorf("friend name too long (max %d characters)", MaxNameLength)
		}
		if len(friends[i].Contact) > MaxContactLength {
			return nil, fmt.Errorf("friend contact too long (max %d characters)", MaxContactLength)
		}
	}
	return friends, nil
}
