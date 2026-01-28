package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/lasmarois/vega-hub/internal/cli"
	"github.com/lasmarois/vega-hub/internal/goals"
	"github.com/spf13/cobra"
)

var migrateRegistryCmd = &cobra.Command{
	Use:   "migrate-registry",
	Short: "Migrate REGISTRY.md to registry.jsonl",
	Long:  `Converts the markdown-based REGISTRY.md to the new JSONL format.`,
	RunE:  runMigrateRegistry,
}

var migrateRegistryDryRun bool

func init() {
	rootCmd.AddCommand(migrateRegistryCmd)
	migrateRegistryCmd.Flags().BoolVar(&migrateRegistryDryRun, "dry-run", false, "Show what would be migrated without writing")
}

func runMigrateRegistry(cmd *cobra.Command, args []string) error {
	vegaDir, err := cli.GetVegaDir()
	if err != nil {
		return fmt.Errorf("failed to detect vega-missile directory: %w", err)
	}
	
	registryPath := filepath.Join(vegaDir, "goals", "REGISTRY.md")
	
	file, err := os.Open(registryPath)
	if err != nil {
		return fmt.Errorf("failed to open REGISTRY.md: %w", err)
	}
	defer file.Close()

	var entries []goals.RegistryEntry
	scanner := bufio.NewScanner(file)
	section := ""
	now := time.Now().Format(time.RFC3339)

	for scanner.Scan() {
		line := scanner.Text()

		// Detect sections
		if strings.HasPrefix(line, "## Active Goals") {
			section = "active"
			continue
		} else if strings.HasPrefix(line, "## Iced Goals") {
			section = "iced"
			continue
		} else if strings.HasPrefix(line, "## Completed Goals") {
			section = "completed"
			continue
		} else if strings.HasPrefix(line, "## ") || strings.HasPrefix(line, "---") {
			section = ""
			continue
		}

		// Skip non-table lines
		if !strings.HasPrefix(line, "|") || strings.Contains(line, "---") || strings.Contains(line, "| ID |") {
			continue
		}

		// Parse table row
		parts := strings.Split(line, "|")
		if len(parts) < 4 {
			continue
		}

		// Clean parts
		cleanParts := make([]string, 0)
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p != "" {
				cleanParts = append(cleanParts, p)
			}
		}

		if len(cleanParts) < 3 || cleanParts[0] == "" {
			continue
		}

		entry := goals.RegistryEntry{
			ID:        cleanParts[0],
			Title:     cleanParts[1],
			Projects:  parseProjectList(cleanParts[2]),
			CreatedAt: now,
			UpdatedAt: now,
		}

		switch section {
		case "active":
			entry.Status = "active"
			if len(cleanParts) >= 5 {
				entry.Phase = cleanParts[4]
			} else {
				entry.Phase = "1/?"
			}
		case "iced":
			entry.Status = "iced"
			if len(cleanParts) >= 4 {
				entry.Reason = cleanParts[3]
			}
		case "completed":
			entry.Status = "completed"
			if len(cleanParts) >= 4 {
				entry.CompletedAt = cleanParts[3]
			}
		default:
			continue
		}

		entries = append(entries, entry)
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to read REGISTRY.md: %w", err)
	}

	// Count by status
	active, iced, completed := 0, 0, 0
	for _, e := range entries {
		switch e.Status {
		case "active":
			active++
		case "iced":
			iced++
		case "completed":
			completed++
		}
	}

	fmt.Printf("Found: %d active, %d iced, %d completed goals\n", active, iced, completed)

	if migrateRegistryDryRun {
		fmt.Println("\n--- Dry run: would write the following ---")
		for _, e := range entries {
			data, _ := json.Marshal(e)
			fmt.Println(string(data))
		}
		return nil
	}

	// Write to registry.jsonl
	registry := goals.NewRegistry(vegaDir)
	if err := registry.Save(entries); err != nil {
		return fmt.Errorf("failed to write registry.jsonl: %w", err)
	}

	fmt.Printf("\n✅ Migrated %d goals to goals/registry.jsonl\n", len(entries))
	fmt.Println("\nYou can now delete goals/REGISTRY.md")

	return nil
}

func parseProjectList(s string) []string {
	projects := strings.Split(s, ",")
	result := make([]string, 0, len(projects))
	for _, p := range projects {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}
