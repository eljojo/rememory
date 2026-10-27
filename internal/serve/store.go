package serve

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"sync"
	"time"
)

// BundleMeta holds non-secret metadata about a stored manifest.
type BundleMeta struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Created      string `json:"created"` // RFC 3339
	Threshold    int    `json:"threshold"`
	Total        int    `json:"total"`
	ManifestSize int    `json:"manifestSize"`
}

// Store manages bundle storage on the filesystem.
// Data directory layout:
//
//	<data-dir>/
//	  admin.age              # age-encrypted known plaintext (admin password verification)
//	  bundles/
//	    <uuid>/
//	      meta.json          # BundleMeta
//	      MANIFEST.age       # The encrypted archive
type Store struct {
	dir string
	mu  sync.RWMutex
}

// maxMetaFileSize bounds a stored meta.json: a 4 KiB meta field can grow
// up to 6x when re-encoded (JSON escapes <, > and & as \u003c).
const maxMetaFileSize = 32 << 10

var uuidRegex = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

// NewStore creates a Store backed by the given directory.
// The directory is created if it doesn't exist.
func NewStore(dir string) (*Store, error) {
	bundlesDir := filepath.Join(dir, "bundles")
	if err := os.MkdirAll(bundlesDir, 0700); err != nil {
		return nil, fmt.Errorf("creating data directory: %w", err)
	}
	return &Store{dir: dir}, nil
}

// Save stores a manifest and its metadata, returning the generated UUID.
func (s *Store) Save(manifest []byte, meta BundleMeta) (string, error) {
	id, err := generateUUID()
	if err != nil {
		return "", fmt.Errorf("generating bundle ID: %w", err)
	}

	meta.ID = id
	meta.Created = time.Now().UTC().Format(time.RFC3339)
	meta.ManifestSize = len(manifest)

	bundleDir := filepath.Join(s.dir, "bundles", id)

	s.mu.Lock()
	defer s.mu.Unlock()

	if err := os.MkdirAll(bundleDir, 0700); err != nil {
		return "", fmt.Errorf("creating bundle directory: %w", err)
	}

	// Write metadata
	metaJSON, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshaling metadata: %w", err)
	}
	if err := os.WriteFile(filepath.Join(bundleDir, "meta.json"), metaJSON, 0600); err != nil {
		return "", fmt.Errorf("writing metadata: %w", err)
	}

	// Write manifest
	if err := os.WriteFile(filepath.Join(bundleDir, "MANIFEST.age"), manifest, 0600); err != nil {
		return "", fmt.Errorf("writing manifest: %w", err)
	}

	return id, nil
}

// Delete removes a bundle by ID.
func (s *Store) Delete(id string) error {
	if !isValidUUID(id) {
		return fmt.Errorf("invalid bundle ID")
	}

	bundleDir := filepath.Join(s.dir, "bundles", id)

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, err := os.Stat(bundleDir); os.IsNotExist(err) {
		return fmt.Errorf("bundle not found")
	}

	if err := os.RemoveAll(bundleDir); err != nil {
		return fmt.Errorf("deleting bundle: %w", err)
	}

	return nil
}

// List returns metadata for all bundles, sorted by creation time (newest first).
func (s *Store) List() ([]BundleMeta, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	bundlesDir := filepath.Join(s.dir, "bundles")
	entries, err := os.ReadDir(bundlesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading bundles directory: %w", err)
	}

	var metas []BundleMeta
	for _, entry := range entries {
		if !entry.IsDir() || !isValidUUID(entry.Name()) {
			continue
		}
		metaPath := filepath.Join(bundlesDir, entry.Name(), "meta.json")
		// Skip oversized metadata, which older versions accepted from
		// anonymous uploads; this runs on every page view.
		if info, err := os.Stat(metaPath); err != nil || info.Size() > maxMetaFileSize {
			continue
		}
		data, err := os.ReadFile(metaPath)
		if err != nil {
			continue
		}
		var meta BundleMeta
		if err := json.Unmarshal(data, &meta); err != nil {
			continue
		}
		metas = append(metas, meta)
	}

	// Sort by creation time, newest first
	sort.Slice(metas, func(i, j int) bool {
		return metas[i].Created > metas[j].Created
	})

	return metas, nil
}

// Latest returns the metadata for the most recently created bundle, or nil if none exist.
func (s *Store) Latest() (*BundleMeta, error) {
	metas, err := s.List()
	if err != nil {
		return nil, err
	}
	if len(metas) == 0 {
		return nil, nil
	}
	return &metas[0], nil
}

// ManifestPath returns the filesystem path to a bundle's MANIFEST.age.
func (s *Store) ManifestPath(id string) string {
	return filepath.Join(s.dir, "bundles", id, "MANIFEST.age")
}

// HasManifest returns true if at least one bundle exists.
func (s *Store) HasManifest() bool {
	meta, _ := s.Latest()
	return meta != nil
}

// AdminFilePath returns the path to the admin.age password file.
func (s *Store) AdminFilePath() string {
	return filepath.Join(s.dir, "admin.age")
}

// isValidUUID validates a v4 UUID string to prevent path traversal.
func isValidUUID(s string) bool {
	return uuidRegex.MatchString(s)
}

// generateUUID generates a random v4 UUID.
func generateUUID() (string, error) {
	var uuid [16]byte
	if _, err := rand.Read(uuid[:]); err != nil {
		return "", err
	}
	uuid[6] = (uuid[6] & 0x0f) | 0x40 // version 4
	uuid[8] = (uuid[8] & 0x3f) | 0x80 // variant 10
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:16]), nil
}
