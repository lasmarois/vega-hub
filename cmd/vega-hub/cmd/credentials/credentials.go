package credentials

import (
	"github.com/spf13/cobra"
)

// CredentialsCmd is the parent command for credential operations
var CredentialsCmd = &cobra.Command{
	Use:   "credentials",
	Short: "Manage and validate git credentials",
	Long:  `Commands for checking and managing git credentials for projects.`,
}

func init() {
	// Subcommands are added in their respective files
}
