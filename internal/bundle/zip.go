package bundle

import (
	"archive/zip"
	"fmt"
	"os"
	"time"
)

// ZipFile represents a file to be added to a ZIP archive.
type ZipFile struct {
	Name    string
	Content []byte
	ModTime time.Time
}

// CreateZip creates a ZIP archive at the given path with the given files.
func CreateZip(path string, files []ZipFile) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("creating zip file: %w", err)
	}
	defer f.Close()

	w := zip.NewWriter(f)
	defer w.Close()

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

	return nil
}
