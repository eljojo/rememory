package cmd

import (
	"os"
	"path/filepath"
	"testing"
	"time"

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

// recover refuses pieces whose Created times differ, so a seal whose loop
// crossed a minute boundary used to produce a set that couldn't be combined.
func TestSealStampsEveryShareWithOneTime(t *testing.T) {
	start := time.Date(2026, 10, 26, 23, 59, 0, 0, time.UTC)
	calls := 0
	sealClock = func() time.Time {
		calls++
		return start.Add(time.Duration(calls) * time.Minute)
	}
	t.Cleanup(func() { sealClock = time.Now })

	dir := t.TempDir()
	p, err := project.New(dir, "test", 2, []project.Friend{{Name: "Ann"}, {Name: "Bo"}, {Name: "Cy"}})
	if err != nil {
		t.Fatalf("creating project: %v", err)
	}
	if err := os.WriteFile(filepath.Join(p.ManifestPath(), "secret.txt"), []byte("secret"), 0600); err != nil {
		t.Fatalf("writing manifest file: %v", err)
	}
	if err := sealProject(p, "", false, ""); err != nil {
		t.Fatalf("sealing: %v", err)
	}

	files, _ := filepath.Glob(filepath.Join(p.SharesPath(), "SHARE-*.txt"))
	shares := make([]*core.Share, len(files))
	for i, f := range files {
		data, _ := os.ReadFile(f)
		if shares[i], err = core.ParseShare(data); err != nil {
			t.Fatalf("parsing %s: %v", f, err)
		}
	}
	if err := validateSharesCompatible(shares); err != nil {
		t.Errorf("shares from one seal are not combinable: %v", err)
	}
	// Every piece carries the seal's own time, not a clock read per share.
	for _, share := range shares {
		if !share.Created.Equal(p.Sealed.At) {
			t.Errorf("share %d created %v, seal at %v", share.Index, share.Created, p.Sealed.At)
		}
	}
	if !p.Sealed.At.Equal(start.Add(time.Minute)) {
		t.Errorf("sealed at %v, want the injected clock's %v", p.Sealed.At, start.Add(time.Minute))
	}
}
