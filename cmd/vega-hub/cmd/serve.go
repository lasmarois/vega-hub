package cmd

import (
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"

	"github.com/lasmarois/vega-hub/internal/api"
	"github.com/lasmarois/vega-hub/internal/cli"
	"github.com/lasmarois/vega-hub/internal/goals"
	"github.com/lasmarois/vega-hub/internal/hub"
	"github.com/spf13/cobra"
)

var (
	servePort int
)

// WebFS is set by main.go to provide embedded web files
var WebFS embed.FS

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the vega-hub server in foreground",
	Long: `Start the vega-hub HTTP server in the foreground.

This is the default behavior when running vega-hub without a subcommand.
The server provides:
  - HTTP API for executor communication
  - Web UI for answering questions
  - SSE for real-time updates

Use 'vega-hub start' for daemon mode with automatic port management.`,
	Run: runServe,
}

func init() {
	rootCmd.AddCommand(serveCmd)
	serveCmd.Flags().IntVarP(&servePort, "port", "p", 8080, "Port to listen on")
}

func runServe(cmd *cobra.Command, args []string) {
	dir := cli.VegaDir
	if dir == "" {
		// Try to auto-detect
		detected, err := cli.GetVegaDir()
		if err != nil {
			// Not fatal - can run without a directory
			log.Printf("Warning: %v", err)
		} else {
			dir = detected
		}
	}

	// Initialize the hub and goals parser
	h := hub.New(dir)
	p := goals.NewParser(dir)

	// Start file watcher for real-time updates
	if dir != "" {
		if err := h.StartFileWatcher(); err != nil {
			log.Printf("Warning: could not start file watcher: %v", err)
		}
	}

	// Set up API routes
	mux := http.NewServeMux()
	api.RegisterRoutes(mux, h, p)

	// Serve static files from embedded filesystem
	if os.Getenv("VEGA_HUB_DEV") == "true" {
		log.Println("Development mode: frontend at http://localhost:5173")
	} else {
		webContent, err := fs.Sub(WebFS, "web")
		if err != nil {
			log.Printf("Warning: could not load embedded web files: %v", err)
		} else {
			fileServer := http.FileServer(http.FS(webContent))
			mux.Handle("/", fileServer)
		}
	}

	addr := fmt.Sprintf(":%d", servePort)
	log.Printf("vega-hub starting on http://localhost%s", addr)
	if dir != "" {
		log.Printf("Managing directory: %s", dir)
	}

	if err := http.ListenAndServe(addr, mux); err != nil {
		cli.OutputError(cli.ExitInternalError, "server_failed", fmt.Sprintf("Server failed: %v", err), nil, nil)
	}
}
