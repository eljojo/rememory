package cmd

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/eljojo/rememory/internal/html"
	"github.com/eljojo/rememory/internal/serve"
	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start a self-hosted ReMemory server",
	Long: `Start a self-hosted ReMemory web server for creating and recovering bundles.

The server provides a web interface for bundle creation (using WASM) and
recovery (using native JavaScript crypto). An admin password protects
bundle deletion.

On localhost, the first visit prompts for an admin password. A server
listening on other interfaces (--host 0.0.0.0, as in Docker) generates one
at first start and prints it once, so no visitor can claim it. Set
REMEMORY_ADMIN_PASSWORD to choose it yourself; it replaces a stored one.

After setup:
  - If no manifest exists, the create page is shown
  - If a manifest exists, the recover page is shown

Examples:
  rememory serve
  rememory serve --port 3000 --data /var/lib/rememory
  rememory serve --max-manifest-size 100MB`,
	RunE: runServe,
}

func init() {
	serveCmd.Flags().StringP("port", "p", "8080", "Port to listen on")
	serveCmd.Flags().String("host", "127.0.0.1", "Host to bind to")
	serveCmd.Flags().StringP("data", "d", "./rememory-data", "Data directory for storing bundles and config")
	serveCmd.Flags().String("max-manifest-size", "50MB", "Maximum MANIFEST.age size (e.g. 50MB, 1GB)")
	rootCmd.AddCommand(serveCmd)
}

// flagOrEnv returns the flag value if explicitly set, otherwise falls back to
// the environment variable. If neither is set, returns the flag's default.
func flagOrEnv(cmd *cobra.Command, flagName, envName string) string {
	if cmd.Flags().Changed(flagName) {
		v, _ := cmd.Flags().GetString(flagName)
		return v
	}
	if v := os.Getenv(envName); v != "" {
		return v
	}
	v, _ := cmd.Flags().GetString(flagName)
	return v
}

func runServe(cmd *cobra.Command, args []string) error {
	port := flagOrEnv(cmd, "port", "REMEMORY_PORT")
	host := flagOrEnv(cmd, "host", "REMEMORY_HOST")
	dataDir := flagOrEnv(cmd, "data", "REMEMORY_DATA")
	maxSizeStr := flagOrEnv(cmd, "max-manifest-size", "REMEMORY_MAX_MANIFEST_SIZE")

	maxSize, err := parseSize(maxSizeStr)
	if err != nil {
		return fmt.Errorf("invalid --max-manifest-size: %w", err)
	}

	createWASM := html.GetCreateWASMBytes()
	if len(createWASM) == 0 {
		return fmt.Errorf("create.wasm not embedded — rebuild with 'make build'")
	}

	srv, err := serve.New(serve.Config{
		Host:            host,
		Port:            port,
		DataDir:         dataDir,
		MaxManifestSize: maxSize,
		Version:         version,
		BuildDate:       buildDate,
	})
	if err != nil {
		return fmt.Errorf("starting server: %w", err)
	}

	generated, err := serve.EnsureAdminPassword(srv.Store(), host, os.Getenv("REMEMORY_ADMIN_PASSWORD"))
	if err != nil {
		return fmt.Errorf("setting admin password: %w", err)
	}
	if generated != "" {
		fmt.Printf("\nThis server is reachable from other machines, so it chose its admin password:\n\n")
		fmt.Printf("    %s\n\n", generated)
		fmt.Printf("It won't be printed again. Keep it somewhere safe. To choose your own,\n")
		fmt.Printf("set REMEMORY_ADMIN_PASSWORD, or delete admin.age from the data directory and restart.\n\n")
	}

	addr := host + ":" + port
	fmt.Printf("ReMemory server listening on http://%s\n", addr)
	fmt.Printf("Data directory: %s\n", dataDir)
	return srv.ListenAndServe(addr)
}

// parseSize parses a human-readable size string (e.g. "50MB", "1GB") into bytes.
func parseSize(s string) (int, error) {
	s = strings.TrimSpace(s)
	s = strings.ToUpper(s)

	// Check longer suffixes first to avoid "B" matching "MB", "GB", "KB"
	type sizeUnit struct {
		suffix string
		mult   int
	}
	units := []sizeUnit{
		{"GB", 1 << 30},
		{"MB", 1 << 20},
		{"KB", 1 << 10},
		{"B", 1},
	}

	for _, u := range units {
		if strings.HasSuffix(s, u.suffix) {
			numStr := strings.TrimSuffix(s, u.suffix)
			numStr = strings.TrimSpace(numStr)
			n, err := strconv.ParseFloat(numStr, 64)
			if err != nil {
				return 0, fmt.Errorf("invalid number: %s", numStr)
			}
			return int(n * float64(u.mult)), nil
		}
	}

	// Try plain number (bytes)
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("unrecognized size format: %s", s)
	}
	return n, nil
}
