//go:build js && wasm && create

package main

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"fmt"
	"syscall/js"
	"time"

	"github.com/eljojo/rememory/internal/bundle"
	"github.com/eljojo/rememory/internal/core"
	"github.com/eljojo/rememory/internal/crypto"
	"github.com/eljojo/rememory/internal/html"
	"github.com/eljojo/rememory/internal/pdf"
	"github.com/eljojo/rememory/internal/project"
	"github.com/eljojo/rememory/internal/translations"

	"gopkg.in/yaml.v3"
)

// FileEntry represents a file passed from JavaScript.
type FileEntry struct {
	Name string // Relative path (e.g., "manifest/secrets.txt")
	Data []byte
}

// FriendInput represents friend data from JavaScript.
type FriendInput struct {
	Name     string
	Contact  string
	Language string
}

// CreateBundlesConfig holds all parameters for bundle creation.
type CreateBundlesConfig struct {
	ProjectName     string
	Threshold       int
	Friends         []FriendInput
	Files           []FileEntry
	Version         string
	GitHubURL       string
	Anonymous       bool
	DefaultLanguage string // Default bundle language for all friends
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
	if defLang := configJS.Get("defaultLanguage"); !defLang.IsUndefined() && !defLang.IsNull() {
		config.DefaultLanguage = defLang.String()
	}

	// Parse friends array
	friendsJS := configJS.Get("friends")
	friendsLen := friendsJS.Length()
	config.Friends = make([]FriendInput, friendsLen)
	for i := 0; i < friendsLen; i++ {
		f := friendsJS.Index(i)
		config.Friends[i] = FriendInput{
			Name: f.Get("name").String(),
		}
		if contact := f.Get("contact"); !contact.IsUndefined() && !contact.IsNull() {
			config.Friends[i].Contact = contact.String()
		}
		if lang := f.Get("language"); !lang.IsUndefined() && !lang.IsNull() {
			config.Friends[i].Language = lang.String()
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

	// Validate friends
	for i, f := range config.Friends {
		if f.Name == "" {
			return nil, fmt.Errorf("friend %d: name is required", i+1)
		}
	}

	// Create tar.gz archive of files
	archiveData, err := createTarGz(config.Files)
	if err != nil {
		return nil, fmt.Errorf("creating archive: %w", err)
	}

	// Generate random passphrase (v2: split raw bytes, not the base64 string)
	raw, passphrase, err := crypto.GenerateRawPassphrase(crypto.DefaultPassphraseBytes)
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
	rawShares, err := core.Split(raw, n, k)
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
			Version:   2,
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
			Name:     f.Name,
			Contact:  f.Contact,
			Language: f.Language,
		}
	}

	// Generate bundle for each friend
	for i, friend := range config.Friends {
		share := shares[i]

		// Resolve language: friend override > project default > "en"
		lang := friend.Language
		if lang == "" {
			lang = config.DefaultLanguage
		}
		if lang == "" {
			lang = "en"
		}

		// Get other friends (excluding this one) - empty for anonymous mode
		var otherFriends []project.Friend
		var otherFriendsInfo []html.FriendInfo
		if !config.Anonymous {
			otherFriends = make([]project.Friend, 0, n-1)
			otherFriendsInfo = make([]html.FriendInfo, 0, n-1)
			for j, f := range projectFriends {
				if j != i {
					otherFriends = append(otherFriends, f)
					otherFriendsInfo = append(otherFriendsInfo, html.FriendInfo{
						Name:       f.Name,
						Contact:    f.Contact,
						ShareIndex: j + 1, // 1-based share index
					})
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
			Language:     lang,
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
			Language:         lang,
		}
		readmeContent := bundle.GenerateReadme(readmeData)

		// Generate README.pdf
		// Web-created bundles always use the GitHub Pages recovery URL
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
			Language:         lang,
		}
		pdfContent, err := pdf.GenerateReadme(pdfData)
		if err != nil {
			return nil, fmt.Errorf("generating PDF for %s: %w", friend.Name, err)
		}

		// Create ZIP bundle
		readmeFileTxt := translations.ReadmeFilename(lang, ".txt")
		readmeFilePdf := translations.ReadmeFilename(lang, ".pdf")
		zipFiles := []bundle.ZipFile{
			{Name: readmeFileTxt, Content: []byte(readmeContent), ModTime: now},
			{Name: readmeFilePdf, Content: pdfContent, ModTime: now},
			{Name: "MANIFEST.age", Content: manifestData, ModTime: now},
			{Name: "recover.html", Content: []byte(recoverHTML), ModTime: now},
		}

		zipData, err := createZipInMemory(zipFiles)
		if err != nil {
			return nil, fmt.Errorf("creating ZIP for %s: %w", friend.Name, err)
		}

		bundles[i] = BundleOutput{
			FriendName: friend.Name,
			FileName:   fmt.Sprintf("bundle-%s.zip", core.SanitizeFilename(friend.Name)),
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
			"name":     f.Name,
			"contact":  f.Contact,
			"language": f.Language,
		}
	}

	return js.ValueOf(map[string]any{
		"project": map[string]any{
			"name":      proj.Name,
			"threshold": proj.Threshold,
			"language":  proj.Language,
			"friends":   friends,
		},
		"error": nil,
	})
}

// ProjectYAML is a minimal struct for parsing project.yml
type ProjectYAML struct {
	Name      string `yaml:"name"`
	Threshold int    `yaml:"threshold"`
	Language  string `yaml:"language,omitempty"`
	Friends   []struct {
		Name     string `yaml:"name"`
		Contact  string `yaml:"contact,omitempty"`
		Language string `yaml:"language,omitempty"`
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

// Functions are registered in main.go
func init() {
	// This file is compiled only for WASM, functions will be registered in main()
}
