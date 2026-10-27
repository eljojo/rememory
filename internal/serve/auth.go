package serve

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/eljojo/rememory/internal/core"
)

// knownPlaintext is the fixed string encrypted with the admin password.
// Verification works by decrypting admin.age and checking the result matches.
const knownPlaintext = "rememory-admin"

// ErrAlreadySet is returned by SetPassword when an admin password exists.
var ErrAlreadySet = errors.New("admin password is already set")

// SetPassword encrypts a known plaintext with the given password and saves it.
// Returns ErrAlreadySet if a password is already set, even when another
// process sets one while this call is running.
func SetPassword(store *Store, password string) error {
	if password == "" {
		return fmt.Errorf("password cannot be empty")
	}

	path := store.AdminFilePath()
	if _, err := os.Stat(path); err == nil {
		return ErrAlreadySet
	}

	var buf bytes.Buffer
	if err := core.Encrypt(&buf, strings.NewReader(knownPlaintext), password); err != nil {
		return fmt.Errorf("encrypting admin password: %w", err)
	}

	// Write a complete temp file, then link it into place: the link fails if
	// admin.age already exists, so the first password stays, and a crash
	// never leaves a half-written admin.age.
	tmp, err := os.CreateTemp(filepath.Dir(path), ".admin-*.age")
	if err != nil {
		return fmt.Errorf("writing admin password file: %w", err)
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(buf.Bytes()); err != nil {
		tmp.Close()
		return fmt.Errorf("writing admin password file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("writing admin password file: %w", err)
	}
	if err := os.Link(tmp.Name(), path); err != nil {
		if errors.Is(err, fs.ErrExist) {
			return ErrAlreadySet
		}
		// Some filesystems have no hard links (SMB shares, FAT/exFAT, some
		// FUSE mounts); an exclusive create still keeps the first password.
		return createExclusive(path, buf.Bytes())
	}

	return nil
}

// CheckPassword verifies the given password against the stored admin password.
func CheckPassword(store *Store, password string) bool {
	if password == "" {
		return false
	}

	path := store.AdminFilePath()
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}

	decrypted, err := core.DecryptBytes(data, password)
	if err != nil {
		return false
	}

	return string(decrypted) == knownPlaintext
}

// createExclusive writes data to path only if path doesn't exist yet. It's
// the fallback where hard links aren't available; unlike the link, a crash
// mid-write could leave a partial file, which a restart then reports.
func createExclusive(path string, data []byte) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		if errors.Is(err, fs.ErrExist) {
			return ErrAlreadySet
		}
		return fmt.Errorf("writing admin password file: %w", err)
	}
	if _, err := f.Write(data); err != nil {
		f.Close()
		os.Remove(path)
		return fmt.Errorf("writing admin password file: %w", err)
	}
	if err := f.Close(); err != nil {
		os.Remove(path)
		return fmt.Errorf("writing admin password file: %w", err)
	}
	return nil
}

// IsSetup returns true if an admin password has been configured.
func IsSetup(store *Store) bool {
	_, err := os.Stat(store.AdminFilePath())
	return err == nil
}
