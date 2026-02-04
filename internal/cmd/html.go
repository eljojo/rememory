package cmd

import (
	"fmt"
	"os"

	"github.com/eljojo/rememory/internal/html"
	"github.com/spf13/cobra"
)

var htmlCmd = &cobra.Command{
	Use:   "html [recover|create]",
	Short: "Generate standalone HTML files for static hosting",
	Long: `Generate standalone HTML files that can be hosted on a static website.

Commands:
  recover  Generate recover.html (recovery tool for collecting shares)
  create   Generate rememory.html (bundle creation tool)

Both HTML files are completely self-contained with embedded WASM binary,
JavaScript, and CSS. They work fully offline.

Examples:
  rememory html recover > recover.html
  rememory html create > rememory.html
  rememory html recover --output dist/recover.html`,
	Args: cobra.ExactArgs(1),
	RunE: runHTML,
}

var htmlOutputFile string

func init() {
	htmlCmd.Flags().StringVarP(&htmlOutputFile, "output", "o", "", "Output file path (default: stdout)")
	rootCmd.AddCommand(htmlCmd)
}

func runHTML(cmd *cobra.Command, args []string) error {
	subcommand := args[0]

	// Get embedded WASM binary
	wasmBytes := html.GetWASMBytes()
	if len(wasmBytes) == 0 {
		return fmt.Errorf("WASM binary not embedded - rebuild with 'make build'")
	}

	var content string
	githubURL := fmt.Sprintf("https://github.com/eljojo/rememory/releases/tag/%s", version)

	switch subcommand {
	case "recover":
		// Generate generic recover.html (without personalization)
		content = html.GenerateRecoverHTML(wasmBytes, version, githubURL, nil)

	case "create":
		// Generate rememory.html (bundle creation tool)
		content = html.GenerateRememoryHTML(wasmBytes, version, githubURL)

	default:
		return fmt.Errorf("unknown subcommand: %s (use 'recover' or 'create')", subcommand)
	}

	// Output to file or stdout
	if htmlOutputFile != "" {
		if err := os.WriteFile(htmlOutputFile, []byte(content), 0644); err != nil {
			return fmt.Errorf("writing file: %w", err)
		}
		fmt.Fprintf(os.Stderr, "Generated %s (%s)\n", htmlOutputFile, formatSize(int64(len(content))))
	} else {
		fmt.Print(content)
	}

	return nil
}
