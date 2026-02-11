package pdf

import (
	"bytes"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/go-pdf/fpdf"
	qrcode "github.com/skip2/go-qrcode"
	"golang.org/x/text/unicode/norm"

	"github.com/eljojo/rememory/internal/core"
	"github.com/eljojo/rememory/internal/project"
	"github.com/eljojo/rememory/internal/translations"
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
	RecoveryURL      string // Base URL for QR code (e.g. "https://example.com/recover.html")
	Language         string // Bundle language (e.g. "en", "es"); defaults to "en"
}

// Font sizes
const (
	titleSize   = 16.0
	headingSize = 12.0
	bodySize    = 10.0
	monoSize    = 8.0
	smallMono   = 7.0
)

// QR code size in mm on the PDF page.
const qrSizeMM = 70.0

// QRContent returns the string that will be encoded in the QR code.
// Returns "URL#share=COMPACT". If RecoveryURL is not set, defaults to the production URL.
func (d ReadmeData) QRContent() string {
	compact := d.Share.CompactEncode()
	recoveryURL := d.RecoveryURL
	if recoveryURL == "" {
		recoveryURL = core.DefaultRecoveryURL
	}
	return recoveryURL + "#share=" + url.QueryEscape(compact)
}

// GenerateReadme creates the README.pdf content.
func GenerateReadme(data ReadmeData) ([]byte, error) {
	lang := data.Language
	if lang == "" {
		lang = "en"
	}
	t := func(key string, args ...any) string {
		return translations.T("readme", lang, key, args...)
	}

	p := fpdf.New("P", "mm", "A4", "")
	p.SetMargins(20, 20, 20)
	p.SetAutoPageBreak(true, 20)

	// Register embedded UTF-8 TrueType fonts (DejaVu Sans)
	registerUTF8Fonts(p)

	p.AddPage()

	// Title
	p.SetFont(fontSans, "B", titleSize)
	p.CellFormat(0, 10, t("title"), "", 1, "C", false, 0, "")
	p.SetFont(fontSans, "", headingSize)
	p.CellFormat(0, 8, t("for", data.Holder), "", 1, "C", false, 0, "")
	p.Ln(5)

	// Warning box
	p.SetFillColor(255, 240, 240)
	p.SetFont(fontSans, "B", bodySize)
	p.CellFormat(0, 7, "  !! "+t("warning_cannot_alone"), "", 1, "L", true, 0, "")
	p.SetFont(fontSans, "", bodySize)
	if data.Anonymous {
		p.CellFormat(0, 5, "  "+t("warning_need_shares"), "", 1, "L", true, 0, "")
	} else {
		p.CellFormat(0, 5, "  "+t("warning_need_friends"), "", 1, "L", true, 0, "")
	}
	p.Ln(2)
	p.SetFont(fontSans, "B", bodySize)
	p.CellFormat(0, 7, "  !! "+t("warning_confidential"), "", 1, "L", true, 0, "")
	p.SetFont(fontSans, "", bodySize)
	p.CellFormat(0, 5, "  "+t("warning_keep_safe"), "", 1, "L", true, 0, "")
	p.Ln(5)

	// Section: What is this?
	addSection(p, t("what_is_this"))
	addBody(p, t("what_bundle_for", data.ProjectName))
	addBody(p, t("what_one_of", data.Total))
	addBody(p, t("what_threshold", data.Threshold))
	p.Ln(5)

	// Section: Other share holders - skip for anonymous mode
	if !data.Anonymous {
		addSection(p, t("other_holders"))
		for _, friend := range data.OtherFriends {
			p.SetFont(fontSans, "B", bodySize)
			p.CellFormat(0, 6, friend.Name, "", 1, "L", false, 0, "")
			p.SetFont(fontSans, "", bodySize)
			if friend.Contact != "" {
				p.CellFormat(0, 5, "    "+t("contact_label", friend.Contact), "", 1, "L", false, 0, "")
			}
			p.Ln(2)
		}
		p.Ln(5)
	}

	// Section: Your Share (QR code + PEM block)
	addSection(p, t("your_share"))
	p.Ln(2)

	// Generate QR code PNG
	qrContent := data.QRContent()
	qrPNG, err := generateQRPNG(qrContent)
	if err != nil {
		return nil, fmt.Errorf("generating QR code: %w", err)
	}

	// Register QR image and place it centered
	qrReader := bytes.NewReader(qrPNG)
	opts := fpdf.ImageOptions{ImageType: "PNG", ReadDpi: true}
	p.RegisterImageOptionsReader("qrcode", opts, qrReader)
	pageWidth, _ := p.GetPageSize()
	leftMargin, _, rightMargin, _ := p.GetMargins()
	contentWidth := pageWidth - leftMargin - rightMargin
	qrX := leftMargin + (contentWidth-qrSizeMM)/2
	p.ImageOptions("qrcode", qrX, p.GetY(), qrSizeMM, qrSizeMM, false, opts, 0, "")
	p.SetY(p.GetY() + qrSizeMM + 3)

	// Caption under QR code
	p.SetFont(fontSans, "I", bodySize)
	p.CellFormat(0, 5, t("qr_caption"), "", 1, "C", false, 0, "")
	p.Ln(2)

	// Show the compact string below the QR for manual entry
	compact := data.Share.CompactEncode()
	p.SetFont(fontMono, "", smallMono)
	p.SetFillColor(245, 245, 245)
	p.CellFormat(0, 4, compact, "", 1, "C", true, 0, "")
	p.Ln(8)

	// Word grids (recovery words in two columns)
	nativeWords, _ := data.Share.WordsForLang(core.Lang(lang))
	if len(nativeWords) > 0 {
		if lang != "en" {
			// Non-English: show native language grid first, then English
			langName := t("lang_" + lang)
			renderWordGridPDF(p, nativeWords, t("recovery_words_title_lang", len(nativeWords), langName), leftMargin, contentWidth)
			p.SetFont(fontSans, "I", bodySize)
			p.MultiCell(0, 5, t("recovery_words_hint"), "", "L", false)
			p.Ln(5)

			// English fallback grid
			englishWords, _ := data.Share.Words()
			renderWordGridPDF(p, englishWords, t("recovery_words_title_english", len(englishWords)), leftMargin, contentWidth)
			p.SetFont(fontSans, "I", bodySize)
			p.MultiCell(0, 5, t("recovery_words_dual_hint"), "", "L", false)
			p.Ln(5)
		} else {
			// English only: single grid
			renderWordGridPDF(p, nativeWords, t("recovery_words_title", len(nativeWords)), leftMargin, contentWidth)
			p.SetFont(fontSans, "I", bodySize)
			p.MultiCell(0, 5, t("recovery_words_hint"), "", "L", false)
			p.Ln(5)
		}
	}

	// PEM block (machine-readable format)
	// Ensure PEM block starts on a page with enough room for the header + content
	shareText := data.Share.Encode()
	shareLines := strings.Split(shareText, "\n")
	pemHeight := 10.0 // section header
	for _, line := range shareLines {
		if line != "" {
			pemHeight += 3.5
		} else {
			pemHeight += 1.5
		}
	}
	{
		_, pageHeight := p.GetPageSize()
		_, _, _, bottomMargin := p.GetMargins()
		usableBottom := pageHeight - bottomMargin
		if p.GetY()+pemHeight > usableBottom {
			p.AddPage()
		}
	}
	addSection(p, t("machine_readable"))
	p.SetFont(fontMono, "", smallMono)
	p.SetFillColor(245, 245, 245)

	for _, line := range shareLines {
		if line != "" {
			p.CellFormat(0, 3.5, line, "", 1, "L", true, 0, "")
		} else {
			p.Ln(1.5)
		}
	}
	p.Ln(5)

	// Section: Browser recovery
	addSection(p, t("recover_browser"))
	addBody(p, t("recover_step1"))
	p.Ln(2)
	p.SetFont(fontSans, "B", bodySize)
	p.MultiCell(0, 5, "   "+t("recover_share_loaded"), "", "L", false)
	p.Ln(2)
	addBody(p, t("recover_step2"))
	addBody(p, "   "+t("recover_step2_drag"))
	addBody(p, "   "+t("recover_step2_click"))
	p.Ln(2)
	if data.Anonymous {
		addBody(p, t("recover_anon_step3"))
		addBody(p, "   "+t("recover_anon_step3_drag"))
		addBody(p, "   "+t("recover_anon_step3_paste"))
		p.Ln(2)
		addBody(p, t("recover_anon_step4_auto", data.Threshold))
		p.Ln(2)
		addBody(p, t("recover_anon_step5"))
	} else {
		addBody(p, t("recover_step3_contact"))
		addBody(p, "   "+t("recover_step3_ask"))
		p.Ln(2)
		addBody(p, t("recover_step4"))
		addBody(p, "   "+t("recover_step4_drag"))
		addBody(p, "   "+t("recover_step4_paste"))
		p.Ln(2)
		addBody(p, t("recover_step5_checkmarks"))
		addBody(p, "   "+t("recover_step5_auto", data.Threshold))
		p.Ln(2)
		addBody(p, t("recover_step6"))
	}
	p.Ln(2)
	p.SetFont(fontSans, "I", bodySize)
	p.MultiCell(0, 5, t("recover_offline"), "", "L", false)
	p.Ln(5)

	// Section: CLI fallback
	addSection(p, t("recover_cli"))
	addBody(p, t("recover_cli_hint"))
	p.SetFont(fontMono, "", monoSize)
	p.MultiCell(0, 5, data.GitHubReleaseURL, "", "L", false)
	p.Ln(2)
	addBody(p, t("recover_cli_usage"))
	p.Ln(5)

	// Footer: Metadata
	p.SetFont(fontSans, "B", smallMono)
	p.CellFormat(0, 5, "METADATA", "", 1, "L", false, 0, "")
	p.SetFont(fontMono, "", smallMono)
	p.SetFillColor(245, 245, 245)
	addMeta(p, "rememory-version", data.Version)
	addMeta(p, "created", data.Created.Format(time.RFC3339))
	addMeta(p, "project", data.ProjectName)
	addMeta(p, "threshold", fmt.Sprintf("%d", data.Threshold))
	addMeta(p, "total", fmt.Sprintf("%d", data.Total))
	addMeta(p, "github-release", data.GitHubReleaseURL)
	addMeta(p, "checksum-manifest", data.ManifestChecksum)
	addMeta(p, "checksum-recover-html", data.RecoverChecksum)

	// Write to buffer
	var buf bytes.Buffer
	if err := p.Output(&buf); err != nil {
		return nil, fmt.Errorf("writing PDF: %w", err)
	}

	return buf.Bytes(), nil
}

// renderWordGridPDF renders a two-column word grid with page-break detection.
func renderWordGridPDF(p *fpdf.Fpdf, words []string, title string, leftMargin, contentWidth float64) {
	half := (len(words) + 1) / 2
	rowHeight := 5.5
	gridHeight := 10 + float64(half)*rowHeight + 2
	_, pageHeight := p.GetPageSize()
	_, _, _, bottomMargin := p.GetMargins()
	usableBottom := pageHeight - bottomMargin

	if p.GetY()+gridHeight > usableBottom {
		p.AddPage()
	}

	addSection(p, title)
	p.SetFont(fontMono, "", bodySize)

	colWidth := contentWidth / 2
	startY := p.GetY()

	for i := 0; i < half; i++ {
		y := startY + float64(i)*rowHeight

		// NFC-normalize words so accented characters render as single glyphs
		// (BIP39 word lists may store them in NFD form: ra + combining accent + pido)
		p.SetXY(leftMargin, y)
		p.CellFormat(colWidth, 5, fmt.Sprintf("%2d. %s", i+1, norm.NFC.String(words[i])), "", 0, "L", false, 0, "")

		if i+half < len(words) {
			p.SetXY(leftMargin+colWidth, y)
			p.CellFormat(colWidth, 5, fmt.Sprintf("%2d. %s", i+half+1, norm.NFC.String(words[i+half])), "", 0, "L", false, 0, "")
		}
	}

	p.SetY(startY + float64(half)*rowHeight + 2)
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

// generateQRPNG creates a QR code PNG image for the given content string.
func generateQRPNG(content string) ([]byte, error) {
	return qrcode.Encode(content, qrcode.Medium, 512)
}
