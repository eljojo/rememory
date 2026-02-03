package bundle

import (
	"fmt"
	"strings"
	"time"

	"github.com/eljojo/rememory/internal/project"
	"github.com/eljojo/rememory/internal/shamir"
)

// ReadmeData contains all data needed to generate README.txt
type ReadmeData struct {
	ProjectName      string
	Holder           string
	Share            *shamir.Share
	OtherFriends     []project.Friend
	Threshold        int
	Total            int
	Version          string
	GitHubReleaseURL string
	ManifestChecksum string
	RecoverChecksum  string
	Created          time.Time
}

// GenerateReadme creates the README.txt content with all embedded information.
func GenerateReadme(data ReadmeData) string {
	var sb strings.Builder

	// Header
	sb.WriteString("================================================================================\n")
	sb.WriteString("                          REMEMORY RECOVERY BUNDLE\n")
	sb.WriteString(fmt.Sprintf("                              For: %s\n", data.Holder))
	sb.WriteString("================================================================================\n\n")

	// Warnings
	sb.WriteString("!!  YOU CANNOT USE THIS FILE ALONE\n")
	sb.WriteString("    You will need help from other friends listed below.\n\n")
	sb.WriteString("!!  CONFIDENTIAL - DO NOT SHARE THIS FILE\n")
	sb.WriteString("    This document contains your secret share. Keep it safe.\n\n")

	// What is this
	sb.WriteString("--------------------------------------------------------------------------------\n")
	sb.WriteString("WHAT IS THIS?\n")
	sb.WriteString("--------------------------------------------------------------------------------\n")
	sb.WriteString(fmt.Sprintf("This bundle allows you to help recover encrypted secrets for: %s\n", data.ProjectName))
	sb.WriteString(fmt.Sprintf("You are one of %d trusted friends who hold pieces of the recovery key.\n", data.Total))
	sb.WriteString(fmt.Sprintf("At least %d of you must cooperate to decrypt the contents.\n\n", data.Threshold))

	// Primary method - Browser
	sb.WriteString("--------------------------------------------------------------------------------\n")
	sb.WriteString("HOW TO RECOVER (PRIMARY METHOD - Browser)\n")
	sb.WriteString("--------------------------------------------------------------------------------\n")
	sb.WriteString("1. Open recover.html in any modern browser (Chrome, Firefox, Safari, Edge)\n")
	sb.WriteString("2. Drag and drop this README.txt file (or paste your share from below)\n")
	sb.WriteString("3. Collect shares from other friends (they drag their README.txt too)\n")
	sb.WriteString("4. Once you have enough shares, the tool will decrypt automatically\n")
	sb.WriteString("5. Download the recovered files\n\n")
	sb.WriteString("Works completely offline - no internet required!\n\n")

	// Fallback method - CLI
	sb.WriteString("--------------------------------------------------------------------------------\n")
	sb.WriteString("HOW TO RECOVER (FALLBACK - Command Line)\n")
	sb.WriteString("--------------------------------------------------------------------------------\n")
	sb.WriteString("If recover.html doesn't work, download the CLI tool from:\n")
	sb.WriteString(fmt.Sprintf("%s\n\n", data.GitHubReleaseURL))
	sb.WriteString("Usage: rememory recover --shares share1.txt,share2.txt,... --manifest MANIFEST.age\n\n")

	// Share block
	sb.WriteString("--------------------------------------------------------------------------------\n")
	sb.WriteString("YOUR SHARE (upload this file or copy-paste this block)\n")
	sb.WriteString("--------------------------------------------------------------------------------\n")
	sb.WriteString(data.Share.Encode())
	sb.WriteString("\n")

	// Other share holders
	sb.WriteString("--------------------------------------------------------------------------------\n")
	sb.WriteString("OTHER SHARE HOLDERS (contact to coordinate recovery)\n")
	sb.WriteString("--------------------------------------------------------------------------------\n")
	for _, friend := range data.OtherFriends {
		sb.WriteString(fmt.Sprintf("%s\n", friend.Name))
		sb.WriteString(fmt.Sprintf("  Email: %s\n", friend.Email))
		if friend.Phone != "" {
			sb.WriteString(fmt.Sprintf("  Phone: %s\n", friend.Phone))
		}
		sb.WriteString("\n")
	}

	// Metadata footer
	sb.WriteString("================================================================================\n")
	sb.WriteString("METADATA FOOTER (machine-parseable)\n")
	sb.WriteString("================================================================================\n")
	sb.WriteString(fmt.Sprintf("rememory-version: %s\n", data.Version))
	sb.WriteString(fmt.Sprintf("created: %s\n", data.Created.Format(time.RFC3339)))
	sb.WriteString(fmt.Sprintf("project: %s\n", data.ProjectName))
	sb.WriteString(fmt.Sprintf("threshold: %d\n", data.Threshold))
	sb.WriteString(fmt.Sprintf("total: %d\n", data.Total))
	sb.WriteString(fmt.Sprintf("github-release: %s\n", data.GitHubReleaseURL))
	sb.WriteString(fmt.Sprintf("checksum-manifest: %s\n", data.ManifestChecksum))
	sb.WriteString(fmt.Sprintf("checksum-recover-html: %s\n", data.RecoverChecksum))
	sb.WriteString("================================================================================\n")

	return sb.String()
}
