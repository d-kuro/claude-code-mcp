package file

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/d-kuro/claude-code-mcp/internal/tools"
)

func TestReadFileContent(t *testing.T) {
	// Create temporary directory for tests
	tempDir, err := os.MkdirTemp("", "reader_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	tests := []struct {
		name           string
		content        string
		offset         *int
		limit          *int
		expectedLines  int
		expectedFormat bool // whether to expect line numbers
		expectError    bool
	}{
		{
			name:           "empty file",
			content:        "",
			expectedLines:  0,
			expectedFormat: false,
		},
		{
			name:           "small file - all lines",
			content:        "line1\nline2\nline3",
			expectedLines:  3,
			expectedFormat: true,
		},
		{
			name:           "small file - with limit",
			content:        "line1\nline2\nline3\nline4\nline5",
			limit:          intPtrReader(3),
			expectedLines:  3,
			expectedFormat: true,
		},
		{
			name:           "small file - with offset",
			content:        "line1\nline2\nline3\nline4\nline5",
			offset:         intPtrReader(2),
			expectedLines:  3,
			expectedFormat: true,
		},
		{
			name:           "small file - with offset and limit",
			content:        "line1\nline2\nline3\nline4\nline5",
			offset:         intPtrReader(1),
			limit:          intPtrReader(2),
			expectedLines:  2,
			expectedFormat: true,
		},
		{
			name:           "single line without newline",
			content:        "single line",
			expectedLines:  1,
			expectedFormat: true,
		},
		{
			name:           "lines with very long content",
			content:        strings.Repeat("a", MaxLineLength+100) + "\nshort line",
			expectedLines:  2,
			expectedFormat: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test file
			testFile := filepath.Join(tempDir, "test_"+tt.name+".txt")
			if err := os.WriteFile(testFile, []byte(tt.content), 0644); err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}

			result, err := readFileContent(testFile, tt.offset, tt.limit)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			// Check empty file handling
			if tt.content == "" {
				if !strings.Contains(result, "empty contents") {
					t.Errorf("Expected empty file warning, got: %s", result)
				}
				return
			}

			// Count lines in result
			lines := strings.Split(result, "\n")
			if lines[len(lines)-1] == "" {
				lines = lines[:len(lines)-1] // Remove empty last line from split
			}

			if len(lines) != tt.expectedLines {
				t.Errorf("Expected %d lines, got %d", tt.expectedLines, len(lines))
			}

			// Check line number formatting
			if tt.expectedFormat && tt.expectedLines > 0 {
				firstLine := lines[0]
				if !strings.Contains(firstLine, "→") {
					t.Errorf("Expected line number formatting in: %s", firstLine)
				}
			}

			// Check long line truncation
			if strings.Contains(tt.content, strings.Repeat("a", MaxLineLength+100)) {
				found := false
				for _, line := range lines {
					if strings.Contains(line, "... (truncated)") {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected line truncation marker")
				}
			}
		})
	}
}

func TestReadLargeFile(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "reader_large_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Create large file content
	var content strings.Builder
	for i := 0; i < 1000; i++ {
		content.WriteString(strings.Repeat("x", 100))
		content.WriteByte('\n')
	}

	testFile := filepath.Join(tempDir, "large_test.txt")
	if err := os.WriteFile(testFile, []byte(content.String()), 0644); err != nil {
		t.Fatalf("Failed to create large test file: %v", err)
	}

	// Test reading with limits
	result, err := readFileContent(testFile, nil, intPtrReader(10))
	if err != nil {
		t.Errorf("Failed to read large file: %v", err)
		return
	}

	lines := strings.Split(result, "\n")
	if lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	if len(lines) != 10 {
		t.Errorf("Expected 10 lines, got %d", len(lines))
	}
}

func TestReadFileErrors(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "reader_error_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	tests := []struct {
		name        string
		setupFunc   func() string
		expectError string
	}{
		{
			name: "nonexistent file",
			setupFunc: func() string {
				return filepath.Join(tempDir, "nonexistent.txt")
			},
			expectError: "failed to open file",
		},
		{
			name: "directory instead of file",
			setupFunc: func() string {
				dirPath := filepath.Join(tempDir, "testdir")
				_ = os.Mkdir(dirPath, 0755)
				return dirPath
			},
			expectError: "path is a directory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testPath := tt.setupFunc()

			_, err := readFileContent(testPath, nil, nil)

			if err == nil {
				t.Errorf("Expected error but got none")
				return
			}

			if !strings.Contains(err.Error(), tt.expectError) {
				t.Errorf("Expected error containing '%s', got: %v", tt.expectError, err)
			}
		})
	}
}

func TestReadStrategySwitching(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "reader_strategy_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Test small file strategy
	smallContent := "line1\nline2\nline3"
	smallFile := filepath.Join(tempDir, "small.txt")
	if err := os.WriteFile(smallFile, []byte(smallContent), 0644); err != nil {
		t.Fatalf("Failed to create small test file: %v", err)
	}

	// Test large file strategy trigger
	var largeContent strings.Builder
	for i := 0; i < 50000; i++ { // Create content that exceeds LargeFileThreshold
		largeContent.WriteString("This is a test line that will help us exceed the large file threshold.\n")
	}

	largeFile := filepath.Join(tempDir, "large.txt")
	if err := os.WriteFile(largeFile, []byte(largeContent.String()), 0644); err != nil {
		t.Fatalf("Failed to create large test file: %v", err)
	}

	// Both should work and produce formatted output
	smallResult, err := readFileContent(smallFile, nil, nil)
	if err != nil {
		t.Errorf("Failed to read small file: %v", err)
	}

	if !strings.Contains(smallResult, "→") {
		t.Errorf("Expected formatted output from small file")
	}

	largeResult, err := readFileContent(largeFile, nil, intPtrReader(5))
	if err != nil {
		t.Errorf("Failed to read large file: %v", err)
	}

	if !strings.Contains(largeResult, "→") {
		t.Errorf("Expected formatted output from large file")
	}
}

func TestCreateReadTool(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "read_tool_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Create test file
	testContent := "line1\nline2\nline3"
	testFile := filepath.Join(tempDir, "test.txt")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create context with mock validator
	ctx := &tools.Context{
		Validator: &mockValidator{allowedPath: testFile},
	}

	// Create the tool
	tool := CreateReadTool(ctx)

	if tool.Tool.Name != "Read" {
		t.Errorf("Expected tool name 'Read', got '%s'", tool.Tool.Name)
	}

	// Test the core functionality directly (MCP integration would require more setup)
	result, err := readFileContent(testFile, nil, intPtrReader(2))
	if err != nil {
		t.Errorf("Tool function failed: %v", err)
	}

	lines := strings.Split(result, "\n")
	if lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	if len(lines) != 2 {
		t.Errorf("Expected 2 lines with limit, got %d", len(lines))
	}
}

func TestWriteFormattedLine(t *testing.T) {
	tests := []struct {
		name       string
		lineNumber int
		line       string
		expected   string
	}{
		{
			name:       "single digit line number",
			lineNumber: 5,
			line:       "test line",
			expected:   "    5→test line",
		},
		{
			name:       "double digit line number",
			lineNumber: 42,
			line:       "another test",
			expected:   "   42→another test",
		},
		{
			name:       "triple digit line number",
			lineNumber: 123,
			line:       "third test",
			expected:   "  123→third test",
		},
		{
			name:       "four digit line number",
			lineNumber: 1000,
			line:       "fourth test",
			expected:   " 1000→fourth test",
		},
		{
			name:       "five digit line number",
			lineNumber: 12345,
			line:       "fifth test",
			expected:   "12345→fifth test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var builder strings.Builder
			writeFormattedLine(&builder, tt.lineNumber, tt.line)

			result := builder.String()
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

// Helper functions
func intPtrReader(i int) *int {
	return &i
}

// Mock validator for testing
type mockValidator struct {
	allowedPath string
}

func (m *mockValidator) SanitizePath(path string) (string, error) {
	if path == m.allowedPath {
		return path, nil
	}
	if strings.Contains(path, "invalid") {
		return "", errors.New("invalid path")
	}
	return path, nil
}

func (m *mockValidator) ValidatePath(path string) error {
	if path == m.allowedPath {
		return nil
	}
	if strings.Contains(path, "forbidden") {
		return errors.New("forbidden path")
	}
	return nil
}

func (m *mockValidator) ValidateContent(content []byte) error {
	return nil
}

func (m *mockValidator) ValidateCommand(cmd string, args []string) error {
	return nil
}

func (m *mockValidator) ValidateURL(url string) error {
	return nil
}
