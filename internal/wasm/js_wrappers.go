//go:build js && wasm

package main

import (
	"syscall/js"
)

// parseShareJS parses a share from text content.
// Args: content (string)
// Returns: { share: {...}, error: string|null }
func parseShareJS(this js.Value, args []js.Value) any {
	if len(args) < 1 {
		return errorResult("missing content argument")
	}

	content := args[0].String()
	share, err := parseShare(content)
	if err != nil {
		return errorResult(err.Error())
	}

	return js.ValueOf(map[string]any{
		"share": shareInfoToJS(share),
		"error": nil,
	})
}

// combineSharesJS combines multiple shares to recover the passphrase.
// Args: sharesJSON (array of share objects with dataB64)
// Returns: { passphrase: string, error: string|null }
func combineSharesJS(this js.Value, args []js.Value) any {
	if len(args) < 1 {
		return errorResult("missing shares argument")
	}

	sharesArray := args[0]
	length := sharesArray.Length()

	shares := make([]ShareData, length)
	for i := 0; i < length; i++ {
		shareObj := sharesArray.Index(i)
		shares[i] = ShareData{
			Version:   shareObj.Get("version").Int(),
			Index:     shareObj.Get("index").Int(),
			Threshold: shareObj.Get("threshold").Int(),
			DataB64:   shareObj.Get("dataB64").String(),
		}
	}

	passphrase, err := combineShares(shares)
	if err != nil {
		return errorResult(err.Error())
	}

	return js.ValueOf(map[string]any{
		"passphrase": passphrase,
		"error":      nil,
	})
}

// decryptManifestJS decrypts an age-encrypted manifest.
// Args: encryptedData (Uint8Array), passphrase (string)
// Returns: { data: Uint8Array, error: string|null }
func decryptManifestJS(this js.Value, args []js.Value) any {
	if len(args) < 2 {
		return errorResult("missing arguments (need encryptedData, passphrase)")
	}

	// Read Uint8Array from JS
	jsData := args[0]
	dataLen := jsData.Get("length").Int()
	encryptedData := make([]byte, dataLen)
	js.CopyBytesToGo(encryptedData, jsData)

	passphrase := args[1].String()

	decrypted, err := decryptManifest(encryptedData, passphrase)
	if err != nil {
		return errorResult(err.Error())
	}

	// Create Uint8Array to return
	jsResult := js.Global().Get("Uint8Array").New(len(decrypted))
	js.CopyBytesToJS(jsResult, decrypted)

	return js.ValueOf(map[string]any{
		"data":  jsResult,
		"error": nil,
	})
}

// extractTarGzJS extracts files from tar.gz data.
// Args: tarGzData (Uint8Array)
// Returns: { files: [{name: string, data: Uint8Array}], error: string|null }
func extractTarGzJS(this js.Value, args []js.Value) any {
	if len(args) < 1 {
		return errorResult("missing tarGzData argument")
	}

	// Read Uint8Array from JS
	jsData := args[0]
	dataLen := jsData.Get("length").Int()
	tarGzData := make([]byte, dataLen)
	js.CopyBytesToGo(tarGzData, jsData)

	files, err := extractTarGz(tarGzData)
	if err != nil {
		return errorResult(err.Error())
	}

	// Convert files to JS array
	jsFiles := make([]any, len(files))
	for i, f := range files {
		jsFileData := js.Global().Get("Uint8Array").New(len(f.Data))
		js.CopyBytesToJS(jsFileData, f.Data)
		jsFiles[i] = map[string]any{
			"name": f.Name,
			"data": jsFileData,
		}
	}

	return js.ValueOf(map[string]any{
		"files": jsFiles,
		"error": nil,
	})
}

// extractBundleJS extracts share and manifest from a bundle ZIP.
// Args: zipData (Uint8Array)
// Returns: { share: {...}, manifest: Uint8Array|null, error: string|null }
func extractBundleJS(this js.Value, args []js.Value) any {
	if len(args) < 1 {
		return errorResult("missing zipData argument")
	}

	// Read Uint8Array from JS
	jsData := args[0]
	dataLen := jsData.Get("length").Int()
	zipData := make([]byte, dataLen)
	js.CopyBytesToGo(zipData, jsData)

	bundle, err := extractBundle(zipData)
	if err != nil {
		return errorResult(err.Error())
	}

	result := map[string]any{
		"share": shareInfoToJS(bundle.Share),
		"error": nil,
	}

	// Include manifest if present
	if len(bundle.Manifest) > 0 {
		jsManifest := js.Global().Get("Uint8Array").New(len(bundle.Manifest))
		js.CopyBytesToJS(jsManifest, bundle.Manifest)
		result["manifest"] = jsManifest
	} else {
		result["manifest"] = nil
	}

	return js.ValueOf(result)
}

// parseCompactShareJS parses a compact-encoded share string (e.g. RM1:2:5:3:BASE64:CHECK).
// Args: compact (string)
// Returns: { share: {...}, error: string|null }
func parseCompactShareJS(this js.Value, args []js.Value) any {
	if len(args) < 1 {
		return errorResult("missing compact share argument")
	}

	compact := args[0].String()
	share, err := parseCompactShare(compact)
	if err != nil {
		return errorResult(err.Error())
	}

	return js.ValueOf(map[string]any{
		"share": shareInfoToJS(share),
		"error": nil,
	})
}

// decodeWordsJS decodes 25 BIP39 words to raw share data bytes and share index.
// The first 24 words encode the data; the 25th word packs 4 bits of index + 7 bits of checksum.
// Returns index=0 if the share index was > 15 (sentinel for "unknown — UI should not highlight a specific contact").
// Returns an error if the embedded checksum doesn't match (wrong word order, typos, etc.).
// Args: words (string array)
// Returns: { data: Uint8Array, index: number, checksum: string, error: string|null }
func decodeWordsJS(this js.Value, args []js.Value) any {
	if len(args) < 1 {
		return errorResult("missing words argument")
	}

	wordsArray := args[0]
	length := wordsArray.Length()
	words := make([]string, length)
	for i := 0; i < length; i++ {
		words[i] = wordsArray.Index(i).String()
	}

	data, index, checksum, lang, err := decodeShareWords(words)
	if err != nil {
		return errorResult(err.Error())
	}

	jsData := js.Global().Get("Uint8Array").New(len(data))
	js.CopyBytesToJS(jsData, data)

	return js.ValueOf(map[string]any{
		"data":     jsData,
		"index":    index,
		"checksum": checksum,
		"lang":     lang,
		"error":    nil,
	})
}

// shareInfoToJS converts a ShareInfo to a JS-compatible map.
func shareInfoToJS(s *ShareInfo) map[string]any {
	return map[string]any{
		"version":   s.Version,
		"index":     s.Index,
		"total":     s.Total,
		"threshold": s.Threshold,
		"holder":    s.Holder,
		"created":   s.Created,
		"checksum":  s.Checksum,
		"dataB64":   s.DataB64,
		"compact":   s.Compact,
	}
}

func errorResult(msg string) any {
	return js.ValueOf(map[string]any{
		"error": msg,
	})
}
