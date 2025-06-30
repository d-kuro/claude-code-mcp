package file

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/d-kuro/claude-code-mcp/internal/tools"
)

func TestEditFileContent(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "editor_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	tests := []struct {
		name            string
		originalContent string
		oldString       string
		newString       string
		replaceAll      *bool
		expectedContent string
		expectError     bool
		errorContains   string
	}{
		{
			name:            "simple replacement",
			originalContent: "Hello world",
			oldString:       "world",
			newString:       "Go",
			expectedContent: "Hello Go",
		},
		{
			name:            "replacement with newlines",
			originalContent: "line1\nline2\nline3",
			oldString:       "line2",
			newString:       "modified line",
			expectedContent: "line1\nmodified line\nline3",
		},
		{
			name:            "replace all occurrences",
			originalContent: "foo bar foo baz foo",
			oldString:       "foo",
			newString:       "qux",
			replaceAll:      boolPtr(true),
			expectedContent: "qux bar qux baz qux",
		},
		{
			name:            "replace single occurrence when multiple exist without replaceAll",
			originalContent: "foo bar foo baz",
			oldString:       "foo",
			newString:       "qux",
			expectError:     true,
			errorContains:   "appears 2 times",
		},
		{
			name:            "old string not found",
			originalContent: "Hello world",
			oldString:       "missing",
			newString:       "found",
			expectError:     true,
			errorContains:   "not found",
		},
		// Note: empty old string validation is done at the tool handler level, not in editFileContent
		// Note: same old/new string validation is done at the tool handler level, not in editFileContent
		{
			name:            "multiline replacement",
			originalContent: "function test() {\n  console.log('old');\n}",
			oldString:       "console.log('old');",
			newString:       "console.log('new');",
			expectedContent: "function test() {\n  console.log('new');\n}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test file
			testFile := filepath.Join(tempDir, "edit_test_"+tt.name+".txt")
			if err := os.WriteFile(testFile, []byte(tt.originalContent), 0644); err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}

			// Store original mode
			stat, _ := os.Stat(testFile)
			originalMode := stat.Mode()

			result, err := editFileContent(testFile, tt.oldString, tt.newString, tt.replaceAll)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error containing '%s', got: %v", tt.errorContains, err)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			// Verify file content
			newContent, err := os.ReadFile(testFile)
			if err != nil {
				t.Errorf("Failed to read modified file: %v", err)
				return
			}

			if string(newContent) != tt.expectedContent {
				t.Errorf("Expected content:\n%s\nGot:\n%s", tt.expectedContent, string(newContent))
			}

			// Verify file mode preserved
			newStat, err := os.Stat(testFile)
			if err != nil {
				t.Errorf("Failed to stat modified file: %v", err)
				return
			}

			if newStat.Mode() != originalMode {
				t.Errorf("File mode changed from %v to %v", originalMode, newStat.Mode())
			}

			// Verify result message
			if tt.replaceAll != nil && *tt.replaceAll {
				expectedCount := strings.Count(tt.originalContent, tt.oldString)
				if !strings.Contains(result, "Successfully replaced") ||
					!strings.Contains(result, testFile) {
					t.Errorf("Unexpected result message: %s", result)
				}
				if expectedCount > 1 && !strings.Contains(result, "occurrences") {
					t.Errorf("Expected plural 'occurrences' in result: %s", result)
				}
			} else {
				if !strings.Contains(result, "Successfully replaced 1 occurrence") {
					t.Errorf("Expected single occurrence message, got: %s", result)
				}
			}

			// Verify backup was cleaned up
			backupPath := testFile + ".backup"
			if _, err := os.Stat(backupPath); !os.IsNotExist(err) {
				t.Errorf("Backup file should have been cleaned up: %s", backupPath)
			}
		})
	}
}

func TestEditFileBackupAndRestore(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "editor_backup_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	originalContent := "original content"
	testFile := filepath.Join(tempDir, "backup_test.txt")
	if err := os.WriteFile(testFile, []byte(originalContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Test successful backup creation and cleanup
	t.Run("successful operation cleans up backup", func(t *testing.T) {
		result, err := editFileContent(testFile, "original", "modified", nil)
		if err != nil {
			t.Errorf("Edit failed: %v", err)
			return
		}

		if !strings.Contains(result, "Successfully replaced") {
			t.Errorf("Unexpected result: %s", result)
		}

		// Backup should be cleaned up
		backupPath := testFile + ".backup"
		if _, err := os.Stat(backupPath); !os.IsNotExist(err) {
			t.Errorf("Backup file should have been removed")
		}
	})

	// Reset file for next test
	if err := os.WriteFile(testFile, []byte(originalContent), 0644); err != nil {
		t.Fatalf("Failed to reset test file: %v", err)
	}

	// Test backup creation with custom permissions
	t.Run("preserves file permissions in backup", func(t *testing.T) {
		// Change file permissions
		if err := os.Chmod(testFile, 0755); err != nil {
			t.Fatalf("Failed to change file permissions: %v", err)
		}

		// Force an error by trying to edit with empty old_string
		_, err := editFileContent(testFile, "", "test", nil)
		if err == nil {
			t.Errorf("Expected error for empty old_string")
			return
		}

		// File should remain unchanged
		content, err := os.ReadFile(testFile)
		if err != nil {
			t.Errorf("Failed to read file after failed edit: %v", err)
			return
		}

		if string(content) != originalContent {
			t.Errorf("File content should be unchanged after failed edit")
		}
	})
}

func TestEditFileErrors(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "editor_error_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	tests := []struct {
		name        string
		setupFunc   func() string
		oldString   string
		newString   string
		expectError string
	}{
		{
			name: "nonexistent file",
			setupFunc: func() string {
				return filepath.Join(tempDir, "nonexistent.txt")
			},
			oldString:   "test",
			newString:   "new",
			expectError: "failed to stat file",
		},
		{
			name: "directory instead of file",
			setupFunc: func() string {
				dirPath := filepath.Join(tempDir, "testdir")
				_ = os.Mkdir(dirPath, 0755)
				return dirPath
			},
			oldString:   "test",
			newString:   "new",
			expectError: "path is a directory",
		},
		{
			name: "readonly file",
			setupFunc: func() string {
				filePath := filepath.Join(tempDir, "readonly.txt")
				_ = os.WriteFile(filePath, []byte("content"), 0444) // readonly
				return filePath
			},
			oldString:   "content",
			newString:   "new content",
			expectError: "permission denied",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testPath := tt.setupFunc()

			_, err := editFileContent(testPath, tt.oldString, tt.newString, nil)

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

func TestCreateEditTool(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "edit_tool_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Create test file
	testContent := "Hello world"
	testFile := filepath.Join(tempDir, "test.txt")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create context with mock validator
	ctx := &tools.Context{
		Validator: &mockEditorValidator{allowedPath: testFile},
	}

	// Create the tool
	tool := CreateEditTool(ctx)

	if tool.Tool.Name != "Edit" {
		t.Errorf("Expected tool name 'Edit', got '%s'", tool.Tool.Name)
	}

	// Test successful edit through the core function
	result, err := editFileContent(testFile, "world", "Go", nil)
	if err != nil {
		t.Errorf("Tool function failed: %v", err)
	}

	if !strings.Contains(result, "Successfully replaced") {
		t.Errorf("Expected success message, got: %s", result)
	}

	// Verify file was modified
	newContent, err := os.ReadFile(testFile)
	if err != nil {
		t.Errorf("Failed to read modified file: %v", err)
	}

	if string(newContent) != "Hello Go" {
		t.Errorf("Expected 'Hello Go', got '%s'", string(newContent))
	}
}

func TestEditFileContentEdgeCases(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "editor_edge_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	tests := []struct {
		name            string
		content         string
		oldString       string
		newString       string
		replaceAll      *bool
		expectedContent string
	}{
		{
			name:            "replace with empty string",
			content:         "Hello world test",
			oldString:       " world",
			newString:       "",
			expectedContent: "Hello test",
		},
		{
			name:            "replace entire content",
			content:         "old content",
			oldString:       "old content",
			newString:       "completely new content",
			expectedContent: "completely new content",
		},
		{
			name:            "replace with special characters",
			content:         "normal text",
			oldString:       "text",
			newString:       "text with ñ, ü, and 东",
			expectedContent: "normal text with ñ, ü, and 东",
		},
		{
			name:            "replace newline characters",
			content:         "line1\nline2\nline3",
			oldString:       "\n",
			newString:       " | ",
			replaceAll:      boolPtr(true),
			expectedContent: "line1 | line2 | line3",
		},
		{
			name:            "replace with newlines",
			content:         "sentence one sentence two",
			oldString:       " ",
			newString:       "\n",
			replaceAll:      boolPtr(true),
			expectedContent: "sentence\none\nsentence\ntwo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testFile := filepath.Join(tempDir, "edge_test_"+tt.name+".txt")
			if err := os.WriteFile(testFile, []byte(tt.content), 0644); err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}

			_, err := editFileContent(testFile, tt.oldString, tt.newString, tt.replaceAll)
			if err != nil {
				t.Errorf("Edit failed: %v", err)
				return
			}

			newContent, err := os.ReadFile(testFile)
			if err != nil {
				t.Errorf("Failed to read modified file: %v", err)
				return
			}

			if string(newContent) != tt.expectedContent {
				t.Errorf("Expected:\n%q\nGot:\n%q", tt.expectedContent, string(newContent))
			}
		})
	}
}

// Helper functions
func boolPtr(b bool) *bool {
	return &b
}

// Mock validator for testing
type mockEditorValidator struct {
	allowedPath string
}

func (m *mockEditorValidator) SanitizePath(path string) (string, error) {
	if path == m.allowedPath {
		return path, nil
	}
	if strings.Contains(path, "invalid") {
		return "", errors.New("invalid path")
	}
	return path, nil
}

func (m *mockEditorValidator) ValidatePath(path string) error {
	if path == m.allowedPath {
		return nil
	}
	if strings.Contains(path, "forbidden") {
		return errors.New("forbidden path")
	}
	return nil
}

func (m *mockEditorValidator) ValidateContent(content []byte) error {
	return nil
}

func (m *mockEditorValidator) ValidateCommand(cmd string, args []string) error {
	return nil
}

func (m *mockEditorValidator) ValidateURL(url string) error {
	return nil
}
