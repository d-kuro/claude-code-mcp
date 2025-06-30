// Package auth provides registration for OAuth2 authentication tools.
package auth

import (
	"os"
	"path/filepath"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/d-kuro/claude-code-mcp/internal/storage"
	"github.com/d-kuro/claude-code-mcp/internal/tools"
)

// CreateAuthTools creates all OAuth2 authentication tools using MCP SDK patterns.
// NOTE: OAuth2 authentication tools have been moved to CLI commands.
// This function now returns an empty slice.
func CreateAuthTools(ctx *tools.Context) []*mcp.ServerTool {
	// OAuth2 authentication is now handled via CLI commands:
	// - claude-code-mcp google-login
	// - claude-code-mcp google-logout
	// - claude-code-mcp google-status
	return []*mcp.ServerTool{}
}

// getConfigDir returns the configuration directory for storing OAuth2 credentials
func getConfigDir() string {
	// Check for custom config directory environment variable
	if configDir := os.Getenv("CLAUDE_CODE_MCP_CONFIG_DIR"); configDir != "" {
		return configDir
	}

	// Use default location in user's home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	return filepath.Join(homeDir, ".claude-code-mcp")
}

// GetDefaultConfigDir returns the default configuration directory path
func GetDefaultConfigDir() string {
	return getConfigDir()
}

// GetCredentialStore creates and returns a credential store instance
func GetCredentialStore() (storage.CredentialStore, error) {
	configDir := getConfigDir()
	return storage.NewFileSystemStore(configDir)
}

// InitializeAuthTools initializes the authentication tools and ensures
// the configuration directory exists
func InitializeAuthTools() error {
	configDir := getConfigDir()

	// Ensure configuration directory exists
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return err
	}

	return nil
}

// CleanupAuthTools performs cleanup operations for authentication tools
func CleanupAuthTools() error {
	store, err := GetCredentialStore()
	if err != nil {
		return err
	}

	return store.Close()
}

// IsAuthenticated checks if the user is currently authenticated
func IsAuthenticated() bool {
	store, err := GetCredentialStore()
	if err != nil {
		return false
	}

	defer func() { _ = store.Close() }()

	return store.HasToken()
}

// GetAuthenticationStatus returns the current authentication status
func GetAuthenticationStatus() (bool, string, error) {
	store, err := GetCredentialStore()
	if err != nil {
		return false, "", err
	}

	defer func() { _ = store.Close() }()

	if !store.HasToken() {
		return false, "No authentication token found", nil
	}

	token, err := store.LoadToken()
	if err != nil {
		return false, "Failed to load authentication token", err
	}

	if storage.IsTokenExpired(token) {
		return false, "Authentication token has expired", nil
	}

	return true, "Authenticated", nil
}

// ClearAuthentication removes stored authentication tokens
func ClearAuthentication() error {
	store, err := GetCredentialStore()
	if err != nil {
		return err
	}

	defer func() { _ = store.Close() }()

	return store.DeleteToken()
}
