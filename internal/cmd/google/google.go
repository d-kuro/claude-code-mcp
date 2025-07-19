package google

import (
	"github.com/spf13/cobra"
)

// NewGoogleCmd creates a new google command with subcommands
func NewGoogleCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "google",
		Short: "Manage Google OAuth2 authentication",
		Long:  `Commands for managing Google OAuth2 authentication used by the web search functionality.`,
	}

	// Add subcommands
	cmd.AddCommand(NewLoginCmd())
	cmd.AddCommand(NewLogoutCmd())
	cmd.AddCommand(NewStatusCmd())

	return cmd
}
