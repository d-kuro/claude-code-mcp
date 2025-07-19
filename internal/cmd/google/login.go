package google

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/oauth2"

	"github.com/d-kuro/claude-code-mcp/internal/logging"
	"github.com/d-kuro/geminiwebtools"
	"github.com/d-kuro/geminiwebtools/pkg/storage"
)

// loginFlags holds the flags for the login command
type loginFlags struct {
	port int
}

// NewLoginCmd creates a new login command
func NewLoginCmd() *cobra.Command {
	opts := &loginFlags{}

	cmd := &cobra.Command{
		Use:   "login",
		Short: "Authenticate with Google OAuth2 for web search functionality",
		Long: `Authenticate with Google OAuth2 to enable web search functionality.
This command will open a web browser to complete the OAuth2 authentication flow.
The authentication token will be stored securely for future use.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLogin(cmd.Context(), opts)
		},
	}

	// Add flags to the google-login command
	cmd.Flags().IntVarP(&opts.port, "port", "p", 8080, "Port for OAuth2 callback server")

	return cmd
}

// runLogin executes the OAuth2 authentication flow
func runLogin(ctx context.Context, opts *loginFlags) error {
	// Initialize logger
	logger := logging.NewLogger("info")
	logger.Info("Starting OAuth2 authentication flow")

	// Create credential store
	credStore, err := createCredentialStore()
	if err != nil {
		return fmt.Errorf("failed to create credential store: %w", err)
	}

	// Check if already authenticated
	if credStore.HasToken() {
		token, err := credStore.LoadToken()
		if err == nil && token.Valid() {
			logger.Info("Already authenticated. Use 'claude-code-mcp google logout' command to re-authenticate.")
			fmt.Println("✓ Already authenticated. Use 'claude-code-mcp google logout' command to re-authenticate.")
			return nil
		}
	}

	// Use minimal configuration browser authentication
	fmt.Printf("Opening browser for authentication...\n")

	// Use geminiwebtools client for authentication
	client, err := geminiwebtools.NewClient(
		geminiwebtools.WithCredentialStore(credStore),
	)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	logger.Debug("Starting browser authentication with geminiwebtools")
	// The client automatically handles authentication when needed
	// We can trigger it by making a simple request
	_, err = client.Search(ctx, "test")
	if err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	// Get the token from the credential store after authentication
	token, err := credStore.LoadToken()
	if err != nil {
		return fmt.Errorf("failed to load token after authentication: %w", err)
	}

	logger.Debug("OAuth2 authentication successful")

	// Store the token (geminiwebtools handles this automatically)
	// but we still call StoreToken to ensure compatibility
	if err := credStore.StoreToken(token); err != nil {
		return fmt.Errorf("authentication succeeded but failed to store token: %w", err)
	}

	// Get user information from token
	userInfo, err := getUserInfo(ctx, token)
	if err != nil {
		logger.Warn("Failed to get user information", slog.Any("error", err))
		userInfo = &UserInfo{Email: "authenticated@example.com"}
	}

	// Calculate expiration time
	expiresIn := time.Until(token.Expiry)
	if token.Expiry.IsZero() {
		expiresIn = 0
	}

	logger.Info("OAuth2 authentication completed", slog.String("email", userInfo.Email))

	fmt.Printf("✓ Authentication successful!\n")
	fmt.Printf("  Email: %s\n", userInfo.Email)
	if expiresIn > 0 {
		fmt.Printf("  Token expires in: %s\n", expiresIn.Round(time.Second))
	}
	fmt.Printf("  Token stored in: %s\n", getConfigDir())

	return nil
}

// createCredentialStore creates a credential store for OAuth2 authentication
func createCredentialStore() (storage.CredentialStore, error) {
	// Get config directory
	configDir := getConfigDir()

	// Create filesystem-based credential store
	store, err := storage.NewFileSystemStore(configDir)
	if err != nil {
		// Fall back to default location if custom directory fails
		if store, err = storage.NewFileSystemStore(""); err != nil {
			return nil, fmt.Errorf("failed to create credential store: %w", err)
		}
	}

	return store, nil
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

	return homeDir + "/.claude-code-mcp"
}

// UserInfo represents user information from OAuth2 token
type UserInfo struct {
	Email string `json:"email"`
	Name  string `json:"name"`
}

// getUserInfo retrieves user information using the OAuth2 token
func getUserInfo(ctx context.Context, token *oauth2.Token) (*UserInfo, error) {
	client := oauth2.NewClient(ctx, oauth2.StaticTokenSource(token))

	resp, err := client.Get("https://www.googleapis.com/oauth2/v1/userinfo")
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("user info request failed with status: %d", resp.StatusCode)
	}

	var userInfo UserInfo
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, fmt.Errorf("failed to decode user info: %w", err)
	}

	return &userInfo, nil
}
