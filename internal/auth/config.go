// Package auth provides OAuth2 authentication configuration and utilities.
package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"os/exec"
	"runtime"
	"time"

	"golang.org/x/oauth2"
)

// OAuth2 configuration constants
const (
	// Other defaults
	BaseURL     = "https://accounts.google.com"
	DefaultPort = 8080
	RedirectURI = "http://localhost:8080/oauth2callback"
)

// GenerateSecureState generates a cryptographically secure random state parameter
func GenerateSecureState() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate secure state: %w", err)
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// FindAvailablePort finds an available port for the OAuth2 callback server
func FindAvailablePort() (int, error) {
	// Try common ports first
	commonPorts := []int{8080, 8081, 8082, 8083, 8084, 8085}

	for _, port := range commonPorts {
		if isPortAvailable(port) {
			return port, nil
		}
	}

	// If common ports are taken, find any available port
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		return 0, fmt.Errorf("failed to find available port: %w", err)
	}
	defer func() { _ = listener.Close() }()

	addr := listener.Addr().(*net.TCPAddr)
	return addr.Port, nil
}

// isPortAvailable checks if a port is available for use
func isPortAvailable(port int) bool {
	timeout := time.Second
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("localhost:%d", port), timeout)
	if err != nil {
		return true // Port is available
	}
	_ = conn.Close()
	return false // Port is in use
}

// OpenBrowser opens the default browser to the specified URL
func OpenBrowser(url string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start"}
	case "darwin":
		cmd = "open"
	default: // "linux", "freebsd", "openbsd", "netbsd"
		cmd = "xdg-open"
	}

	args = append(args, url)
	return exec.Command(cmd, args...).Start()
}

// BuildAuthURL constructs the OAuth2 authorization URL
func BuildAuthURL(config *oauth2.Config, state string) string {
	return config.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.ApprovalForce)
}

// ExchangeCodeForToken exchanges an authorization code for an access token
func ExchangeCodeForToken(ctx context.Context, config *oauth2.Config, code string) (*oauth2.Token, error) {
	token, err := config.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code for token: %w", err)
	}

	return token, nil
}

// ValidateToken validates an OAuth2 token by making a test API call
func ValidateToken(ctx context.Context, token *oauth2.Token) error {
	if token == nil {
		return fmt.Errorf("token is nil")
	}

	if token.Expiry.Before(time.Now()) {
		return fmt.Errorf("token is expired")
	}

	// Test the token with a simple API call
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequestWithContext(ctx, "GET", "https://www.googleapis.com/oauth2/v1/userinfo", nil)
	if err != nil {
		return fmt.Errorf("failed to create validation request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token.AccessToken)

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to validate token: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("token validation failed with status: %d", resp.StatusCode)
	}

	return nil
}
