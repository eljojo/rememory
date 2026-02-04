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

// Embedded assets for the bundle creation HTML

//go:embed assets/maker.html
var makerHTMLTemplate string

//go:embed assets/create-app.js
var createAppJS string

// createWASM is set at build time for the CLI binary (not for WASM builds)
// This avoids circular dependency since create.wasm embeds the html package
var createWASM []byte

// GetRecoverWASMBytes returns the embedded recovery-only WASM binary.
// This smaller WASM is used in recover.html for bundle distribution.
func GetRecoverWASMBytes() []byte {
	return recoverWASM
}

// GetCreateWASMBytes returns the full WASM binary with bundle creation.
// This larger WASM is used in maker.html for the creation tool.
// Note: Must be set via SetCreateWASMBytes before use (done in CLI init).
func GetCreateWASMBytes() []byte {
	return createWASM
}

// SetCreateWASMBytes sets the create.wasm bytes.
// Called by CLI initialization to avoid circular embedding.
func SetCreateWASMBytes(data []byte) {
	createWASM = data
}
