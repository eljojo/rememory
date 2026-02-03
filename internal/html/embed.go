package html

import (
	_ "embed"
)

// Embedded assets for the recovery HTML
// These files are embedded at compile time

//go:embed assets/recover.html
var recoverHTMLTemplate string

//go:embed assets/app.js
var appJS string

//go:embed assets/styles.css
var stylesCSS string

//go:embed assets/wasm_exec.js
var wasmExecJS string

//go:embed assets/recover.wasm
var recoverWASM []byte

// GetWASMBytes returns the embedded WASM binary.
func GetWASMBytes() []byte {
	return recoverWASM
}
