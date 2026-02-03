//go:build js && wasm

package main

import (
	"syscall/js"
)

func main() {
	// Register functions on the global object
	js.Global().Set("rememoryParseShare", js.FuncOf(parseShareJS))
	js.Global().Set("rememoryCombineShares", js.FuncOf(combineSharesJS))
	js.Global().Set("rememoryDecryptManifest", js.FuncOf(decryptManifestJS))
	js.Global().Set("rememoryExtractTarGz", js.FuncOf(extractTarGzJS))
	js.Global().Set("rememoryExtractBundle", js.FuncOf(extractBundleJS))

	// Signal that WASM is ready
	js.Global().Set("rememoryReady", true)

	// Keep the Go program running
	select {}
}

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
		"share": map[string]any{
			"version":   share.Version,
			"index":     share.Index,
			"total":     share.Total,
			"threshold": share.Threshold,
			"holder":    share.Holder,
			"created":   share.Created,
			"checksum":  share.Checksum,
			"dataB64":   share.DataB64,
		},
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
			Index:   shareObj.Get("index").Int(),
			DataB64: shareObj.Get("dataB64").String(),
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
		"share": map[string]any{
			"version":   bundle.Share.Version,
			"index":     bundle.Share.Index,
			"total":     bundle.Share.Total,
			"threshold": bundle.Share.Threshold,
			"holder":    bundle.Share.Holder,
			"created":   bundle.Share.Created,
			"checksum":  bundle.Share.Checksum,
			"dataB64":   bundle.Share.DataB64,
		},
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

func errorResult(msg string) any {
	return js.ValueOf(map[string]any{
		"error": msg,
	})
}
