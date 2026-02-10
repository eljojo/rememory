package bundle

import (
	"fmt"
	"strings"
	"time"

	"github.com/eljojo/rememory/internal/core"
	"github.com/eljojo/rememory/internal/project"
)

// ReadmeData contains all data needed to generate README.txt
type ReadmeData struct {
	ProjectName      string
	Holder           string
	Share            *core.Share
	OtherFriends     []project.Friend
	Threshold        int
	Total            int
	Version          string
	GitHubReleaseURL string
	ManifestChecksum string
	RecoverChecksum  string
	Created          time.Time
	Anonymous        bool
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
	if data.Anonymous {
		sb.WriteString("    You will need to combine this with other shares.\n\n")
	} else {
		sb.WriteString("    You will need help from other friends listed below.\n\n")
	}
	sb.WriteString("!!  CONFIDENTIAL - DO NOT SHARE THIS FILE\n")
	sb.WriteString("    This document contains your secret share. Keep it safe.\n\n")

	// AI help note (in Spanish for non-English speakers) - right after warning
	sb.WriteString("    NOTA PARA HISPANOHABLANTES:\n")
	sb.WriteString("    Si no entiendes inglés, puedes usar ChatGPT u otra inteligencia artificial\n")
	sb.WriteString("    para que te ayude a entender estas instrucciones y recuperar los datos.\n")
	sb.WriteString("    Copia este documento completo y pídele a la IA que te explique los pasos.\n")
	sb.WriteString("    La herramienta recover.html también está disponible en español.\n\n")

	// What is this
	sb.WriteString("--------------------------------------------------------------------------------\n")
	sb.WriteString("WHAT IS THIS?\n")
	sb.WriteString("--------------------------------------------------------------------------------\n")
	sb.WriteString(fmt.Sprintf("This bundle allows you to help recover encrypted secrets for: %s\n", data.ProjectName))
	sb.WriteString(fmt.Sprintf("You are one of %d trusted friends who hold pieces of the recovery key.\n", data.Total))
	sb.WriteString(fmt.Sprintf("At least %d of you must cooperate to decrypt the contents.\n\n", data.Threshold))

	// Other share holders - right after What is this (skip for anonymous mode)
	if !data.Anonymous {
		sb.WriteString("--------------------------------------------------------------------------------\n")
		sb.WriteString("OTHER SHARE HOLDERS (contact to coordinate recovery)\n")
		sb.WriteString("--------------------------------------------------------------------------------\n")
		for _, friend := range data.OtherFriends {
			sb.WriteString(fmt.Sprintf("%s\n", friend.Name))
			if friend.Contact != "" {
				sb.WriteString(fmt.Sprintf("  Contact: %s\n", friend.Contact))
			}
			sb.WriteString("\n")
		}
	}

	// Primary method - Browser
	sb.WriteString("--------------------------------------------------------------------------------\n")
	sb.WriteString("HOW TO RECOVER (PRIMARY METHOD - Browser)\n")
	sb.WriteString("--------------------------------------------------------------------------------\n")
	sb.WriteString("1. Open recover.html in any modern browser (Chrome, Firefox, Safari, Edge)\n\n")
	sb.WriteString("   YOUR SHARE IS ALREADY LOADED! The recovery tool is personalized for you.\n\n")
	sb.WriteString("2. Load the encrypted file (MANIFEST.age) from this bundle:\n")
	sb.WriteString("   - Drag and drop it onto the manifest area, OR\n")
	sb.WriteString("   - Click to browse and select it\n\n")
	if data.Anonymous {
		sb.WriteString("3. Add other shares as you receive them\n")
		sb.WriteString("   - Drag and drop README.txt files onto the page, OR\n")
		sb.WriteString("   - Click the clipboard button to paste share text\n\n")
		sb.WriteString(fmt.Sprintf("4. Once you have %d shares total, recovery happens AUTOMATICALLY\n\n", data.Threshold))
		sb.WriteString("5. Download the recovered files\n\n")
	} else {
		sb.WriteString("3. You'll see a contact list showing other friends who hold shares\n")
		sb.WriteString("   Contact them and ask them to send you their README.txt file\n\n")
		sb.WriteString("4. For each friend's README.txt you receive:\n")
		sb.WriteString("   - Drag and drop it onto the page, OR\n")
		sb.WriteString("   - Click the clipboard button to paste their share text\n\n")
		sb.WriteString("5. As you add shares, checkmarks appear next to each friend's name\n")
		sb.WriteString(fmt.Sprintf("   Once you have %d shares total, recovery happens AUTOMATICALLY\n\n", data.Threshold))
		sb.WriteString("6. Download the recovered files\n\n")
	}
	sb.WriteString("Works completely offline - no internet required!\n\n")

	// Fallback method - CLI
	sb.WriteString("--------------------------------------------------------------------------------\n")
	sb.WriteString("HOW TO RECOVER (FALLBACK - Command Line)\n")
	sb.WriteString("--------------------------------------------------------------------------------\n")
	sb.WriteString("If recover.html doesn't work, download the CLI tool from:\n")
	sb.WriteString(fmt.Sprintf("%s\n\n", data.GitHubReleaseURL))
	sb.WriteString("Usage: rememory recover share1.txt share2.txt ... --manifest MANIFEST.age\n\n")

	// Share block
	sb.WriteString("--------------------------------------------------------------------------------\n")
	sb.WriteString("YOUR SHARE (upload this file or copy-paste this block)\n")
	sb.WriteString("--------------------------------------------------------------------------------\n")
	sb.WriteString(data.Share.Encode())
	sb.WriteString("\n")

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
