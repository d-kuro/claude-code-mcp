// Package main implements the CLI commands for the Claude Code MCP server.
package main

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/spf13/cobra"

	"github.com/d-kuro/claude-code-mcp/internal/logging"
	"github.com/d-kuro/claude-code-mcp/internal/storage"
)

// googleStatusCmd represents the google-status command
var googleStatusCmd = &cobra.Command{
	Use:   "google-status",
	Short: "Check Google OAuth2 authentication status",
	Long: `Check the current OAuth2 authentication status.
This command will display information about the stored authentication token,
including expiration time and user information.`,
	RunE: runStatus,
}

// runStatus checks the OAuth2 authentication status
func runStatus(cmd *cobra.Command, args []string) error {
	// Initialize logger
	logger := logging.NewLogger("info")
	logger.Debug("Checking OAuth2 authentication status")

	// Create credential store
	credStore, err := createCredentialStore()
	if err != nil {
		return fmt.Errorf("failed to create credential store: %w", err)
	}

	// Check if token exists
	if !credStore.HasToken() {
		fmt.Println("❌ Not authenticated")
		fmt.Println("   Run 'google-login' command to authenticate")
		return nil
	}

	// Load the token
	token, err := credStore.LoadToken()
	if err != nil {
		return fmt.Errorf("failed to load authentication token: %w", err)
	}

	// Check if token is expired
	isExpired := storage.IsTokenExpired(token)

	if isExpired {
		fmt.Println("❌ Authentication token has expired")
		fmt.Printf("   Token expired: %s\n", token.Expiry.Format(time.RFC3339))
		if token.RefreshToken != "" {
			fmt.Println("   Refresh token available - authentication may be automatically renewed")
		}
		fmt.Println("   Run 'google-login' command to re-authenticate")
		return nil
	}

	// Get user information
	userInfo, err := getUserInfo(cmd.Context(), token)
	email := "unknown@example.com"
	if err == nil && userInfo != nil {
		email = userInfo.Email
	}

	// Calculate expiration time
	expiresIn := time.Until(token.Expiry)
	if token.Expiry.IsZero() {
		expiresIn = 0
	}

	fmt.Println("✓ Authenticated")
	fmt.Printf("   Email: %s\n", email)
	fmt.Printf("   Token type: %s\n", token.TokenType)
	if expiresIn > 0 {
		fmt.Printf("   Expires in: %s\n", expiresIn.Round(time.Second))
		fmt.Printf("   Expires at: %s\n", token.Expiry.Format(time.RFC3339))
	} else {
		fmt.Println("   Expires: Never")
	}
	fmt.Printf("   Has refresh token: %v\n", token.RefreshToken != "")
	fmt.Printf("   Token stored in: %s\n", getConfigDir())

	logger.Debug("OAuth2 status check completed",
		slog.Bool("authenticated", true),
		slog.String("email", email))

	return nil
}
