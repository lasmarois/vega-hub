package main

import (
	"embed"

	"github.com/lasmarois/vega-hub/cmd/vega-hub/cmd"
)

// Version is set via ldflags at build time
var Version = "dev"

//go:embed all:web
var webFS embed.FS

func main() {
	// Set version and embedded files for commands
	cmd.Version = Version
	cmd.WebFS = webFS

	// Execute CLI
	cmd.Execute()
}
