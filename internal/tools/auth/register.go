// Package auth provides registration for OAuth2 authentication tools.
package auth

import (
	"os"
	"path/filepath"
)

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

	return filepath.Join(homeDir, ".config", "claude-code-mcp")
}

// GetDefaultConfigDir returns the default configuration directory path
func GetDefaultConfigDir() string {
	return getConfigDir()
}
