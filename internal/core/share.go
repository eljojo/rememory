package core

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

const (
	ShareBegin = "-----BEGIN REMEMORY SHARE-----"
	ShareEnd   = "-----END REMEMORY SHARE-----"

	// DefaultRecoveryURL is the default base URL for QR codes in PDFs.
	// Points to the recover.html hosted on GitHub Pages.
	DefaultRecoveryURL = "https://eljojo.github.io/rememory/recover.html"
)

// Share represents a single Shamir share with metadata.
type Share struct {
	Version   int       // Format version (currently 1)
	Index     int       // Which share (1-indexed for humans)
	Total     int       // Total shares (N)
	Threshold int       // Required shares (K)
	Holder    string    // Name of the person holding this share
	Created   time.Time // When the share was created
	Data      []byte    // The actual share bytes
	Checksum  string    // SHA-256 of Data
}

// NewShare creates a Share with the given parameters and computes its checksum.
func NewShare(index, total, threshold int, holder string, data []byte) *Share {
	return &Share{
		Version:   1,
		Index:     index,
		Total:     total,
		Threshold: threshold,
		Holder:    holder,
		Created:   time.Now().UTC(),
		Data:      data,
		Checksum:  HashBytes(data),
	}
}

// Encode converts the share to a human-readable PEM-like format.
func (s *Share) Encode() string {
	var sb strings.Builder

	sb.WriteString(ShareBegin + "\n")
	sb.WriteString(fmt.Sprintf("Version: %d\n", s.Version))
	sb.WriteString(fmt.Sprintf("Index: %d\n", s.Index))
	sb.WriteString(fmt.Sprintf("Total: %d\n", s.Total))
	sb.WriteString(fmt.Sprintf("Threshold: %d\n", s.Threshold))
	if s.Holder != "" {
		sb.WriteString(fmt.Sprintf("Holder: %s\n", s.Holder))
	}
	sb.WriteString(fmt.Sprintf("Created: %s\n", s.Created.Format(time.RFC3339)))
	sb.WriteString(fmt.Sprintf("Checksum: %s\n", s.Checksum))
	sb.WriteString("\n")
	sb.WriteString(base64.StdEncoding.EncodeToString(s.Data))
	sb.WriteString("\n")
	sb.WriteString(ShareEnd + "\n")

	return sb.String()
}

// ParseShare parses a share from its encoded format.
// The content can be a full README.txt file - it will find the share block.
func ParseShare(content []byte) (*Share, error) {
	text := string(content)

	// Find the PEM block
	beginIdx := strings.Index(text, ShareBegin)
	endIdx := strings.Index(text, ShareEnd)
	if beginIdx == -1 || endIdx == -1 || endIdx <= beginIdx {
		return nil, fmt.Errorf("invalid share format: missing BEGIN/END markers")
	}

	// Extract content between markers
	inner := text[beginIdx+len(ShareBegin) : endIdx]
	lines := strings.Split(strings.TrimSpace(inner), "\n")

	share := &Share{}
	var dataLines []string
	inData := false

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			inData = true
			continue
		}

		if inData {
			dataLines = append(dataLines, line)
			continue
		}

		// Parse header fields
		parts := strings.SplitN(line, ": ", 2)
		if len(parts) != 2 {
			// Line doesn't look like a header - must be base64 data
			// This handles cases where empty line is missing (e.g., copied from PDF)
			inData = true
			dataLines = append(dataLines, line)
			continue
		}

		key, value := parts[0], parts[1]
		switch key {
		case "Version":
			v, err := strconv.Atoi(value)
			if err != nil {
				return nil, fmt.Errorf("invalid version: %w", err)
			}
			share.Version = v
		case "Index":
			v, err := strconv.Atoi(value)
			if err != nil {
				return nil, fmt.Errorf("invalid index: %w", err)
			}
			share.Index = v
		case "Total":
			v, err := strconv.Atoi(value)
			if err != nil {
				return nil, fmt.Errorf("invalid total: %w", err)
			}
			share.Total = v
		case "Threshold":
			v, err := strconv.Atoi(value)
			if err != nil {
				return nil, fmt.Errorf("invalid threshold: %w", err)
			}
			share.Threshold = v
		case "Holder":
			share.Holder = value
		case "Created":
			t, err := time.Parse(time.RFC3339, value)
			if err != nil {
				return nil, fmt.Errorf("invalid created time: %w", err)
			}
			share.Created = t
		case "Checksum":
			share.Checksum = value
		}
	}

	// Decode base64 data
	dataStr := strings.Join(dataLines, "")
	data, err := base64.StdEncoding.DecodeString(dataStr)
	if err != nil {
		return nil, fmt.Errorf("invalid base64 data: %w", err)
	}
	share.Data = data

	// Validate required fields
	if share.Version == 0 {
		return nil, fmt.Errorf("missing version")
	}
	if share.Index == 0 {
		return nil, fmt.Errorf("missing index")
	}
	if share.Total == 0 {
		return nil, fmt.Errorf("missing total")
	}
	if share.Threshold == 0 {
		return nil, fmt.Errorf("missing threshold")
	}
	if len(share.Data) == 0 {
		return nil, fmt.Errorf("missing share data")
	}

	return share, nil
}

// Verify checks that the share's checksum matches its data.
// Uses constant-time comparison to prevent timing attacks.
func (s *Share) Verify() error {
	if s.Checksum == "" {
		return nil // No checksum to verify
	}
	computed := HashBytes(s.Data)
	if !VerifyHash(computed, s.Checksum) {
		return fmt.Errorf("share checksum verification failed")
	}
	return nil
}

// CompactEncode returns a short string encoding of the share suitable for
// QR codes and URL fragments. Format: RM{version}:{index}:{total}:{threshold}:{base64url_data}:{short_check}
// The short_check is the first 4 hex characters of the SHA-256 of the raw share data.
func (s *Share) CompactEncode() string {
	data := base64.RawURLEncoding.EncodeToString(s.Data)
	check := shortChecksum(s.Data)
	return fmt.Sprintf("RM%d:%d:%d:%d:%s:%s", s.Version, s.Index, s.Total, s.Threshold, data, check)
}

// ParseCompact parses a compact-encoded share string back into a Share.
// It validates the format, decodes the data, and verifies the short checksum.
func ParseCompact(s string) (*Share, error) {
	parts := strings.Split(s, ":")
	if len(parts) != 6 {
		return nil, fmt.Errorf("invalid compact share: expected 6 colon-separated fields, got %d", len(parts))
	}

	prefix := parts[0]
	if !strings.HasPrefix(prefix, "RM") {
		return nil, fmt.Errorf("invalid compact share: must start with 'RM', got %q", prefix)
	}

	version, err := strconv.Atoi(prefix[2:])
	if err != nil || version < 1 {
		return nil, fmt.Errorf("invalid compact share: bad version %q", prefix[2:])
	}

	index, err := strconv.Atoi(parts[1])
	if err != nil || index < 1 {
		return nil, fmt.Errorf("invalid compact share: bad index %q", parts[1])
	}

	total, err := strconv.Atoi(parts[2])
	if err != nil || total < 1 {
		return nil, fmt.Errorf("invalid compact share: bad total %q", parts[2])
	}

	threshold, err := strconv.Atoi(parts[3])
	if err != nil || threshold < 1 {
		return nil, fmt.Errorf("invalid compact share: bad threshold %q", parts[3])
	}

	data, err := base64.RawURLEncoding.DecodeString(parts[4])
	if err != nil {
		return nil, fmt.Errorf("invalid compact share: bad base64 data: %w", err)
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("invalid compact share: empty data")
	}

	// Verify short checksum
	expectedCheck := shortChecksum(data)
	if parts[5] != expectedCheck {
		return nil, fmt.Errorf("invalid compact share: checksum mismatch (got %s, want %s)", parts[5], expectedCheck)
	}

	return &Share{
		Version:   version,
		Index:     index,
		Total:     total,
		Threshold: threshold,
		Data:      data,
		Checksum:  HashBytes(data),
	}, nil
}

// shortChecksum returns the first 4 hex characters of the SHA-256 of data.
func shortChecksum(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:2])
}

// Filename returns a suggested filename for this share.
func (s *Share) Filename() string {
	name := s.Holder
	if name == "" {
		name = fmt.Sprintf("%d", s.Index)
	}
	name = SanitizeFilename(name)
	return fmt.Sprintf("SHARE-%s.txt", name)
}

// SanitizeFilename converts a name to a filesystem-safe lowercase ASCII string.
// It transliterates accented/diacritic characters to their ASCII base form
// (e.g. "José" → "jose", "Müller" → "muller") using NFD decomposition.
func SanitizeFilename(name string) string {
	// NFD decompose: split characters like "é" into "e" + combining accent,
	// then drop combining marks to keep only the base letter.
	var stripped []rune
	for _, r := range norm.NFD.String(name) {
		if !unicode.Is(unicode.Mn, r) {
			stripped = append(stripped, r)
		}
	}

	// Keep alphanumeric, convert spaces/hyphens/underscores to hyphen, drop rest.
	var b strings.Builder
	for _, r := range stripped {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		} else if r == ' ' || r == '-' || r == '_' {
			b.WriteRune('-')
		}
	}

	result := strings.ToLower(b.String())

	// Collapse consecutive hyphens and trim leading/trailing hyphens.
	for strings.Contains(result, "--") {
		result = strings.ReplaceAll(result, "--", "-")
	}
	result = strings.Trim(result, "-")

	return result
}
