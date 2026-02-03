package shamir

import (
	"encoding/base64"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/eljojo/rememory/internal/crypto"
)

const (
	shareBegin = "-----BEGIN REMEMORY SHARE-----"
	shareEnd   = "-----END REMEMORY SHARE-----"
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
		Checksum:  crypto.HashBytes(data),
	}
}

// Encode converts the share to a human-readable PEM-like format.
func (s *Share) Encode() string {
	var sb strings.Builder

	sb.WriteString(shareBegin + "\n")
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
	sb.WriteString(shareEnd + "\n")

	return sb.String()
}

// ParseShare parses a share from its encoded format.
func ParseShare(content []byte) (*Share, error) {
	text := string(content)

	// Find the PEM block
	beginIdx := strings.Index(text, shareBegin)
	endIdx := strings.Index(text, shareEnd)
	if beginIdx == -1 || endIdx == -1 || endIdx <= beginIdx {
		return nil, fmt.Errorf("invalid share format: missing BEGIN/END markers")
	}

	// Extract content between markers
	inner := text[beginIdx+len(shareBegin) : endIdx]
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
	computed := crypto.HashBytes(s.Data)
	if !crypto.VerifyHash(computed, s.Checksum) {
		return fmt.Errorf("share checksum verification failed")
	}
	return nil
}

// Filename returns a suggested filename for this share.
func (s *Share) Filename() string {
	name := s.Holder
	if name == "" {
		name = fmt.Sprintf("%d", s.Index)
	}
	// Sanitize the name for filesystem use
	name = sanitizeFilename(name)
	return fmt.Sprintf("SHARE-%s.txt", strings.ToLower(name))
}

// sanitizeFilename removes characters that are problematic in filenames.
func sanitizeFilename(name string) string {
	// Replace spaces with hyphens, remove other problematic chars
	name = strings.ReplaceAll(name, " ", "-")
	reg := regexp.MustCompile(`[^a-zA-Z0-9\-_]`)
	return reg.ReplaceAllString(name, "")
}
