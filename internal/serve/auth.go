package serve

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/eljojo/rememory/internal/core"
	"github.com/eljojo/rememory/internal/crypto"
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

	if IsSetup(store) {
		return ErrAlreadySet
	}
	return writePassword(store, password, false)
}

// writePassword encrypts the known plaintext with password and stores it.
// Without replace, an existing admin.age is kept (ErrAlreadySet).
func writePassword(store *Store, password string, replace bool) error {
	path := store.AdminFilePath()
	var buf bytes.Buffer
	if err := core.Encrypt(&buf, strings.NewReader(knownPlaintext), password); err != nil {
		return fmt.Errorf("encrypting admin password: %w", err)
	}

	// Write a complete temp file, then move it into place, so a crash never
	// leaves a half-written admin.age. Without replace it is linked, which
	// fails if admin.age already exists: the first password stays.
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
	if replace {
		if err := os.Rename(tmp.Name(), path); err != nil {
			return fmt.Errorf("writing admin password file: %w", err)
		}
		return nil
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

// EnsureAdminPassword prepares the admin password before the server starts.
// envPassword (REMEMORY_ADMIN_PASSWORD) is stored, replacing a different
// one. Otherwise a server listening beyond loopback gets a generated
// password, returned so it can be shown once; until then, its setup page
// would let the first visitor choose. Local servers keep the setup page.
func EnsureAdminPassword(store *Store, host, envPassword string) (generated string, err error) {
	if envPassword != "" {
		if len(envPassword) > maxPasswordLen {
			return "", fmt.Errorf("REMEMORY_ADMIN_PASSWORD is longer than %d bytes, the most the server accepts", maxPasswordLen)
		}
		if IsSetup(store) && CheckPassword(store, envPassword) {
			return "", nil
		}
		return "", writePassword(store, envPassword, true)
	}
	if IsSetup(store) || isLoopback(host) {
		return "", nil
	}
	password, err := crypto.GeneratePassphrase(16)
	if err != nil {
		return "", fmt.Errorf("generating admin password: %w", err)
	}
	if err := SetPassword(store, password); err != nil {
		return "", err
	}
	return password, nil
}

// isLoopback reports whether host only accepts connections from this machine.
// An empty host listens on every interface.
func isLoopback(host string) bool {
	host = strings.Trim(host, "[]")
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
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
