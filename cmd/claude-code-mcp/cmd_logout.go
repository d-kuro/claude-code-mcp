// Package main implements the CLI commands for the Claude Code MCP server.
package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/d-kuro/claude-code-mcp/internal/logging"
)

// googleLogoutCmd represents the google-logout command
var googleLogoutCmd = &cobra.Command{
	Use:   "google-logout",
	Short: "Remove stored Google OAuth2 authentication credentials",
	Long: `Remove stored OAuth2 authentication credentials.
This command will delete the stored authentication token, requiring re-authentication
before using web search functionality.`,
	RunE: runLogout,
}

// runLogout removes the stored OAuth2 token
func runLogout(cmd *cobra.Command, args []string) error {
	// Initialize logger
	logger := logging.NewLogger("info")
	logger.Info("Starting OAuth2 logout")

	// Create credential store
	credStore, err := createCredentialStore()
	if err != nil {
		return fmt.Errorf("failed to create credential store: %w", err)
	}

	// Check if token exists
	if !credStore.HasToken() {
		fmt.Println("No authentication token found. Already logged out.")
		return nil
	}

	// Delete the token
	if err := credStore.ClearToken(); err != nil {
		return fmt.Errorf("failed to delete authentication token: %w", err)
	}

	logger.Info("OAuth2 logout completed")
	fmt.Println("✓ Successfully logged out. Authentication token removed.")

	return nil
}
