package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

// Version is set via ldflags at build time
var Version = "dev"

// Global flags
var (
	jsonOutput bool
	quietMode  bool
	vegaDir    string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "vega-hub",
	Short: "Real-time communication hub for vega-missile",
	Long: `vega-hub is a communication hub that enables direct interaction
between humans and Claude Code executors in vega-missile.

It provides:
  - HTTP API for executor communication
  - Web UI for answering questions
  - Real-time updates via SSE
  - Goal and executor lifecycle management

Run 'vega-hub serve' to start the server, or use subcommands for other operations.`,
	Version: Version,
	// Default to showing help if no subcommand specified
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	// Global flags available to all commands
	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "Output in JSON format")
	rootCmd.PersistentFlags().BoolVarP(&quietMode, "quiet", "q", false, "Minimal output")
	rootCmd.PersistentFlags().StringVarP(&vegaDir, "dir", "d", "", "Vega-missile directory (default: auto-detect)")

	// Set version template
	rootCmd.SetVersionTemplate("{{.Version}}\n")
}

// GetVegaDir returns the vega-missile directory, auto-detecting if not specified
func GetVegaDir() (string, error) {
	if vegaDir != "" {
		return vegaDir, nil
	}

	// Auto-detect: look for .claude/ directory walking up from cwd
	dir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("could not get working directory: %w", err)
	}

	for dir != "/" {
		if _, err := os.Stat(dir + "/.claude"); err == nil {
			if _, err := os.Stat(dir + "/goals"); err == nil {
				return dir, nil
			}
		}
		dir = dir[:len(dir)-len("/"+filepath.Base(dir))]
	}

	return "", fmt.Errorf("could not auto-detect vega-missile directory (no .claude/ and goals/ found)")
}
