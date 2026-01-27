package cmd

import (
	"os"

	"github.com/lasmarois/vega-hub/cmd/vega-hub/cmd/credentials"
	"github.com/lasmarois/vega-hub/cmd/vega-hub/cmd/executor"
	"github.com/lasmarois/vega-hub/cmd/vega-hub/cmd/goal"
	"github.com/lasmarois/vega-hub/cmd/vega-hub/cmd/lock"
	"github.com/lasmarois/vega-hub/cmd/vega-hub/cmd/project"
	"github.com/lasmarois/vega-hub/cmd/vega-hub/cmd/worktree"
	"github.com/lasmarois/vega-hub/internal/cli"
	"github.com/spf13/cobra"
)

// Version is set via ldflags at build time
var Version = "dev"

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
	// Sync flag values to cli package globals before running any command
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Flags are already bound to cli package variables via init()
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	// Global flags - bind directly to cli package variables
	rootCmd.PersistentFlags().BoolVar(&cli.JSONOutput, "json", false, "Output in JSON format")
	rootCmd.PersistentFlags().BoolVarP(&cli.QuietMode, "quiet", "q", false, "Minimal output")
	rootCmd.PersistentFlags().StringVarP(&cli.VegaDir, "dir", "d", "", "Vega-missile directory (default: auto-detect)")

	// Set version template
	rootCmd.SetVersionTemplate("{{.Version}}\n")

	// Add subcommands
	rootCmd.AddCommand(goal.GoalCmd)
	rootCmd.AddCommand(project.ProjectCmd)
	rootCmd.AddCommand(executor.ExecutorCmd)
	rootCmd.AddCommand(credentials.CredentialsCmd)
	rootCmd.AddCommand(worktree.WorktreeCmd)
	rootCmd.AddCommand(lock.LockCmd)
}
