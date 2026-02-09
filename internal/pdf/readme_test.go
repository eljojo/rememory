package pdf

import (
	"bytes"
	"image/png"
	"testing"
	"time"

	"github.com/eljojo/rememory/internal/core"
	"github.com/eljojo/rememory/internal/project"
)

func testReadmeData() ReadmeData {
	share := core.NewShare(1, 3, 2, "Alice", []byte("test-share-data-for-qr-code-12345"))
	return ReadmeData{
		ProjectName:      "Test Project",
		Holder:           "Alice",
		Share:            share,
		OtherFriends:     []project.Friend{{Name: "Bob", Contact: "bob@example.com"}},
		Threshold:        2,
		Total:            3,
		Version:          "v0.0.1-test",
		GitHubReleaseURL: "https://github.com/eljojo/rememory/releases",
		ManifestChecksum: "sha256:abcdef1234567890",
		RecoverChecksum:  "sha256:0987654321fedcba",
		Created:          time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}
}

func TestGenerateReadme(t *testing.T) {
	data := testReadmeData()
	pdfBytes, err := GenerateReadme(data)
	if err != nil {
		t.Fatalf("GenerateReadme: %v", err)
	}
	if len(pdfBytes) == 0 {
		t.Fatal("generated PDF is empty")
	}
	// Verify it's a valid PDF (starts with %PDF-)
	if !bytes.HasPrefix(pdfBytes, []byte("%PDF-")) {
		t.Error("output does not start with PDF header")
	}
}

func TestGenerateReadmeAnonymous(t *testing.T) {
	data := testReadmeData()
	data.Anonymous = true
	data.OtherFriends = nil
	pdfBytes, err := GenerateReadme(data)
	if err != nil {
		t.Fatalf("GenerateReadme (anonymous): %v", err)
	}
	if len(pdfBytes) == 0 {
		t.Fatal("generated PDF is empty")
	}
}

func TestQRContent(t *testing.T) {
	data := testReadmeData()

	// Without RecoveryURL: just the compact share string
	content := data.QRContent()
	expected := data.Share.CompactEncode()
	if content != expected {
		t.Errorf("QRContent without URL: got %q, want %q", content, expected)
	}

	// Verify the compact string round-trips
	_, err := core.ParseCompact(content)
	if err != nil {
		t.Fatalf("compact string from QRContent doesn't parse: %v", err)
	}
}

func TestQRContentWithRecoveryURL(t *testing.T) {
	data := testReadmeData()
	data.RecoveryURL = "https://example.com/recover.html"

	content := data.QRContent()
	expected := "https://example.com/recover.html#share=" + data.Share.CompactEncode()
	if content != expected {
		t.Errorf("QRContent with URL: got %q, want %q", content, expected)
	}
}

func TestQRCodeGeneratesValidPNG(t *testing.T) {
	data := testReadmeData()

	// Generate the PDF (which includes the QR code)
	pdfBytes, err := GenerateReadme(data)
	if err != nil {
		t.Fatalf("GenerateReadme: %v", err)
	}
	if len(pdfBytes) == 0 {
		t.Fatal("generated PDF is empty")
	}

	// Also verify the QR code PNG directly
	qrContent := data.QRContent()
	qrPNG, err := generateQRPNG(qrContent)
	if err != nil {
		t.Fatalf("generateQRPNG: %v", err)
	}

	// Verify it's a valid PNG
	img, err := png.Decode(bytes.NewReader(qrPNG))
	if err != nil {
		t.Fatalf("QR code is not valid PNG: %v", err)
	}

	bounds := img.Bounds()
	if bounds.Dx() == 0 || bounds.Dy() == 0 {
		t.Error("QR code image has zero dimensions")
	}
}

func TestQRCodeContentMatchesCompact(t *testing.T) {
	// Verify the data encoded in the QR code is exactly the compact share string
	share := core.NewShare(2, 5, 3, "Bob", []byte("another-share-data-for-testing"))
	data := ReadmeData{
		Share:     share,
		Holder:    "Bob",
		Threshold: 3,
		Total:     5,
	}

	qrContent := data.QRContent()
	compact := share.CompactEncode()

	if qrContent != compact {
		t.Errorf("QR content doesn't match compact encoding:\n  got:  %q\n  want: %q", qrContent, compact)
	}

	// Verify the compact string correctly round-trips
	parsed, err := core.ParseCompact(qrContent)
	if err != nil {
		t.Fatalf("ParseCompact: %v", err)
	}
	if parsed.Index != share.Index || parsed.Total != share.Total || parsed.Threshold != share.Threshold {
		t.Errorf("parsed share metadata mismatch: got %d/%d/%d, want %d/%d/%d",
			parsed.Index, parsed.Total, parsed.Threshold,
			share.Index, share.Total, share.Threshold)
	}
	if !bytes.Equal(parsed.Data, share.Data) {
		t.Error("parsed share data mismatch")
	}
}
