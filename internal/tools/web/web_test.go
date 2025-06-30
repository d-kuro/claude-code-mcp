// Package web provides tests for web operation tools.
package web

import (
	"testing"

	"github.com/d-kuro/claude-code-mcp/internal/tools"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// mockValidator provides a mock implementation of the Validator interface for testing.
type mockValidator struct{}

func (m *mockValidator) ValidatePath(path string) error                  { return nil }
func (m *mockValidator) ValidateCommand(cmd string, args []string) error { return nil }
func (m *mockValidator) ValidateURL(url string) error                    { return nil }
func (m *mockValidator) SanitizePath(path string) (string, error)        { return path, nil }

// mockLogger provides a mock implementation of the Logger interface for testing.
type mockLogger struct{}

func (m *mockLogger) Debug(msg string, args ...any)             {}
func (m *mockLogger) Info(msg string, args ...any)              {}
func (m *mockLogger) Warn(msg string, args ...any)              {}
func (m *mockLogger) Error(msg string, args ...any)             {}
func (m *mockLogger) WithTool(toolName string) tools.Logger     { return m }
func (m *mockLogger) WithSession(sessionID string) tools.Logger { return m }

// createTestContext creates a test context with mock dependencies.
func createTestContext() *tools.Context {
	return &tools.Context{
		Logger:    &mockLogger{},
		Validator: &mockValidator{},
	}
}

func TestCreateWebFetchTool(t *testing.T) {
	ctx := createTestContext()
	tool := CreateWebFetchTool(ctx)

	if tool == nil {
		t.Fatal("CreateWebFetchTool returned nil")
	}

	// Test tool was created successfully
	// The MCP SDK handles the internal structure
}

func TestCreateWebSearchTool(t *testing.T) {
	ctx := createTestContext()
	tool := CreateWebSearchTool(ctx)

	if tool == nil {
		t.Fatal("CreateWebSearchTool returned nil")
	}

	// Test tool was created successfully
	// The MCP SDK handles the internal structure
}

func TestExtractDomain(t *testing.T) {
	tests := []struct {
		url      string
		expected string
	}{
		{"https://example.com/path", "example.com"},
		{"http://sub.example.com", "sub.example.com"},
		{"https://EXAMPLE.COM", "example.com"},
		{"invalid-url", ""},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			result := extractDomain(tt.url)
			if result != tt.expected {
				t.Errorf("extractDomain(%q) = %q, want %q", tt.url, result, tt.expected)
			}
		})
	}
}

func TestIsBlocked(t *testing.T) {
	blockedDomains := []string{"blocked.com", "evil.org"}

	tests := []struct {
		domain   string
		expected bool
	}{
		{"blocked.com", true},
		{"sub.blocked.com", true},
		{"evil.org", true},
		{"sub.evil.org", true},
		{"allowed.com", false},
		{"blocked.com.safe.org", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.domain, func(t *testing.T) {
			result := isBlocked(tt.domain, blockedDomains)
			if result != tt.expected {
				t.Errorf("isBlocked(%q, %v) = %v, want %v", tt.domain, blockedDomains, result, tt.expected)
			}
		})
	}
}

func TestIsAllowed(t *testing.T) {
	allowedDomains := []string{"allowed.com", "safe.org"}

	tests := []struct {
		domain   string
		expected bool
	}{
		{"allowed.com", true},
		{"sub.allowed.com", true},
		{"safe.org", true},
		{"sub.safe.org", true},
		{"blocked.com", false},
		{"allowed.com.evil.org", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.domain, func(t *testing.T) {
			result := isAllowed(tt.domain, allowedDomains)
			if result != tt.expected {
				t.Errorf("isAllowed(%q, %v) = %v, want %v", tt.domain, allowedDomains, result, tt.expected)
			}
		})
	}
}

func TestCreateErrorResponse(t *testing.T) {
	message := "Test error message"
	resp := createErrorResponse(message)

	if resp == nil {
		t.Fatal("createErrorResponse returned nil")
	}

	if !resp.IsError {
		t.Error("Expected IsError to be true")
	}

	if len(resp.Content) != 1 {
		t.Fatalf("Expected 1 content item, got %d", len(resp.Content))
	}

	textContent, ok := resp.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatal("Expected TextContent")
	}

	if textContent.Text != message {
		t.Errorf("Expected content %q, got %q", message, textContent.Text)
	}
}

func TestWebFetchArgsValidation(t *testing.T) {
	args := WebFetchArgs{
		URL:    "https://example.com",
		Prompt: "Summarize this page",
	}

	// Test that args struct has the correct fields
	if args.URL == "" {
		t.Error("URL field should not be empty")
	}
	if args.Prompt == "" {
		t.Error("Prompt field should not be empty")
	}
}

func TestWebSearchArgsValidation(t *testing.T) {
	args := WebSearchArgs{
		Query:          "Go programming",
		AllowedDomains: []string{"golang.org"},
		BlockedDomains: []string{"badsite.com"},
	}

	// Test that args struct has the correct fields
	if args.Query == "" {
		t.Error("Query field should not be empty")
	}
	if len(args.AllowedDomains) == 0 {
		t.Error("AllowedDomains should have values")
	}
	if len(args.BlockedDomains) == 0 {
		t.Error("BlockedDomains should have values")
	}
}
