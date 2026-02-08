//go:build js && wasm && create

package main

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"fmt"
	"syscall/js"
	"time"
	"unicode"

	"golang.org/x/text/unicode/norm"

	"github.com/eljojo/rememory/internal/bundle"
	"github.com/eljojo/rememory/internal/core"
	"github.com/eljojo/rememory/internal/crypto"
	"github.com/eljojo/rememory/internal/html"
	"github.com/eljojo/rememory/internal/pdf"
	"github.com/eljojo/rememory/internal/project"

	"gopkg.in/yaml.v3"
)

// FileEntry represents a file passed from JavaScript.
type FileEntry struct {
	Name string // Relative path (e.g., "manifest/secrets.txt")
	Data []byte
}

// FriendInput represents friend data from JavaScript.
type FriendInput struct {
	Name  string
	Email string
	Phone string
}

// CreateBundlesConfig holds all parameters for bundle creation.
type CreateBundlesConfig struct {
	ProjectName string
	Threshold   int
	Friends     []FriendInput
	Files       []FileEntry
	Version     string
	GitHubURL   string
	Anonymous   bool
}

// BundleOutput represents a generated bundle for JavaScript.
type BundleOutput struct {
	FriendName string
	FileName   string
	Data       []byte
}

// createBundlesJS is the WASM entry point for bundle creation.
// Args: config object with projectName, threshold, friends, files, version, githubURL
// Returns: { bundles: [...], error: string|null }
func createBundlesJS(this js.Value, args []js.Value) any {
	if len(args) < 1 {
		return errorResult("missing config argument")
	}

	configJS := args[0]

	// Parse configuration from JavaScript
	config := CreateBundlesConfig{
		ProjectName: configJS.Get("projectName").String(),
		Threshold:   configJS.Get("threshold").Int(),
		Version:     configJS.Get("version").String(),
		GitHubURL:   configJS.Get("githubURL").String(),
		Anonymous:   configJS.Get("anonymous").Bool(),
	}

	// Parse friends array
	friendsJS := configJS.Get("friends")
	friendsLen := friendsJS.Length()
	config.Friends = make([]FriendInput, friendsLen)
	for i := 0; i < friendsLen; i++ {
		f := friendsJS.Index(i)
		config.Friends[i] = FriendInput{
			Name:  f.Get("name").String(),
			Email: f.Get("email").String(),
		}
		if phone := f.Get("phone"); !phone.IsUndefined() && !phone.IsNull() {
			config.Friends[i].Phone = phone.String()
		}
	}

	// Parse files array
	filesJS := configJS.Get("files")
	filesLen := filesJS.Length()
	config.Files = make([]FileEntry, filesLen)
	for i := 0; i < filesLen; i++ {
		f := filesJS.Index(i)
		name := f.Get("name").String()
		dataJS := f.Get("data")
		dataLen := dataJS.Get("length").Int()
		data := make([]byte, dataLen)
		js.CopyBytesToGo(data, dataJS)
		config.Files[i] = FileEntry{
			Name: name,
			Data: data,
		}
	}

	// Create bundles
	bundles, err := createBundles(config)
	if err != nil {
		return errorResult(err.Error())
	}

	// Convert bundles to JavaScript array
	jsBundles := make([]any, len(bundles))
	for i, b := range bundles {
		jsData := js.Global().Get("Uint8Array").New(len(b.Data))
		js.CopyBytesToJS(jsData, b.Data)
		jsBundles[i] = map[string]any{
			"friendName": b.FriendName,
			"fileName":   b.FileName,
			"data":       jsData,
		}
	}

	return js.ValueOf(map[string]any{
		"bundles": jsBundles,
		"error":   nil,
	})
}

// createBundles creates bundles for all friends.
func createBundles(config CreateBundlesConfig) ([]BundleOutput, error) {
	// Validate inputs
	if config.ProjectName == "" {
		return nil, fmt.Errorf("project name is required")
	}
	if len(config.Friends) < 2 {
		return nil, fmt.Errorf("need at least 2 friends, got %d", len(config.Friends))
	}
	if config.Threshold < 2 {
		return nil, fmt.Errorf("threshold must be at least 2, got %d", config.Threshold)
	}
	if config.Threshold > len(config.Friends) {
		return nil, fmt.Errorf("threshold (%d) cannot exceed number of friends (%d)", config.Threshold, len(config.Friends))
	}
	if len(config.Files) == 0 {
		return nil, fmt.Errorf("no files provided")
	}

	// Validate friends (email not required for anonymous mode)
	for i, f := range config.Friends {
		if f.Name == "" {
			return nil, fmt.Errorf("friend %d: name is required", i+1)
		}
		if !config.Anonymous && f.Email == "" {
			return nil, fmt.Errorf("friend %d (%s): email is required", i+1, f.Name)
		}
	}

	// Create tar.gz archive of files
	archiveData, err := createTarGz(config.Files)
	if err != nil {
		return nil, fmt.Errorf("creating archive: %w", err)
	}

	// Generate random passphrase
	passphrase, err := crypto.GeneratePassphrase(crypto.DefaultPassphraseBytes)
	if err != nil {
		return nil, fmt.Errorf("generating passphrase: %w", err)
	}

	// Encrypt archive
	var encryptedBuf bytes.Buffer
	if err := core.Encrypt(&encryptedBuf, bytes.NewReader(archiveData), passphrase); err != nil {
		return nil, fmt.Errorf("encrypting archive: %w", err)
	}
	manifestData := encryptedBuf.Bytes()
	manifestChecksum := core.HashBytes(manifestData)

	// Split passphrase using Shamir's Secret Sharing
	n := len(config.Friends)
	k := config.Threshold
	rawShares, err := core.Split([]byte(passphrase), n, k)
	if err != nil {
		return nil, fmt.Errorf("splitting passphrase: %w", err)
	}

	// Current timestamp for all bundles
	now := time.Now().UTC()

	// Get recovery WASM bytes for embedding in recover.html
	// Note: In WASM context, we use the embedded recover.wasm (smaller, recovery-only)
	wasmBytes := html.GetRecoverWASMBytes()

	// Create shares and bundles
	bundles := make([]BundleOutput, n)
	shares := make([]*core.Share, n)

	// Create all shares first
	for i, friend := range config.Friends {
		share := &core.Share{
			Version:   1,
			Index:     i + 1,
			Total:     n,
			Threshold: k,
			Holder:    friend.Name,
			Created:   now,
			Data:      rawShares[i],
			Checksum:  core.HashBytes(rawShares[i]),
		}
		shares[i] = share
	}

	// Convert friends to project.Friend for bundle generation
	projectFriends := make([]project.Friend, len(config.Friends))
	for i, f := range config.Friends {
		projectFriends[i] = project.Friend{
			Name:  f.Name,
			Email: f.Email,
			Phone: f.Phone,
		}
	}

	// Generate bundle for each friend
	for i, friend := range config.Friends {
		share := shares[i]

		// Get other friends (excluding this one) - empty for anonymous mode
		var otherFriends []project.Friend
		var otherFriendsInfo []html.FriendInfo
		if !config.Anonymous {
			otherFriends = make([]project.Friend, 0, n-1)
			for j, f := range projectFriends {
				if j != i {
					otherFriends = append(otherFriends, f)
				}
			}

			// Convert to FriendInfo for HTML personalization
			otherFriendsInfo = make([]html.FriendInfo, len(otherFriends))
			for j, f := range otherFriends {
				otherFriendsInfo[j] = html.FriendInfo{
					Name:  f.Name,
					Email: f.Email,
					Phone: f.Phone,
				}
			}
		}

		// Generate personalized recover.html
		personalization := &html.PersonalizationData{
			Holder:       friend.Name,
			HolderShare:  share.Encode(),
			OtherFriends: otherFriendsInfo,
			Threshold:    k,
			Total:        n,
		}
		recoverHTML := html.GenerateRecoverHTML(wasmBytes, config.Version, config.GitHubURL, personalization)
		recoverChecksum := core.HashString(recoverHTML)

		// Generate README.txt
		readmeData := bundle.ReadmeData{
			ProjectName:      config.ProjectName,
			Holder:           friend.Name,
			Share:            share,
			OtherFriends:     otherFriends,
			Threshold:        k,
			Total:            n,
			Version:          config.Version,
			GitHubReleaseURL: config.GitHubURL,
			ManifestChecksum: manifestChecksum,
			RecoverChecksum:  recoverChecksum,
			Created:          now,
			Anonymous:        config.Anonymous,
		}
		readmeContent := bundle.GenerateReadme(readmeData)

		// Generate README.pdf
		pdfData := pdf.ReadmeData{
			ProjectName:      config.ProjectName,
			Holder:           friend.Name,
			Share:            share,
			OtherFriends:     otherFriends,
			Threshold:        k,
			Total:            n,
			Version:          config.Version,
			GitHubReleaseURL: config.GitHubURL,
			ManifestChecksum: manifestChecksum,
			RecoverChecksum:  recoverChecksum,
			Created:          now,
			Anonymous:        config.Anonymous,
		}
		pdfContent, err := pdf.GenerateReadme(pdfData)
		if err != nil {
			return nil, fmt.Errorf("generating PDF for %s: %w", friend.Name, err)
		}

		// Create ZIP bundle
		zipFiles := []bundle.ZipFile{
			{Name: "README.txt", Content: []byte(readmeContent), ModTime: now},
			{Name: "README.pdf", Content: pdfContent, ModTime: now},
			{Name: "MANIFEST.age", Content: manifestData, ModTime: now},
			{Name: "recover.html", Content: []byte(recoverHTML), ModTime: now},
		}

		zipData, err := createZipInMemory(zipFiles)
		if err != nil {
			return nil, fmt.Errorf("creating ZIP for %s: %w", friend.Name, err)
		}

		bundles[i] = BundleOutput{
			FriendName: friend.Name,
			FileName:   fmt.Sprintf("bundle-%s.zip", sanitizeName(friend.Name)),
			Data:       zipData,
		}
	}

	return bundles, nil
}

// createTarGz creates a tar.gz archive from file entries.
func createTarGz(files []FileEntry) ([]byte, error) {
	var buf bytes.Buffer
	gzw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gzw)

	// Use "manifest" as the root directory name
	rootDir := "manifest"

	// Add root directory entry
	if err := tw.WriteHeader(&tar.Header{
		Name:     rootDir + "/",
		Mode:     0755,
		Typeflag: tar.TypeDir,
		ModTime:  time.Now().UTC(),
	}); err != nil {
		return nil, fmt.Errorf("writing directory header: %w", err)
	}

	for _, f := range files {
		// Normalize the file path - ensure it's under manifest/
		name := f.Name
		// Remove leading slashes or "manifest/" prefix if present
		name = trimLeadingSlashes(name)
		if len(name) > 9 && name[:9] == "manifest/" {
			name = name[9:]
		}
		// Add the manifest/ prefix
		fullPath := rootDir + "/" + name

		header := &tar.Header{
			Name:     fullPath,
			Mode:     0644,
			Size:     int64(len(f.Data)),
			ModTime:  time.Now().UTC(),
			Typeflag: tar.TypeReg,
		}

		if err := tw.WriteHeader(header); err != nil {
			return nil, fmt.Errorf("writing header for %s: %w", f.Name, err)
		}

		if _, err := tw.Write(f.Data); err != nil {
			return nil, fmt.Errorf("writing data for %s: %w", f.Name, err)
		}
	}

	if err := tw.Close(); err != nil {
		return nil, fmt.Errorf("closing tar: %w", err)
	}
	if err := gzw.Close(); err != nil {
		return nil, fmt.Errorf("closing gzip: %w", err)
	}

	return buf.Bytes(), nil
}

// createZipInMemory creates a ZIP archive in memory.
func createZipInMemory(files []bundle.ZipFile) ([]byte, error) {
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)

	for _, file := range files {
		header := &zip.FileHeader{
			Name:   file.Name,
			Method: zip.Deflate,
		}
		header.Modified = file.ModTime

		fw, err := w.CreateHeader(header)
		if err != nil {
			return nil, fmt.Errorf("creating entry %s: %w", file.Name, err)
		}

		if _, err := fw.Write(file.Content); err != nil {
			return nil, fmt.Errorf("writing entry %s: %w", file.Name, err)
		}
	}

	if err := w.Close(); err != nil {
		return nil, fmt.Errorf("closing zip: %w", err)
	}

	return buf.Bytes(), nil
}

// trimLeadingSlashes removes leading slashes from a path.
func trimLeadingSlashes(s string) string {
	for len(s) > 0 && (s[0] == '/' || s[0] == '\\') {
		s = s[1:]
	}
	return s
}

// sanitizeName converts a name to a filesystem-safe lowercase ASCII string.
// It transliterates accented/diacritic characters to their ASCII base form
// (e.g. "José" → "jose", "Ñoño" → "nono") so that filenames are always valid.
func sanitizeName(name string) string {
	// Decompose to NFD: splits characters like "é" into "e" + combining accent.
	// Then drop combining marks to keep only the base letter.
	var stripped []rune
	for _, r := range norm.NFD.String(name) {
		if !unicode.Is(unicode.Mn, r) { // Mn = Mark, Nonspacing (combining diacritics)
			stripped = append(stripped, r)
		}
	}

	result := ""
	for _, r := range stripped {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			result += string(r)
		} else if r == ' ' || r == '-' || r == '_' {
			result += "-"
		}
	}
	// Convert to lowercase
	lower := ""
	for _, r := range result {
		if r >= 'A' && r <= 'Z' {
			lower += string(r + 32)
		} else {
			lower += string(r)
		}
	}
	return lower
}

// parseProjectYAMLJS parses a project.yml file to extract friend information.
// Args: yamlText (string)
// Returns: { project: {...}, error: string|null }
func parseProjectYAMLJS(this js.Value, args []js.Value) any {
	if len(args) < 1 {
		return errorResult("missing yamlText argument")
	}

	yamlText := args[0].String()

	proj, err := parseProjectYAML(yamlText)
	if err != nil {
		return errorResult(err.Error())
	}

	// Convert friends to JS array
	friends := make([]any, len(proj.Friends))
	for i, f := range proj.Friends {
		friends[i] = map[string]any{
			"name":  f.Name,
			"email": f.Email,
			"phone": f.Phone,
		}
	}

	return js.ValueOf(map[string]any{
		"project": map[string]any{
			"name":      proj.Name,
			"threshold": proj.Threshold,
			"friends":   friends,
		},
		"error": nil,
	})
}

// ProjectYAML is a minimal struct for parsing project.yml
type ProjectYAML struct {
	Name      string `yaml:"name"`
	Threshold int    `yaml:"threshold"`
	Friends   []struct {
		Name  string `yaml:"name"`
		Email string `yaml:"email"`
		Phone string `yaml:"phone,omitempty"`
	} `yaml:"friends"`
}

// parseProjectYAML parses project.yml content.
func parseProjectYAML(yamlText string) (*ProjectYAML, error) {
	var proj ProjectYAML
	if err := yaml.Unmarshal([]byte(yamlText), &proj); err != nil {
		return nil, fmt.Errorf("parsing YAML: %w", err)
	}
	return &proj, nil
}

// generatePassphraseJS generates a random passphrase.
// Args: numBytes (int, optional - defaults to 32)
// Returns: { passphrase: string, error: string|null }
func generatePassphraseJS(this js.Value, args []js.Value) any {
	numBytes := crypto.DefaultPassphraseBytes
	if len(args) > 0 && !args[0].IsUndefined() && !args[0].IsNull() {
		numBytes = args[0].Int()
	}

	passphrase, err := crypto.GeneratePassphrase(numBytes)
	if err != nil {
		return errorResult(err.Error())
	}

	return js.ValueOf(map[string]any{
		"passphrase": passphrase,
		"error":      nil,
	})
}

// hashBytesJS computes SHA-256 hash of bytes.
// Args: data (Uint8Array)
// Returns: string (sha256:...)
func hashBytesJS(this js.Value, args []js.Value) any {
	if len(args) < 1 {
		return ""
	}

	jsData := args[0]
	dataLen := jsData.Get("length").Int()
	data := make([]byte, dataLen)
	js.CopyBytesToGo(data, jsData)

	return core.HashBytes(data)
}

// encryptAgeJS encrypts data using age/scrypt.
// Args: data (Uint8Array), passphrase (string)
// Returns: { encrypted: Uint8Array, error: string|null }
func encryptAgeJS(this js.Value, args []js.Value) any {
	if len(args) < 2 {
		return errorResult("missing arguments (need data, passphrase)")
	}

	jsData := args[0]
	dataLen := jsData.Get("length").Int()
	data := make([]byte, dataLen)
	js.CopyBytesToGo(data, jsData)

	passphrase := args[1].String()

	var buf bytes.Buffer
	if err := core.Encrypt(&buf, bytes.NewReader(data), passphrase); err != nil {
		return errorResult(err.Error())
	}

	encrypted := buf.Bytes()
	jsResult := js.Global().Get("Uint8Array").New(len(encrypted))
	js.CopyBytesToJS(jsResult, encrypted)

	return js.ValueOf(map[string]any{
		"encrypted": jsResult,
		"error":     nil,
	})
}

// splitPassphraseJS splits a passphrase using Shamir's Secret Sharing.
// Args: passphrase (string), n (int), k (int)
// Returns: { shares: [{index, dataB64}], error: string|null }
func splitPassphraseJS(this js.Value, args []js.Value) any {
	if len(args) < 3 {
		return errorResult("missing arguments (need passphrase, n, k)")
	}

	passphrase := args[0].String()
	n := args[1].Int()
	k := args[2].Int()

	shares, err := core.Split([]byte(passphrase), n, k)
	if err != nil {
		return errorResult(err.Error())
	}

	jsShares := make([]any, len(shares))
	for i, shareData := range shares {
		jsShares[i] = map[string]any{
			"index":   i + 1,
			"dataB64": base64.StdEncoding.EncodeToString(shareData),
		}
	}

	return js.ValueOf(map[string]any{
		"shares": jsShares,
		"error":  nil,
	})
}

// createShareJS creates an encoded share.
// Args: index, total, threshold (int), holder (string), dataB64 (string), created (string RFC3339)
// Returns: { encoded: string, error: string|null }
func createShareJS(this js.Value, args []js.Value) any {
	if len(args) < 6 {
		return errorResult("missing arguments")
	}

	index := args[0].Int()
	total := args[1].Int()
	threshold := args[2].Int()
	holder := args[3].String()
	dataB64 := args[4].String()
	createdStr := args[5].String()

	data, err := base64.StdEncoding.DecodeString(dataB64)
	if err != nil {
		return errorResult(fmt.Sprintf("invalid base64 data: %v", err))
	}

	created, err := time.Parse(time.RFC3339, createdStr)
	if err != nil {
		return errorResult(fmt.Sprintf("invalid created time: %v", err))
	}

	share := &core.Share{
		Version:   1,
		Index:     index,
		Total:     total,
		Threshold: threshold,
		Holder:    holder,
		Created:   created,
		Data:      data,
		Checksum:  core.HashBytes(data),
	}

	return js.ValueOf(map[string]any{
		"encoded": share.Encode(),
		"error":   nil,
	})
}

// createTarGzJS creates a tar.gz archive from file entries.
// Args: files (array of {name: string, data: Uint8Array})
// Returns: { data: Uint8Array, error: string|null }
func createTarGzJS(this js.Value, args []js.Value) any {
	if len(args) < 1 {
		return errorResult("missing files argument")
	}

	filesJS := args[0]
	filesLen := filesJS.Length()
	files := make([]FileEntry, filesLen)

	for i := 0; i < filesLen; i++ {
		f := filesJS.Index(i)
		name := f.Get("name").String()
		dataJS := f.Get("data")
		dataLen := dataJS.Get("length").Int()
		data := make([]byte, dataLen)
		js.CopyBytesToGo(data, dataJS)
		files[i] = FileEntry{
			Name: name,
			Data: data,
		}
	}

	archiveData, err := createTarGz(files)
	if err != nil {
		return errorResult(err.Error())
	}

	jsResult := js.Global().Get("Uint8Array").New(len(archiveData))
	js.CopyBytesToJS(jsResult, archiveData)

	return js.ValueOf(map[string]any{
		"data":  jsResult,
		"error": nil,
	})
}

// Functions are registered in main.go
func init() {
	// This file is compiled only for WASM, functions will be registered in main()
}
