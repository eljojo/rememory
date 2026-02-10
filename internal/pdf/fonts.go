package pdf

import (
	_ "embed"

	"github.com/go-pdf/fpdf"
)

// DejaVu Sans font files (Bitstream Vera / DejaVu license - see fonts/LICENSE-DejaVu.txt)
// These provide comprehensive UTF-8 / Unicode coverage including Latin, Cyrillic,
// Greek, Arabic, Hebrew, and many other scripts.

//go:embed fonts/DejaVuSans.ttf
var dejaVuSansRegular []byte

//go:embed fonts/DejaVuSans-Bold.ttf
var dejaVuSansBold []byte

//go:embed fonts/DejaVuSans-Oblique.ttf
var dejaVuSansOblique []byte

//go:embed fonts/DejaVuSans-BoldOblique.ttf
var dejaVuSansBoldOblique []byte

//go:embed fonts/DejaVuSansMono.ttf
var dejaVuSansMonoRegular []byte

//go:embed fonts/DejaVuSansMono-Bold.ttf
var dejaVuSansMonoBold []byte

// Font family names used throughout the PDF generator.
const (
	fontSans = "DejaVuSans"
	fontMono = "DejaVuSansMono"
)

// registerUTF8Fonts adds the embedded DejaVu Sans UTF-8 fonts to the PDF instance.
// After calling this, use fontSans and fontMono as the family name in SetFont().
func registerUTF8Fonts(pdf *fpdf.Fpdf) {
	pdf.AddUTF8FontFromBytes(fontSans, "", dejaVuSansRegular)
	pdf.AddUTF8FontFromBytes(fontSans, "B", dejaVuSansBold)
	pdf.AddUTF8FontFromBytes(fontSans, "I", dejaVuSansOblique)
	pdf.AddUTF8FontFromBytes(fontSans, "BI", dejaVuSansBoldOblique)

	pdf.AddUTF8FontFromBytes(fontMono, "", dejaVuSansMonoRegular)
	pdf.AddUTF8FontFromBytes(fontMono, "B", dejaVuSansMonoBold)
}
