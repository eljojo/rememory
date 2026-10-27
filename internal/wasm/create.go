//go:build js && wasm && create

package main

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"fmt"
	"strings"
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

// BundleOutput represents a generated bundle for JavaScript.
type BundleOutput struct {
	FriendName string
	FileName   string
	Data       []byte
}

// CreateBundlesFromArchiveConfig holds parameters for creating bundles from
// pre-built archive data (already tlock-encrypted by JS, if applicable).
type CreateBundlesFromArchiveConfig struct {
	ProjectName     string
	Threshold       int
	Friends         []FriendInput
	ArchiveData     []byte
	Version         string
	Anonymous       bool
	DefaultLanguage string
	TlockRound      uint64
	TlockUnlock     string // RFC 3339 timestamp
}

// bundleGenConfig holds shared parameters for bundle generation from an
// encrypted manifest. Used by both createBundles and createBundlesFromArchive.
type bundleGenConfig struct {
	ProjectName     string
	Threshold       int
	Friends         []FriendInput
	Version         string
	Anonymous       bool
	DefaultLanguage string
	TlockEnabled    bool
}

// createBundlesFromArchive creates bundles from pre-built archive data.
// The archiveData may already be tlock-encrypted by JS. This function
// age-encrypts it (outer layer), optionally prepends the metadata envelope,
// splits the passphrase, and packages bundles.
func createBundlesFromArchive(config CreateBundlesFromArchiveConfig) ([]BundleOutput, []byte, error) {
	if config.ProjectName == "" {
		return nil, nil, fmt.Errorf("project name is required")
	}
	if len(config.Friends) < 2 {
		return nil, nil, fmt.Errorf("need at least 2 friends, got %d", len(config.Friends))
	}
	if config.Threshold < 2 {
		return nil, nil, fmt.Errorf("threshold must be at least 2, got %d", config.Threshold)
	}
	if config.Threshold > len(config.Friends) {
		return nil, nil, fmt.Errorf("threshold (%d) cannot exceed number of friends (%d)", config.Threshold, len(config.Friends))
	}
	if len(config.ArchiveData) == 0 {
		return nil, nil, fmt.Errorf("no archive data provided")
	}
	names := make([]string, len(config.Friends))
	for i, f := range config.Friends {
		if f.Name == "" {
			return nil, nil, fmt.Errorf("friend %d: name is required", i+1)
		}
		names[i] = f.Name
	}
	if _, err := core.FriendFileStems(names); err != nil {
		return nil, nil, err
	}

	// Generate random passphrase
	raw, passphrase, err := crypto.GenerateRawPassphrase(crypto.DefaultPassphraseBytes)
	if err != nil {
		return nil, nil, fmt.Errorf("generating passphrase: %w", err)
	}

	// If tlock metadata provided, wrap tlock-encrypted archive in a container ZIP
	dataToEncrypt := config.ArchiveData
	tlockEnabled := config.TlockRound > 0
	if tlockEnabled {
		meta := &core.TlockMeta{
			V:      core.TlockContainerVersion,
			Method: core.TlockMethodQuicknet,
			Round:  config.TlockRound,
			Unlock: config.TlockUnlock,
			Chain:  core.QuicknetChainHash,
		}
		container, err := core.BuildTlockContainer(meta, config.ArchiveData)
		if err != nil {
			return nil, nil, fmt.Errorf("building tlock container: %w", err)
		}
		dataToEncrypt = container
	}

	// Encrypt with age (outer layer) — always a plain age file
	var encryptedBuf bytes.Buffer
	if err := core.Encrypt(&encryptedBuf, bytes.NewReader(dataToEncrypt), passphrase); err != nil {
		return nil, nil, fmt.Errorf("encrypting archive: %w", err)
	}

	manifestData := encryptedBuf.Bytes()

	bundles, err := bundleFromManifest(manifestData, raw, bundleGenConfig{
		ProjectName:     config.ProjectName,
		Threshold:       config.Threshold,
		Friends:         config.Friends,
		Version:         config.Version,
		Anonymous:       config.Anonymous,
		DefaultLanguage: config.DefaultLanguage,
		TlockEnabled:    tlockEnabled,
	})
	if err != nil {
		return nil, nil, err
	}

	return bundles, manifestData, nil
}

// bundleFromManifest generates bundles for all friends given the final
// encrypted manifest data and the raw passphrase bytes for Shamir splitting.
func bundleFromManifest(manifestData, raw []byte, config bundleGenConfig) ([]BundleOutput, error) {
	html.SetVersion(config.Version)

	// Derive GitHub release URL from version for README/PDF
	githubReleaseURL := fmt.Sprintf("%s/releases/tag/%s", core.GitHubRepo, config.Version)

	manifestChecksum := core.HashBytes(manifestData)

	n := len(config.Friends)
	k := config.Threshold
	rawShares, err := core.Split(raw, n, k)
	if err != nil {
		return nil, fmt.Errorf("splitting passphrase: %w", err)
	}

	now := time.Now().UTC()

	bundles := make([]BundleOutput, n)
	shares := make([]*core.Share, n)

	for i, friend := range config.Friends {
		shares[i] = &core.Share{
			Version:   2,
			Index:     i + 1,
			Total:     n,
			Threshold: k,
			Holder:    friend.Name,
			Created:   now,
			Data:      rawShares[i],
			Checksum:  core.HashBytes(rawShares[i]),
		}
	}

	projectFriends := make([]project.Friend, len(config.Friends))
	for i, f := range config.Friends {
		projectFriends[i] = project.Friend{
			Name:     f.Name,
			Contact:  f.Contact,
			Language: f.Language,
		}
	}

	for i, friend := range config.Friends {
		share := shares[i]

		lang := friend.Language
		if lang == "" {
			lang = config.DefaultLanguage
		}
		if lang == "" {
			lang = "en"
		}

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
						ShareIndex: j + 1,
					})
				}
			}
		}

		personalization := &html.PersonalizationData{
			Holder:       friend.Name,
			HolderShare:  share.Encode(),
			OtherFriends: otherFriendsInfo,
			Threshold:    k,
			Total:        n,
			Language:     lang,
			TlockEnabled: config.TlockEnabled,
		}

		manifestEmbedded := len(manifestData) <= html.MaxEmbeddedManifestSize
		if manifestEmbedded {
			personalization.ManifestB64 = base64.StdEncoding.EncodeToString(manifestData)
		}

		recoverHTML := html.GenerateRecoverHTML(personalization)
		recoverChecksum := core.HashString(recoverHTML)

		readmeData := bundle.ReadmeData{
			ProjectName:      config.ProjectName,
			Holder:           friend.Name,
			Share:            share,
			OtherFriends:     otherFriends,
			Threshold:        k,
			Total:            n,
			Version:          config.Version,
			GitHubReleaseURL: githubReleaseURL,
			ManifestChecksum: manifestChecksum,
			RecoverChecksum:  recoverChecksum,
			Created:          now,
			Anonymous:        config.Anonymous,
			Language:         lang,
			ManifestEmbedded: manifestEmbedded,
		}
		readmeContent := bundle.GenerateReadme(readmeData)

		pdfData := pdf.ReadmeData{
			ProjectName:      config.ProjectName,
			Holder:           friend.Name,
			Share:            share,
			OtherFriends:     otherFriends,
			Threshold:        k,
			Total:            n,
			Version:          config.Version,
			GitHubReleaseURL: githubReleaseURL,
			ManifestChecksum: manifestChecksum,
			RecoverChecksum:  recoverChecksum,
			Created:          now,
			Anonymous:        config.Anonymous,
			Language:         lang,
			ManifestEmbedded: manifestEmbedded,
		}
		pdfContent, err := pdf.GenerateReadme(pdfData)
		if err != nil {
			return nil, fmt.Errorf("generating PDF for %s: %w", friend.Name, err)
		}

		readmeFileTxt := translations.ReadmeFilename(lang, ".txt")
		readmeFilePdf := translations.ReadmeFilename(lang, ".pdf")
		zipFiles := []bundle.ZipFile{
			{Name: readmeFileTxt, Content: []byte(readmeContent), ModTime: now},
			{Name: readmeFilePdf, Content: pdfContent, ModTime: now},
			{Name: "recover.html", Content: []byte(recoverHTML), ModTime: now},
		}
		if !manifestEmbedded {
			zipFiles = append(zipFiles, bundle.ZipFile{Name: "MANIFEST.age", Content: manifestData, ModTime: now})
		}

		zipData, err := createZipInMemory(zipFiles)
		if err != nil {
			return nil, fmt.Errorf("creating ZIP for %s: %w", friend.Name, err)
		}

		bundles[i] = BundleOutput{
			FriendName: friend.Name,
			FileName:   fmt.Sprintf("bundle-%s.zip", core.FriendFileStem(friend.Name, i+1)),
			Data:       zipData,
		}
	}

	return bundles, nil
}

// createZip creates a ZIP archive from file entries.
func createZip(files []FileEntry) ([]byte, error) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	// Use "manifest" as the root directory name
	rootDir := "manifest"

	for _, f := range files {
		// Normalize the file path - ensure it's under manifest/
		name := f.Name
		// Remove leading slashes or "manifest/" prefix if present
		name = trimLeadingSlashes(name)
		// Security: reject path traversal attempts
		if strings.Contains(name, "..") {
			return nil, fmt.Errorf("invalid path in file entry: %s", f.Name)
		}
		if len(name) > 9 && name[:9] == "manifest/" {
			name = name[9:]
		}
		// Add the manifest/ prefix
		fullPath := rootDir + "/" + name

		header := &zip.FileHeader{
			Name:     fullPath,
			Method:   zip.Deflate,
			Modified: time.Now().UTC(),
		}

		fw, err := zw.CreateHeader(header)
		if err != nil {
			return nil, fmt.Errorf("writing header for %s: %w", f.Name, err)
		}

		if _, err := fw.Write(f.Data); err != nil {
			return nil, fmt.Errorf("writing data for %s: %w", f.Name, err)
		}
	}

	if err := zw.Close(); err != nil {
		return nil, fmt.Errorf("closing zip: %w", err)
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
	// A threshold the friends can't meet would otherwise be lowered silently.
	if proj.Threshold != 0 {
		if err := core.ValidateShamirParams(len(proj.Friends), proj.Threshold); err != nil {
			return nil, err
		}
	}
	for _, f := range proj.Friends {
		if f.Language != "" && !translations.IsSupported(f.Language) {
			return nil, fmt.Errorf("friend %q: unsupported language %q (supported: %s)", f.Name, f.Language, strings.Join(translations.Languages, ", "))
		}
	}
	return &proj, nil
}

// createArchiveJS creates a ZIP archive from files, returning the raw bytes.
// This allows JS to tlock-encrypt the archive before passing it to
// createBundlesFromArchiveJS for age encryption and bundle packaging.
// Args: files array [{name: string, data: Uint8Array}, ...]
// Returns: { data: Uint8Array, error: string|null }
func createArchiveJS(this js.Value, args []js.Value) any {
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
		files[i] = FileEntry{Name: name, Data: data}
	}

	archiveData, err := createZip(files)
	if err != nil {
		return errorResult(err.Error())
	}

	jsData := js.Global().Get("Uint8Array").New(len(archiveData))
	js.CopyBytesToJS(jsData, archiveData)

	return js.ValueOf(map[string]any{
		"data":  jsData,
		"error": nil,
	})
}

// createBundlesFromArchiveJS is the WASM entry point for creating bundles from
// pre-built archive data. The archive should already be tlock-encrypted by JS
// if time-lock is enabled. This function age-encrypts it (outer layer),
// prepends the metadata envelope when tlock metadata is provided, splits the
// passphrase via Shamir, and packages personalized bundles.
// Args: config object with archiveData, projectName, threshold, friends, etc.
// Returns: { bundles: [...], manifest: Uint8Array, error: string|null }
func createBundlesFromArchiveJS(this js.Value, args []js.Value) any {
	if len(args) < 1 {
		return errorResult("missing config argument")
	}

	configJS := args[0]

	config := CreateBundlesFromArchiveConfig{
		ProjectName: configJS.Get("projectName").String(),
		Threshold:   configJS.Get("threshold").Int(),
		Version:     configJS.Get("version").String(),
		Anonymous:   configJS.Get("anonymous").Bool(),
	}
	if defLang := configJS.Get("defaultLanguage"); !defLang.IsUndefined() && !defLang.IsNull() {
		config.DefaultLanguage = defLang.String()
	}

	// Parse archive data
	archiveJS := configJS.Get("archiveData")
	archiveLen := archiveJS.Get("length").Int()
	config.ArchiveData = make([]byte, archiveLen)
	js.CopyBytesToGo(config.ArchiveData, archiveJS)

	// Parse tlock metadata (optional)
	if tlockRound := configJS.Get("tlockRound"); !tlockRound.IsUndefined() && !tlockRound.IsNull() {
		config.TlockRound = uint64(tlockRound.Float())
	}
	if tlockUnlock := configJS.Get("tlockUnlock"); !tlockUnlock.IsUndefined() && !tlockUnlock.IsNull() {
		config.TlockUnlock = tlockUnlock.String()
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

	bundles, manifestData, err := createBundlesFromArchive(config)
	if err != nil {
		return errorResult(err.Error())
	}

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

	jsManifest := js.Global().Get("Uint8Array").New(len(manifestData))
	js.CopyBytesToJS(jsManifest, manifestData)

	return js.ValueOf(map[string]any{
		"bundles":  jsBundles,
		"manifest": jsManifest,
		"error":    nil,
	})
}

// Functions are registered in main.go
func init() {
	// This file is compiled only for WASM, functions will be registered in main()
}
