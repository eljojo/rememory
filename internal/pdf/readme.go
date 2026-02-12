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
	ManifestEmbedded bool   // true when manifest is embedded in recover.html
}

// Font sizes
const (
	titleSize   = 22.0
	headingSize = 12.0
	bodySize    = 10.0
	monoSize    = 8.0
	smallMono   = 7.0
)

// bundleColors are soft, distinguishable colors used to give each friend's
// printed PDF a unique visual identity. Indexed by (share.Index - 1) % len.
var bundleColors = [][3]int{
	{122, 143, 166}, // dusty blue
	{85, 115, 90},   // sage
	{166, 130, 100}, // warm tan
	{140, 110, 140}, // muted plum
	{110, 145, 140}, // teal
	{180, 140, 100}, // amber
	{120, 130, 160}, // slate
	{155, 120, 120}, // dusty rose
}

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

	// Bundle identity color — each friend gets a distinct strip
	colorIdx := 0
	if data.Share != nil && data.Share.Index > 0 {
		colorIdx = (data.Share.Index - 1) % len(bundleColors)
	}
	bc := bundleColors[colorIdx]

	// Page numbers — small, centered, low-key, with identity mark
	p.SetFooterFunc(func() {
		pw, _ := p.GetPageSize()
		p.SetY(-15)
		// Small identity mark before the page number
		markW := 15.0
		markH := 2.0
		markX := (pw - markW) / 2
		p.SetFillColor(bc[0], bc[1], bc[2])
		p.Rect(markX, p.GetY()-1, markW, markH, "F")
		p.SetFont(fontSans, "", 7)
		p.SetTextColor(180, 180, 180)
		p.CellFormat(0, 10, fmt.Sprintf("%d", p.PageNo()), "", 0, "C", false, 0, "")
		p.SetTextColor(46, 42, 38)
	})

	p.AddPage()

	// Page dimensions (used throughout for centered elements)
	pageWidth, _ := p.GetPageSize()

	// Identity strip at the top of the first page
	p.SetFillColor(bc[0], bc[1], bc[2])
	p.Rect(0, 0, pageWidth, 4, "F")

	leftMargin, _, rightMargin, _ := p.GetMargins()
	contentWidth := pageWidth - leftMargin - rightMargin

	// ── Title area — certificate feel with breathing room ──
	p.Ln(12)
	p.SetFont(fontSans, "B", titleSize)
	p.CellFormat(0, 12, t("title"), "", 1, "C", false, 0, "")
	p.Ln(3)
	// Decorative horizontal rule
	p.SetDrawColor(180, 180, 180)
	p.SetLineWidth(0.4)
	ruleInset := 35.0
	p.Line(leftMargin+ruleInset, p.GetY(), pageWidth-rightMargin-ruleInset, p.GetY())
	p.Ln(4)
	p.SetFont(fontSans, "", 14)
	p.CellFormat(0, 8, t("for", data.Holder), "", 1, "C", false, 0, "")
	p.Ln(12)

	// ── What is this? — context first ──
	p.SetFont(fontSans, "B", bodySize)
	p.CellFormat(0, 6, t("what_is_this"), "", 1, "L", false, 0, "")
	p.Ln(1)
	addBody(p, t("what_bundle_for", data.ProjectName))
	addBody(p, t("what_one_of", data.Total))
	p.Ln(5)

	// ── Warning stamp — soft, centered, calm ──
	p.SetFillColor(232, 239, 234)
	p.SetTextColor(46, 42, 38)
	p.SetFont(fontSans, "B", headingSize)
	p.CellFormat(0, 11, t("warning_title"), "", 1, "C", true, 0, "")
	p.SetFillColor(232, 242, 234)
	p.SetFont(fontSans, "", 9)
	if data.Anonymous {
		p.MultiCell(0, 5, t("warning_message_shares"), "", "C", true)
	} else {
		p.MultiCell(0, 5, t("warning_message_friends"), "", "C", true)
	}
	p.Ln(8)

	// ── Recovery rule — prominent standalone box ──
	p.SetFillColor(242, 242, 248)
	p.SetDrawColor(140, 140, 160)
	p.SetLineWidth(0.5)
	ruleBoxY := p.GetY()
	ruleBoxH := 20.0
	p.Rect(leftMargin, ruleBoxY, contentWidth, ruleBoxH, "FD")
	p.SetFont(fontSans, "", 9)
	p.SetXY(leftMargin, ruleBoxY+2)
	p.CellFormat(contentWidth, 5, t("recovery_rule"), "", 1, "C", false, 0, "")
	p.SetFont(fontSans, "B", 18)
	p.SetXY(leftMargin, ruleBoxY+8)
	p.CellFormat(contentWidth, 10, t("recovery_rule_count", data.Threshold, data.Total), "", 1, "C", false, 0, "")
	p.SetY(ruleBoxY + ruleBoxH + 8)
	p.SetDrawColor(0, 0, 0)
	p.SetLineWidth(0.2)

	// ── Other share holders — contact card layout ──
	if !data.Anonymous {
		addSection(p, t("other_holders"))
		for i, friend := range data.OtherFriends {
			p.SetFont(fontSans, "B", bodySize)
			if friend.Contact != "" {
				nameStr := "   " + friend.Name + "  "
				nameW := p.GetStringWidth(nameStr)
				p.CellFormat(nameW, 7, nameStr, "", 0, "L", false, 0, "")
				p.SetFont(fontSans, "", bodySize)
				p.CellFormat(0, 7, "\u2014  "+friend.Contact, "", 1, "L", false, 0, "")
			} else {
				p.CellFormat(0, 7, "   "+friend.Name, "", 1, "L", false, 0, "")
			}
			if i < len(data.OtherFriends)-1 {
				p.Ln(2)
			}
		}
		p.Ln(8)
	}

	// ── Sharing your share — procedure card with grey background ──
	p.SetFillColor(245, 245, 245)
	p.SetFont(fontSans, "B", headingSize)
	p.CellFormat(0, 10, " "+t("sharing_title"), "", 1, "L", true, 0, "")
	p.CellFormat(0, 2, "", "", 1, "", true, 0, "")
	p.SetFont(fontSans, "", bodySize)
	p.MultiCell(0, 5, " "+t("sharing_verify"), "", "L", true)
	p.CellFormat(0, 3, "", "", 1, "", true, 0, "")
	p.MultiCell(0, 5, "   \u2022 "+t("sharing_easiest"), "", "L", true)
	p.MultiCell(0, 5, "   \u2022 "+t("sharing_readme_only"), "", "L", true)
	p.MultiCell(0, 5, "   \u2022 "+t("sharing_words_phone"), "", "L", true)
	p.MultiCell(0, 5, "   \u2022 "+t("sharing_qr_mail"), "", "L", true)
	p.CellFormat(0, 3, "", "", 1, "", true, 0, "")
	p.Ln(5)

	// Section: Your Share (QR code + PEM block)
	// Ensure the section header + QR code + caption + compact string stay together
	qrBlockHeight := 10.0 + 2.0 + qrSizeMM + 3.0 + 5.0 + 2.0 + 4.0 // header + gap + QR + gap + caption + gap + compact
	{
		_, pageHeight := p.GetPageSize()
		_, _, _, bottomMargin := p.GetMargins()
		usableBottom := pageHeight - bottomMargin
		if p.GetY()+qrBlockHeight > usableBottom {
			p.AddPage()
		}
	}
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
	p.SetFont(fontSans, "", bodySize)
	p.MultiCell(0, 5, "   "+t("recover_no_html"), "", "L", false)
	p.Ln(2)
	if data.ManifestEmbedded {
		addBody(p, t("recover_step2_embedded"))
		addBody(p, "   "+t("recover_step2_embedded_hint"))
	} else {
		addBody(p, t("recover_step2"))
		addBody(p, "   "+t("recover_step2_drag"))
		addBody(p, "   "+t("recover_step2_click"))
	}
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
