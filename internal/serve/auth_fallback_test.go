package serve

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// Without hard links (SMB shares, FAT/exFAT, some FUSE mounts), setup falls
// back to an exclusive create: the first password stays, nothing else wins.
func TestCreateExclusiveKeepsTheFirst(t *testing.T) {
	path := filepath.Join(t.TempDir(), "admin.age")
	if err := createExclusive(path, []byte("first")); err != nil {
		t.Fatalf("first create: %v", err)
	}
	if err := createExclusive(path, []byte("second")); !errors.Is(err, ErrAlreadySet) {
		t.Errorf("second create: got %v, want ErrAlreadySet", err)
	}
	data, _ := os.ReadFile(path)
	if string(data) != "first" {
		t.Errorf("content %q, want the first", data)
	}
	if info, _ := os.Stat(path); info.Mode().Perm() != 0600 {
		t.Errorf("mode %v, want 0600", info.Mode().Perm())
	}
}
