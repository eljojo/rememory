//go:build js && wasm && !create

package main

import (
	"syscall/js"
)

func main() {
	// Register recovery functions on the global object
	js.Global().Set("rememoryParseShare", js.FuncOf(parseShareJS))
	js.Global().Set("rememoryCombineShares", js.FuncOf(combineSharesJS))
	js.Global().Set("rememoryDecryptManifest", js.FuncOf(decryptManifestJS))
	js.Global().Set("rememoryExtractTarGz", js.FuncOf(extractTarGzJS))
	js.Global().Set("rememoryExtractBundle", js.FuncOf(extractBundleJS))
	js.Global().Set("rememoryParseCompactShare", js.FuncOf(parseCompactShareJS))
	js.Global().Set("rememoryDecodeWords", js.FuncOf(decodeWordsJS))

	// Signal that WASM is ready
	js.Global().Set("rememoryReady", true)

	// Keep the Go program running
	select {}
}
