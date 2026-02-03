package cmd

import (
	"github.com/spf13/cobra"
)

// version is set at build time via -ldflags
var version = "dev"

var rootCmd = &cobra.Command{
	Use:   "rememory",
	Short: "Encrypt secrets and split access among trusted friends",
	Long: `ReMemory encrypts a manifest of secrets with age, splits the passphrase
using Shamir's Secret Sharing, and creates recovery bundles for trusted friends.

Create a project:    rememory init my-recovery
Seal the manifest:   rememory seal
Recover from shares: rememory recover share1.txt share2.txt share3.txt`,
}

func Execute(v string) error {
	version = v
	rootCmd.Version = v
	return rootCmd.Execute()
}

// Color helpers (ANSI escape codes)
func green(s string) string {
	return "\033[32m" + s + "\033[0m"
}

func yellow(s string) string {
	return "\033[33m" + s + "\033[0m"
}

func red(s string) string {
	return "\033[31m" + s + "\033[0m"
}
