package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/eljojo/rememory/internal/core"
	"github.com/eljojo/rememory/internal/project"
)

// Friends whose names have no Latin letters used to all get SHARE-.txt and
// bundle-.zip, each overwriting the last, leaving one share on disk.
func TestSealNonLatinNamesGetDistinctFiles(t *testing.T) {
	dir := t.TempDir()
	friends := []project.Friend{{Name: "小明"}, {Name: "小華"}, {Name: "小美"}}
	p, err := project.New(dir, "test", 2, friends)
	if err != nil {
		t.Fatalf("creating project: %v", err)
	}
	if err := os.WriteFile(filepath.Join(p.ManifestPath(), "secret.txt"), []byte("secret"), 0600); err != nil {
		t.Fatalf("writing manifest file: %v", err)
	}

	if err := sealProject(p, "", false, ""); err != nil {
		t.Fatalf("sealing: %v", err)
	}

	shareFiles, _ := filepath.Glob(filepath.Join(p.SharesPath(), "SHARE-*.txt"))
	if len(shareFiles) != len(friends) {
		t.Fatalf("got %d share files, want %d: %v", len(shareFiles), len(friends), shareFiles)
	}
	seen := map[int]bool{}
	for _, f := range shareFiles {
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("reading %s: %v", f, err)
		}
		share, err := core.ParseShare(data)
		if err != nil {
			t.Fatalf("parsing %s: %v", f, err)
		}
		seen[share.Index] = true
	}
	if len(seen) != len(friends) {
		t.Errorf("share files hold %d distinct indices, want %d", len(seen), len(friends))
	}

	bundles, _ := filepath.Glob(filepath.Join(p.OutputPath(), "bundles", "bundle-*.zip"))
	if len(bundles) != len(friends) {
		t.Errorf("got %d bundles, want %d: %v", len(bundles), len(friends), bundles)
	}
}
