package bundle

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// ZipFile represents a file to be added to a ZIP archive.
type ZipFile struct {
	Name    string
	Content []byte
	ModTime time.Time
}

// CreateZip creates a ZIP archive at the given path with the given files.
// It is written with WritePrivateFile: bundles carry a share.
func CreateZip(path string, files []ZipFile) error {
	return WritePrivateFile(path, func(f io.Writer) error {
		w := zip.NewWriter(f)
		for _, file := range files {
			header := &zip.FileHeader{
				Name:   file.Name,
				Method: zip.Deflate,
			}
			header.Modified = file.ModTime

			fw, err := w.CreateHeader(header)
			if err != nil {
				return fmt.Errorf("creating entry %s: %w", file.Name, err)
			}

			if _, err := fw.Write(file.Content); err != nil {
				return fmt.Errorf("writing entry %s: %w", file.Name, err)
			}
		}
		return w.Close()
	})
}

// WritePrivateFile writes path readable only by its owner, replacing it
// atomically: the content goes to a new temporary file in the same folder,
// which is then renamed over path. An interrupted write leaves the old file
// whole, and a program that opened the old file (permissions are checked
// only on open) keeps reading the old contents, not the new.
func WritePrivateFile(path string, write func(io.Writer) error) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".*.tmp") // created 0600
	if err != nil {
		return fmt.Errorf("creating %s: %w", filepath.Base(path), err)
	}
	defer os.Remove(tmp.Name()) // fails harmlessly once renamed
	if err := write(tmp); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return fmt.Errorf("writing %s: %w", filepath.Base(path), err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("writing %s: %w", filepath.Base(path), err)
	}
	if err := os.Rename(tmp.Name(), path); err != nil {
		return fmt.Errorf("replacing %s: %w", filepath.Base(path), err)
	}
	return nil
}
