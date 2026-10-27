package bundle

import (
	"archive/zip"
	"encoding/base64"
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
	Version         string // Tool version (e.g., "v1.0.0")
	RecoveryURL     string // Optional: base URL for QR code (e.g. "https://example.com/recover.html")
	NoEmbedManifest bool   // If true, do not embed MANIFEST.age in recover.html even when small enough
	TlockEnabled    bool   // If true, bundles include tlock-js for time-lock decryption
}

// GenerateAll creates bundles for all friends in the project.
func GenerateAll(p *project.Project, cfg Config) error {
	html.SetVersion(cfg.Version)

	if p.Sealed == nil {
		return fmt.Errorf("project must be sealed before generating bundles")
	}

	// Each bundle holds a share, so the folder is private like shares/.
	bundlesDir := filepath.Join(p.OutputPath(), "bundles")
	if err := os.MkdirAll(bundlesDir, 0700); err != nil {
		return fmt.Errorf("creating bundles directory: %w", err)
	}
	// A folder owned by another account (e.g. made by a root container) or
	// on a filesystem without permissions can't be tightened; say so and go on.
	if err := os.Chmod(bundlesDir, 0700); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: couldn't make %s private (%v); other accounts on this computer may be able to read it.\n", bundlesDir, err)
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

	// Derive GitHub release URL from version for README/PDF
	githubReleaseURL := fmt.Sprintf("%s/releases/tag/%s", core.GitHubRepo, cfg.Version)

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
		// Empty, not nil, for anonymous bundles: recover.html reads it as an array.
		otherFriendsInfo := []html.FriendInfo{}
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
			TlockEnabled: cfg.TlockEnabled,
		}

		// Embed manifest in recover.html when small enough and not disabled
		manifestEmbedded := !cfg.NoEmbedManifest && len(manifestData) <= html.MaxEmbeddedManifestSize
		if manifestEmbedded {
			personalization.ManifestB64 = base64.StdEncoding.EncodeToString(manifestData)
		}

		recoverHTML := html.GenerateRecoverHTML(personalization)
		recoverChecksum := core.HashString(recoverHTML)

		bundlePath := filepath.Join(bundlesDir, fmt.Sprintf("bundle-%s.zip", core.FriendFileStem(friend.Name, i+1)))

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
			ManifestEmbedded: manifestEmbedded,
			RecoverHTML:      recoverHTML,
			RecoverChecksum:  recoverChecksum,
			Version:          cfg.Version,
			GitHubReleaseURL: githubReleaseURL,
			SealedAt:         p.Sealed.At,
			Anonymous:        p.Anonymous,
			RecoveryURL:      cfg.RecoveryURL,
			Language:         lang,
			TlockEnabled:     cfg.TlockEnabled,
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
	ManifestEmbedded bool // true when manifest is base64-embedded in recover.html
	RecoverHTML      string
	RecoverChecksum  string
	Version          string
	GitHubReleaseURL string
	SealedAt         time.Time
	Anonymous        bool
	RecoveryURL      string
	Language         string // Bundle language for this friend
	TlockEnabled     bool   // true when manifest uses time-lock encryption
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
		ManifestEmbedded: params.ManifestEmbedded,
		TlockEnabled:     params.TlockEnabled,
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
		ManifestEmbedded: params.ManifestEmbedded,
		TlockEnabled:     params.TlockEnabled,
	})
	if err != nil {
		return fmt.Errorf("generating PDF: %w", err)
	}

	// Create ZIP with all files, using sealed date as modification time.
	// When the manifest is embedded in recover.html, skip the separate MANIFEST.age
	// file to avoid duplicating data and inflating the ZIP size.
	readmeFileTxt := translations.ReadmeFilename(params.Language, ".txt")
	readmeFilePdf := translations.ReadmeFilename(params.Language, ".pdf")
	files := []ZipFile{
		{Name: readmeFileTxt, Content: []byte(readmeContent), ModTime: params.SealedAt},
		{Name: readmeFilePdf, Content: pdfContent, ModTime: params.SealedAt},
		{Name: "recover.html", Content: []byte(params.RecoverHTML), ModTime: params.SealedAt},
	}
	if !params.ManifestEmbedded {
		files = append(files, ZipFile{Name: "MANIFEST.age", Content: params.ManifestData, ModTime: params.SealedAt})
	}

	return CreateZip(params.OutputPath, files)
}

// loadShares reads all share files from the project's shares directory.
func loadShares(p *project.Project) ([]*core.Share, error) {
	shares := make([]*core.Share, len(p.Friends))
	for i, friend := range p.Friends {
		sharePath := p.SharePath(i)
		filename := filepath.Base(sharePath)

		data, err := os.ReadFile(sharePath)
		if err != nil {
			return nil, fmt.Errorf("reading share for %s: %w", friend.Name, err)
		}

		share, err := core.ParseShare(data)
		if err != nil {
			return nil, fmt.Errorf("parsing share for %s: %w", friend.Name, err)
		}
		// Guards against a share file overwritten by another friend's, which
		// older versions allowed when two names produced the same file name.
		if share.Index != i+1 {
			return nil, fmt.Errorf("share file %s holds piece #%d, but %s is friend #%d — run 'rememory seal' again", filename, share.Index, friend.Name, i+1)
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
		if closeErr := rc.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
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
	if len(recoverData) == 0 {
		return fmt.Errorf("recover.html not found in bundle")
	}

	// When MANIFEST.age is not in the ZIP, the manifest is embedded in recover.html.
	// Extract it from there for checksum verification.
	if len(manifestData) == 0 {
		extracted, err := html.ExtractManifestFromHTML(recoverData)
		if err != nil {
			return fmt.Errorf("MANIFEST.age not in bundle and could not extract from recover.html: %w", err)
		}
		manifestData = extracted
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
