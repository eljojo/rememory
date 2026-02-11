package bundle

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/eljojo/rememory/internal/core"
	"github.com/eljojo/rememory/internal/html"
	"github.com/eljojo/rememory/internal/pdf"
	"github.com/eljojo/rememory/internal/project"
	"github.com/eljojo/rememory/internal/translations"
)

// Config holds configuration for bundle generation.
type Config struct {
	Version          string // Tool version (e.g., "v1.0.0")
	GitHubReleaseURL string // URL to GitHub release for CLI download
	WASMBytes        []byte // Compiled recover.wasm binary
	RecoveryURL      string // Optional: base URL for QR code (e.g. "https://example.com/recover.html")
}

// GenerateAll creates bundles for all friends in the project.
func GenerateAll(p *project.Project, cfg Config) error {
	if p.Sealed == nil {
		return fmt.Errorf("project must be sealed before generating bundles")
	}

	bundlesDir := filepath.Join(p.OutputPath(), "bundles")
	if err := os.MkdirAll(bundlesDir, 0755); err != nil {
		return fmt.Errorf("creating bundles directory: %w", err)
	}

	// Load all shares
	shares, err := loadShares(p)
	if err != nil {
		return fmt.Errorf("loading shares: %w", err)
	}

	// Read MANIFEST.age
	manifestPath := p.ManifestAgePath()
	manifestData, err := os.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("reading manifest: %w", err)
	}
	manifestChecksum := core.HashBytes(manifestData)

	// Generate bundle for each friend
	for i, friend := range p.Friends {
		share := shares[i]

		// Resolve language: friend override > project default > "en"
		lang := friend.Language
		if lang == "" {
			lang = p.Language
		}
		if lang == "" {
			lang = "en"
		}

		// Get other friends (excluding this one) - empty for anonymous mode
		var otherFriends []project.Friend
		var otherFriendsInfo []html.FriendInfo
		if !p.Anonymous {
			otherFriends = make([]project.Friend, 0, len(p.Friends)-1)
			otherFriendsInfo = make([]html.FriendInfo, 0, len(p.Friends)-1)
			for j, f := range p.Friends {
				if j != i {
					otherFriends = append(otherFriends, f)
					otherFriendsInfo = append(otherFriendsInfo, html.FriendInfo{
						Name:       f.Name,
						Contact:    f.Contact,
						ShareIndex: j + 1, // 1-based share index
					})
				}
			}
		}

		// Generate personalized recover.html for this friend
		personalization := &html.PersonalizationData{
			Holder:       friend.Name,
			HolderShare:  share.Encode(),
			OtherFriends: otherFriendsInfo,
			Threshold:    p.Threshold,
			Total:        len(p.Friends),
			Language:     lang,
		}
		recoverHTML := html.GenerateRecoverHTML(cfg.WASMBytes, cfg.Version, cfg.GitHubReleaseURL, personalization)
		recoverChecksum := core.HashString(recoverHTML)

		bundlePath := filepath.Join(bundlesDir, fmt.Sprintf("bundle-%s.zip", core.SanitizeFilename(friend.Name)))

		err := GenerateBundle(BundleParams{
			OutputPath:       bundlePath,
			ProjectName:      p.Name,
			Friend:           friend,
			Share:            share,
			OtherFriends:     otherFriends,
			Threshold:        p.Threshold,
			Total:            len(p.Friends),
			ManifestData:     manifestData,
			ManifestChecksum: manifestChecksum,
			RecoverHTML:      recoverHTML,
			RecoverChecksum:  recoverChecksum,
			Version:          cfg.Version,
			GitHubReleaseURL: cfg.GitHubReleaseURL,
			SealedAt:         p.Sealed.At,
			Anonymous:        p.Anonymous,
			RecoveryURL:      cfg.RecoveryURL,
			Language:         lang,
		})
		if err != nil {
			return fmt.Errorf("generating bundle for %s: %w", friend.Name, err)
		}

		// Verify the bundle we just created
		if err := VerifyBundle(bundlePath); err != nil {
			return fmt.Errorf("verifying bundle for %s: %w", friend.Name, err)
		}
	}

	return nil
}

// BundleParams contains all parameters for generating a single bundle.
type BundleParams struct {
	OutputPath       string
	ProjectName      string
	Friend           project.Friend
	Share            *core.Share
	OtherFriends     []project.Friend
	Threshold        int
	Total            int
	ManifestData     []byte
	ManifestChecksum string
	RecoverHTML      string
	RecoverChecksum  string
	Version          string
	GitHubReleaseURL string
	SealedAt         time.Time
	Anonymous        bool
	RecoveryURL      string
	Language         string // Bundle language for this friend
}

// GenerateBundle creates a single bundle ZIP file for one friend.
func GenerateBundle(params BundleParams) error {
	// Common data for both README formats
	readmeData := ReadmeData{
		ProjectName:      params.ProjectName,
		Holder:           params.Friend.Name,
		Share:            params.Share,
		OtherFriends:     params.OtherFriends,
		Threshold:        params.Threshold,
		Total:            params.Total,
		Version:          params.Version,
		GitHubReleaseURL: params.GitHubReleaseURL,
		ManifestChecksum: params.ManifestChecksum,
		RecoverChecksum:  params.RecoverChecksum,
		Created:          params.SealedAt,
		Anonymous:        params.Anonymous,
		Language:         params.Language,
	}

	// Generate README.txt
	readmeContent := GenerateReadme(readmeData)

	// Generate README.pdf
	pdfContent, err := pdf.GenerateReadme(pdf.ReadmeData{
		ProjectName:      readmeData.ProjectName,
		Holder:           readmeData.Holder,
		Share:            readmeData.Share,
		OtherFriends:     readmeData.OtherFriends,
		Threshold:        readmeData.Threshold,
		Total:            params.Total,
		Version:          readmeData.Version,
		GitHubReleaseURL: readmeData.GitHubReleaseURL,
		ManifestChecksum: readmeData.ManifestChecksum,
		RecoverChecksum:  readmeData.RecoverChecksum,
		Created:          readmeData.Created,
		Anonymous:        readmeData.Anonymous,
		RecoveryURL:      params.RecoveryURL,
		Language:         params.Language,
	})
	if err != nil {
		return fmt.Errorf("generating PDF: %w", err)
	}

	// Create ZIP with all files, using sealed date as modification time
	readmeFileTxt := translations.ReadmeFilename(params.Language, ".txt")
	readmeFilePdf := translations.ReadmeFilename(params.Language, ".pdf")
	files := []ZipFile{
		{Name: readmeFileTxt, Content: []byte(readmeContent), ModTime: params.SealedAt},
		{Name: readmeFilePdf, Content: pdfContent, ModTime: params.SealedAt},
		{Name: "MANIFEST.age", Content: params.ManifestData, ModTime: params.SealedAt},
		{Name: "recover.html", Content: []byte(params.RecoverHTML), ModTime: params.SealedAt},
	}

	return CreateZip(params.OutputPath, files)
}

// loadShares reads all share files from the project's shares directory.
func loadShares(p *project.Project) ([]*core.Share, error) {
	sharesDir := p.SharesPath()

	shares := make([]*core.Share, len(p.Friends))
	for i, friend := range p.Friends {
		// Try to find share file for this friend
		filename := fmt.Sprintf("SHARE-%s.txt", core.SanitizeFilename(friend.Name))
		sharePath := filepath.Join(sharesDir, filename)

		data, err := os.ReadFile(sharePath)
		if err != nil {
			return nil, fmt.Errorf("reading share for %s: %w", friend.Name, err)
		}

		share, err := core.ParseShare(data)
		if err != nil {
			return nil, fmt.Errorf("parsing share for %s: %w", friend.Name, err)
		}

		shares[i] = share
	}

	return shares, nil
}

// VerifyBundle verifies the integrity of a bundle ZIP file.
// Returns nil if valid, or an error describing the problem.
func VerifyBundle(bundlePath string) error {
	r, err := zip.OpenReader(bundlePath)
	if err != nil {
		return fmt.Errorf("opening bundle: %w", err)
	}
	defer r.Close()

	// Read files from ZIP
	var readmeContent string
	var manifestData []byte
	var recoverData []byte
	var pdfData []byte

	for _, f := range r.File {
		rc, err := f.Open()
		if err != nil {
			return fmt.Errorf("opening %s: %w", f.Name, err)
		}

		data, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return fmt.Errorf("reading %s: %w", f.Name, err)
		}

		switch {
		case translations.IsReadmeFile(f.Name, ".txt"):
			readmeContent = string(data)
		case translations.IsReadmeFile(f.Name, ".pdf"):
			pdfData = data
		case f.Name == "MANIFEST.age":
			manifestData = data
		case f.Name == "recover.html":
			recoverData = data
		}
	}

	if readmeContent == "" {
		return fmt.Errorf("README file (.txt) not found in bundle")
	}
	if len(pdfData) == 0 {
		return fmt.Errorf("README file (.pdf) not found in bundle")
	}
	if len(manifestData) == 0 {
		return fmt.Errorf("MANIFEST.age not found in bundle")
	}
	if len(recoverData) == 0 {
		return fmt.Errorf("recover.html not found in bundle")
	}

	// Parse metadata from footer
	metadata := parseMetadataFooter(readmeContent)

	// Verify manifest checksum
	actualManifestChecksum := core.HashBytes(manifestData)
	expectedManifestChecksum := metadata["checksum-manifest"]
	if expectedManifestChecksum == "" {
		return fmt.Errorf("manifest checksum not found in README metadata")
	}
	if actualManifestChecksum != expectedManifestChecksum {
		return fmt.Errorf("MANIFEST.age checksum mismatch")
	}

	// Verify recover.html checksum
	actualRecoverChecksum := core.HashString(string(recoverData))
	expectedRecoverChecksum := metadata["checksum-recover-html"]
	if expectedRecoverChecksum == "" {
		return fmt.Errorf("recover.html checksum not found in README metadata")
	}
	if actualRecoverChecksum != expectedRecoverChecksum {
		return fmt.Errorf("recover.html checksum mismatch")
	}

	// Verify embedded share
	share, err := core.ParseShare([]byte(readmeContent))
	if err != nil {
		return fmt.Errorf("parsing share: %w", err)
	}

	if err := share.Verify(); err != nil {
		return fmt.Errorf("share verification failed: %w", err)
	}

	return nil
}

// parseMetadataFooter extracts key-value pairs from the README.txt footer section.
func parseMetadataFooter(content string) map[string]string {
	metadata := make(map[string]string)

	footerStart := strings.Index(content, "METADATA FOOTER")
	if footerStart == -1 {
		return metadata
	}

	footer := content[footerStart:]
	lines := strings.Split(footer, "\n")

	keyValueRegex := regexp.MustCompile(`^([a-z0-9-]+):\s*(.+)$`)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		matches := keyValueRegex.FindStringSubmatch(line)
		if len(matches) == 3 {
			metadata[matches[1]] = matches[2]
		}
	}

	return metadata
}
