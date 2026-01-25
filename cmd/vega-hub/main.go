package main

import (
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"

	"github.com/lasmarois/vega-hub/internal/api"
	"github.com/lasmarois/vega-hub/internal/goals"
	"github.com/lasmarois/vega-hub/internal/hub"
)

//go:embed all:web
var webFS embed.FS

func main() {
	port := flag.Int("port", 8080, "Port to listen on")
	dir := flag.String("dir", "", "Vega-missile directory to manage")
	version := flag.Bool("version", false, "Print version and exit")
	flag.Parse()

	if *version {
		fmt.Println("v0.1.0")
		os.Exit(0)
	}

	// Initialize the hub and goals parser
	h := hub.New(*dir)
	p := goals.NewParser(*dir)

	// Start file watcher for real-time updates
	if *dir != "" {
		if err := h.StartFileWatcher(); err != nil {
			log.Printf("Warning: could not start file watcher: %v", err)
		}
	}

	// Set up API routes
	mux := http.NewServeMux()
	api.RegisterRoutes(mux, h, p)

	// Serve static files from embedded filesystem
	if isDev() {
		// In dev mode, proxy to Vite dev server
		log.Println("Development mode: frontend at http://localhost:5173")
	} else {
		// In production, serve embedded files
		webContent, err := fs.Sub(webFS, "web")
		if err != nil {
			log.Printf("Warning: could not load embedded web files: %v", err)
		} else {
			fileServer := http.FileServer(http.FS(webContent))
			mux.Handle("/", fileServer)
		}
	}

	addr := fmt.Sprintf(":%d", *port)
	log.Printf("vega-hub starting on http://localhost%s", addr)
	if *dir != "" {
		log.Printf("Managing directory: %s", *dir)
	}

	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func isDev() bool {
	return os.Getenv("VEGA_HUB_DEV") == "true"
}
