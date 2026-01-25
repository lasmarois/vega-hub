package credentials

import (
	"fmt"

	"github.com/lasmarois/vega-hub/internal/cli"
	"github.com/lasmarois/vega-hub/internal/credentials"
	"github.com/lasmarois/vega-hub/internal/goals"
	"github.com/spf13/cobra"
)

// CheckResult contains the result of credential validation
type CheckResult struct {
	User     credentials.User              `json:"user"`
	Project  string                        `json:"project"`
	Service  credentials.GitService        `json:"service"`
	Valid    bool                          `json:"valid"`
	Statuses []credentials.CredentialStatus `json:"statuses"`
}

var checkCmd = &cobra.Command{
	Use:   "check <project>",
	Short: "Check git credentials for a project",
	Long: `Validate that the current user has git credentials configured for a project.

Examples:
  vega-hub credentials check my-api
  vega-hub credentials check my-api --json

This command checks:
  - GitHub CLI auth (gh auth status)
  - GitLab CLI auth (glab auth status)
  - ~/.netrc entries
  - SSH agent keys

The command returns success (exit 0) if any valid credential is found,
or error (exit 1) with fix instructions if no credentials are configured.`,
	Args: cobra.ExactArgs(1),
	Run:  runCheck,
}

func init() {
	CredentialsCmd.AddCommand(checkCmd)
}

func runCheck(c *cobra.Command, args []string) {
	project := args[0]

	// Get vega-missile directory
	vegaDir, err := cli.GetVegaDir()
	if err != nil {
		cli.OutputError(cli.ExitValidationError, "no_directory", err.Error(), nil, []cli.ErrorOption{
			{Flag: "dir", Description: "Specify vega-missile directory explicitly"},
		})
	}

	// Get current user
	user, err := credentials.GetCurrentUser()
	if err != nil {
		cli.OutputError(cli.ExitInternalError, "user_detection_failed",
			"Failed to detect current user",
			map[string]string{"error": err.Error()},
			nil)
	}

	// Parse project to get git remote
	parser := goals.NewParser(vegaDir)
	proj, err := parser.ParseProject(project)
	if err != nil {
		cli.OutputError(cli.ExitNotFound, "project_not_found",
			fmt.Sprintf("Project '%s' not found", project),
			map[string]string{
				"error":   err.Error(),
				"project": project,
			},
			[]cli.ErrorOption{
				{Action: "list", Description: "Run: vega-hub project list"},
			})
	}

	// Determine git service from remote URL
	var service *credentials.GitService
	if proj.GitRemote != "" {
		service, err = credentials.ParseGitService(proj.GitRemote)
		if err != nil {
			cli.OutputError(cli.ExitStateError, "invalid_remote",
				"Could not parse git remote URL",
				map[string]string{
					"url":   proj.GitRemote,
					"error": err.Error(),
				},
				nil)
		}
	} else if proj.Upstream != "" {
		service, err = credentials.ParseGitService(proj.Upstream)
		if err != nil {
			cli.OutputError(cli.ExitStateError, "invalid_upstream",
				"Could not parse upstream URL",
				map[string]string{
					"url":   proj.Upstream,
					"error": err.Error(),
				},
				nil)
		}
	} else {
		cli.OutputError(cli.ExitStateError, "no_remote",
			fmt.Sprintf("No git remote configured for project '%s'", project),
			map[string]string{"project": project},
			[]cli.ErrorOption{
				{Action: "configure", Description: fmt.Sprintf("Add 'Upstream: <git-url>' to projects/%s.md", project)},
			})
	}

	// Validate credentials
	result := credentials.ValidateCredentials(user, service)

	// Build output
	checkResult := CheckResult{
		User:     result.User,
		Project:  project,
		Service:  result.Service,
		Valid:    result.Valid,
		Statuses: result.Statuses,
	}

	if result.Valid {
		cli.Output(cli.Result{
			Success: true,
			Action:  "credentials_check",
			Message: fmt.Sprintf("Valid credentials found for %s (%s)", project, service.Host),
			Data:    checkResult,
		})

		// Human-readable summary
		if !cli.JSONOutput {
			fmt.Printf("\n  User: %s\n", user.Username)
			fmt.Printf("  Service: %s (%s)\n", service.Name, service.Host)
			fmt.Println("\n  Credential sources:")
			for _, status := range result.Statuses {
				mark := "✗"
				if status.Available {
					mark = "✓"
				}
				fmt.Printf("    %s %s: %s\n", mark, status.Source, status.Message)
			}
		}
	} else {
		// Build error options from fix options
		var errorOptions []cli.ErrorOption
		for _, fix := range result.FixOptions {
			if fix.Command != "" {
				errorOptions = append(errorOptions, cli.ErrorOption{
					Action:      fix.Command,
					Description: fix.Description,
				})
			} else if fix.Manual != "" {
				errorOptions = append(errorOptions, cli.ErrorOption{
					Action:      "manual",
					Description: fmt.Sprintf("%s: %s", fix.Description, fix.Manual),
				})
			}
		}

		// Build details map
		details := map[string]string{
			"user":    user.Username,
			"project": project,
			"service": service.Host,
		}

		cli.OutputError(cli.ExitValidationError, "no_credentials",
			fmt.Sprintf("No valid credentials found for %s (%s)", project, service.Host),
			details,
			errorOptions)
	}
}
