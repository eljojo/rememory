package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/eljojo/rememory/internal/bundle"
	"github.com/eljojo/rememory/internal/project"
)

// Projects sealed before file names were numbered for names without Latin
// letters keep their share at SHARE-.txt; project.yml records where. As long
// as the names didn't collide, `rememory bundle` must still find it.
func TestBundleFindsShareFilesNamedByOlderVersions(t *testing.T) {
	dir := t.TempDir()
	p, err := project.New(dir, "test", 2, []project.Friend{{Name: "Alice"}, {Name: "小明"}, {Name: "Bob"}})
	if err != nil {
		t.Fatalf("creating project: %v", err)
	}
	if err := os.WriteFile(filepath.Join(p.ManifestPath(), "secret.txt"), []byte("secret"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := sealProject(p, "", false, ""); err != nil {
		t.Fatalf("sealing: %v", err)
	}

	// Rename the share as an older version wrote it.
	oldName := filepath.Join(p.SharesPath(), "SHARE-.txt")
	if err := os.Rename(filepath.Join(p.SharesPath(), "SHARE-2.txt"), oldName); err != nil {
		t.Fatal(err)
	}
	p.Sealed.Shares[1].File, _ = filepath.Rel(p.Path, oldName)
	if err := p.Save(); err != nil {
		t.Fatal(err)
	}

	if err := bundle.GenerateAll(p, bundle.Config{Version: "test"}); err != nil {
		t.Errorf("regenerating bundles: %v", err)
	}
}
