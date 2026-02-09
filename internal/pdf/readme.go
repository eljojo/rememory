package pdf

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"github.com/go-pdf/fpdf"

	"github.com/eljojo/rememory/internal/core"
	"github.com/eljojo/rememory/internal/project"
)

// ReadmeData contains all data needed to generate README.pdf
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

// Font sizes
const (
	titleSize   = 16.0
	headingSize = 12.0
	bodySize    = 10.0
	monoSize    = 8.0
)

// GenerateReadme creates the README.pdf content.
func GenerateReadme(data ReadmeData) ([]byte, error) {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(20, 20, 20)
	pdf.SetAutoPageBreak(true, 20)

	// Register embedded UTF-8 TrueType fonts (DejaVu Sans)
	registerUTF8Fonts(pdf)

	pdf.AddPage()

	// Title
	pdf.SetFont(fontSans, "B", titleSize)
	pdf.CellFormat(0, 10, "REMEMORY RECOVERY BUNDLE", "", 1, "C", false, 0, "")
	pdf.SetFont(fontSans, "", headingSize)
	pdf.CellFormat(0, 8, fmt.Sprintf("For: %s", data.Holder), "", 1, "C", false, 0, "")
	pdf.Ln(5)

	// Warning box
	pdf.SetFillColor(255, 240, 240)
	pdf.SetFont(fontSans, "B", bodySize)
	pdf.CellFormat(0, 7, "  !! YOU CANNOT USE THIS FILE ALONE", "", 1, "L", true, 0, "")
	pdf.SetFont(fontSans, "", bodySize)
	if data.Anonymous {
		pdf.CellFormat(0, 5, "  You will need to combine this with other shares.", "", 1, "L", true, 0, "")
	} else {
		pdf.CellFormat(0, 5, "  You will need help from other friends listed below.", "", 1, "L", true, 0, "")
	}
	pdf.Ln(2)
	pdf.SetFont(fontSans, "B", bodySize)
	pdf.CellFormat(0, 7, "  !! CONFIDENTIAL - DO NOT SHARE THIS FILE", "", 1, "L", true, 0, "")
	pdf.SetFont(fontSans, "", bodySize)
	pdf.CellFormat(0, 5, "  This document contains your secret share. Keep it safe.", "", 1, "L", true, 0, "")
	pdf.Ln(3)

	// Spanish AI help note (green background, right after warning)
	pdf.SetFillColor(220, 245, 220)
	pdf.SetFont(fontSans, "B", bodySize)
	pdf.CellFormat(0, 6, "  NOTA PARA HISPANOHABLANTES", "", 1, "L", true, 0, "")
	pdf.SetFont(fontSans, "I", bodySize)
	pdf.MultiCell(0, 5, "  Si no entiendes inglés, puedes usar ChatGPT u otra inteligencia artificial para que te ayude a entender estas instrucciones y recuperar los datos. Copia este documento completo y pídele a la IA que te explique los pasos. La herramienta recover.html también está disponible en español.", "", "L", true)
	pdf.Ln(5)

	// Section: What is this?
	addSection(pdf, "WHAT IS THIS?")
	addBody(pdf, fmt.Sprintf("This bundle allows you to help recover encrypted secrets for: %s", data.ProjectName))
	addBody(pdf, fmt.Sprintf("You are one of %d trusted friends who hold pieces of the recovery key.", data.Total))
	addBody(pdf, fmt.Sprintf("At least %d of you must cooperate to decrypt the contents.", data.Threshold))
	pdf.Ln(5)

	// Section: Other share holders (right after What is this?) - skip for anonymous mode
	if !data.Anonymous {
		addSection(pdf, "OTHER SHARE HOLDERS (contact to coordinate recovery)")
		for _, friend := range data.OtherFriends {
			pdf.SetFont(fontSans, "B", bodySize)
			pdf.CellFormat(0, 6, friend.Name, "", 1, "L", false, 0, "")
			pdf.SetFont(fontSans, "", bodySize)
			if friend.Contact != "" {
				pdf.CellFormat(0, 5, fmt.Sprintf("    Contact: %s", friend.Contact), "", 1, "L", false, 0, "")
			}
			pdf.Ln(2)
		}
		pdf.Ln(5)
	}

	// Section: Browser recovery
	addSection(pdf, "HOW TO RECOVER (PRIMARY METHOD - Browser)")
	addBody(pdf, "1. Open recover.html in any modern browser (Chrome, Firefox, Safari, Edge)")
	pdf.Ln(2)
	pdf.SetFont(fontSans, "B", bodySize)
	pdf.MultiCell(0, 5, "   YOUR SHARE IS ALREADY LOADED!", "", "L", false)
	pdf.SetFont(fontSans, "I", bodySize)
	pdf.MultiCell(0, 5, "   The recovery tool is personalized for you.", "", "L", false)
	pdf.Ln(2)
	addBody(pdf, "2. Load the encrypted file (MANIFEST.age) from this bundle:")
	addBody(pdf, "   - Drag and drop it onto the manifest area, OR click to browse")
	pdf.Ln(2)
	if data.Anonymous {
		addBody(pdf, "3. Add other shares as you receive them")
		addBody(pdf, "   - Drag and drop README.txt files onto the page, OR")
		addBody(pdf, "   - Click the clipboard button to paste share text")
		pdf.Ln(2)
		addBody(pdf, fmt.Sprintf("4. Once you have %d shares total, recovery happens AUTOMATICALLY!", data.Threshold))
		pdf.Ln(2)
		addBody(pdf, "5. Download the recovered files")
	} else {
		addBody(pdf, "3. You'll see a contact list showing other friends who hold shares.")
		addBody(pdf, "   Contact them and ask them to send you their README.txt file.")
		pdf.Ln(2)
		addBody(pdf, "4. For each friend's README.txt you receive:")
		addBody(pdf, "   - Drag and drop it onto the page, OR")
		addBody(pdf, "   - Click the clipboard button to paste their share text")
		pdf.Ln(2)
		addBody(pdf, "5. As you add shares, checkmarks appear next to each friend's name.")
		addBody(pdf, fmt.Sprintf("   Once you have %d shares total, recovery happens AUTOMATICALLY!", data.Threshold))
		pdf.Ln(2)
		addBody(pdf, "6. Download the recovered files")
	}
	pdf.Ln(2)
	pdf.SetFont(fontSans, "I", bodySize)
	pdf.MultiCell(0, 5, "Works completely offline - no internet required!", "", "L", false)
	pdf.Ln(5)

	// Section: CLI fallback
	addSection(pdf, "HOW TO RECOVER (FALLBACK - Command Line)")
	addBody(pdf, "If recover.html doesn't work, download the CLI tool from:")
	pdf.SetFont(fontMono, "", monoSize)
	pdf.MultiCell(0, 5, data.GitHubReleaseURL, "", "L", false)
	pdf.Ln(2)
	addBody(pdf, "Usage: rememory recover share1.txt share2.txt ... --manifest MANIFEST.age")
	pdf.Ln(5)

	// Section: Share
	addSection(pdf, "YOUR SHARE (upload this file or copy-paste this block)")
	pdf.SetFont(fontMono, "", monoSize)
	pdf.SetFillColor(245, 245, 245)

	// Draw share in a box
	shareText := data.Share.Encode()
	shareLines := strings.Split(shareText, "\n")
	for _, line := range shareLines {
		if line != "" {
			pdf.CellFormat(0, 4, line, "", 1, "L", true, 0, "")
		} else {
			pdf.Ln(2)
		}
	}
	pdf.Ln(5)

	// Footer: Metadata
	pdf.SetFont(fontSans, "B", monoSize)
	pdf.CellFormat(0, 5, "METADATA", "", 1, "L", false, 0, "")
	pdf.SetFont(fontMono, "", monoSize)
	pdf.SetFillColor(245, 245, 245)
	addMeta(pdf, "rememory-version", data.Version)
	addMeta(pdf, "created", data.Created.Format(time.RFC3339))
	addMeta(pdf, "project", data.ProjectName)
	addMeta(pdf, "threshold", fmt.Sprintf("%d", data.Threshold))
	addMeta(pdf, "total", fmt.Sprintf("%d", data.Total))
	addMeta(pdf, "github-release", data.GitHubReleaseURL)
	addMeta(pdf, "checksum-manifest", data.ManifestChecksum)
	addMeta(pdf, "checksum-recover-html", data.RecoverChecksum)

	// Write to buffer
	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, fmt.Errorf("writing PDF: %w", err)
	}

	return buf.Bytes(), nil
}

func addSection(pdf *fpdf.Fpdf, title string) {
	pdf.SetFont(fontSans, "B", headingSize)
	pdf.SetFillColor(230, 230, 230)
	pdf.CellFormat(0, 8, " "+title, "", 1, "L", true, 0, "")
	pdf.Ln(2)
}

func addBody(pdf *fpdf.Fpdf, text string) {
	pdf.SetFont(fontSans, "", bodySize)
	pdf.MultiCell(0, 5, text, "", "L", false)
}

func addMeta(pdf *fpdf.Fpdf, key, value string) {
	pdf.CellFormat(0, 4, fmt.Sprintf("%s: %s", key, value), "", 1, "L", true, 0, "")
}
