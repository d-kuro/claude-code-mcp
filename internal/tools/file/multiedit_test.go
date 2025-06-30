package file

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/d-kuro/claude-code-mcp/internal/tools"
)

func TestPerformMultiEdit(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "multiedit_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	tests := []struct {
		name            string
		originalContent string
		edits           []MultiEditOperation
		expectedContent string
		expectError     bool
		errorContains   string
	}{
		{
			name:            "single edit operation",
			originalContent: "Hello world",
			edits: []MultiEditOperation{
				{
					OldString: "world",
					NewString: "Go",
				},
			},
			expectedContent: "Hello Go",
		},
		{
			name:            "multiple sequential edits",
			originalContent: "Hello world. This is a test.",
			edits: []MultiEditOperation{
				{
					OldString: "Hello",
					NewString: "Hi",
				},
				{
					OldString: "world",
					NewString: "Go",
				},
				{
					OldString: "test",
					NewString: "example",
				},
			},
			expectedContent: "Hi Go. This is a example.",
		},
		{
			name:            "edits with replace all",
			originalContent: "foo bar foo baz foo",
			edits: []MultiEditOperation{
				{
					OldString:  "foo",
					NewString:  "qux",
					ReplaceAll: boolPtr(true),
				},
			},
			expectedContent: "qux bar qux baz qux",
		},
		{
			name:            "mixed replace all and single replace",
			originalContent: "foo bar foo test test example",
			edits: []MultiEditOperation{
				{
					OldString:  "foo",
					NewString:  "baz",
					ReplaceAll: boolPtr(true),
				},
				{
					OldString: "test",
					NewString: "demo",
				},
			},
			expectError:   true,
			errorContains: "edit 2: old_string appears 2 times",
		},
		{
			name:            "edit with string not found",
			originalContent: "Hello world",
			edits: []MultiEditOperation{
				{
					OldString: "missing",
					NewString: "found",
				},
			},
			expectError:   true,
			errorContains: "edit 1: old_string not found",
		},
		{
			name:            "sequential dependency edits",
			originalContent: "The quick brown fox",
			edits: []MultiEditOperation{
				{
					OldString: "quick brown",
					NewString: "slow red",
				},
				{
					OldString: "slow red fox",
					NewString: "fast blue wolf",
				},
			},
			expectedContent: "The fast blue wolf",
		},
		{
			name:            "edit with multiline content",
			originalContent: "function test() {\n  console.log('old');\n  return 'old';\n}",
			edits: []MultiEditOperation{
				{
					OldString:  "old",
					NewString:  "new",
					ReplaceAll: boolPtr(true),
				},
			},
			expectedContent: "function test() {\n  console.log('new');\n  return 'new';\n}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test file
			testFile := filepath.Join(tempDir, "multiedit_test_"+tt.name+".txt")
			if err := os.WriteFile(testFile, []byte(tt.originalContent), 0644); err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}

			// Store original mode
			stat, _ := os.Stat(testFile)
			originalMode := stat.Mode()

			result, err := performMultiEdit(testFile, tt.edits)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error containing '%s', got: %v", tt.errorContains, err)
				}

				// On error, file should be restored to original content
				content, readErr := os.ReadFile(testFile)
				if readErr == nil && string(content) != tt.originalContent {
					t.Errorf("File should be restored to original content on error. Expected: %s, Got: %s",
						tt.originalContent, string(content))
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

			// Verify result message format
			expectedPattern := "Successfully applied"
			if !strings.Contains(result, expectedPattern) {
				t.Errorf("Expected result to contain '%s', got: %s", expectedPattern, result)
			}

			if !strings.Contains(result, testFile) {
				t.Errorf("Expected result to contain file path, got: %s", result)
			}

			// Verify backup was cleaned up
			backupPath := testFile + ".backup"
			if _, err := os.Stat(backupPath); !os.IsNotExist(err) {
				t.Errorf("Backup file should have been cleaned up: %s", backupPath)
			}
		})
	}
}

func TestMultiEditBackupAndRestore(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "multiedit_backup_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	originalContent := "original content"
	testFile := filepath.Join(tempDir, "backup_test.txt")
	if err := os.WriteFile(testFile, []byte(originalContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	t.Run("failed edit restores from backup", func(t *testing.T) {
		// This edit will fail because "missing" is not found
		edits := []MultiEditOperation{
			{
				OldString: "original",
				NewString: "modified",
			},
			{
				OldString: "missing", // This will cause failure
				NewString: "found",
			},
		}

		_, err := performMultiEdit(testFile, edits)
		if err == nil {
			t.Error("Expected error for missing string")
			return
		}

		// Verify file content is restored
		content, err := os.ReadFile(testFile)
		if err != nil {
			t.Errorf("Failed to read file after failed edit: %v", err)
			return
		}

		if string(content) != originalContent {
			t.Errorf("Expected original content to be restored, got: %s", string(content))
		}

		// Verify backup was cleaned up (restored as main file)
		backupPath := testFile + ".backup"
		if _, err := os.Stat(backupPath); !os.IsNotExist(err) {
			t.Error("Backup file should not exist after restore")
		}
	})
}

func TestMultiEditErrors(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "multiedit_error_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	tests := []struct {
		name        string
		setupFunc   func() string
		edits       []MultiEditOperation
		expectError string
	}{
		{
			name: "nonexistent file",
			setupFunc: func() string {
				return filepath.Join(tempDir, "nonexistent.txt")
			},
			edits: []MultiEditOperation{
				{OldString: "test", NewString: "new"},
			},
			expectError: "failed to stat file",
		},
		{
			name: "directory instead of file",
			setupFunc: func() string {
				dirPath := filepath.Join(tempDir, "testdir")
				_ = os.Mkdir(dirPath, 0755)
				return dirPath
			},
			edits: []MultiEditOperation{
				{OldString: "test", NewString: "new"},
			},
			expectError: "path is a directory",
		},
		{
			name: "empty edits array",
			setupFunc: func() string {
				filePath := filepath.Join(tempDir, "empty_edits.txt")
				_ = os.WriteFile(filePath, []byte("content"), 0644)
				return filePath
			},
			edits:       []MultiEditOperation{},
			expectError: "", // This should be handled by the tool handler, not performMultiEdit
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testPath := tt.setupFunc()

			_, err := performMultiEdit(testPath, tt.edits)

			if tt.expectError == "" {
				// Special case for empty edits - performMultiEdit might accept it
				return
			}

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

func TestCreateMultiEditTool(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "multiedit_tool_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Create test file
	testContent := "Hello world. This is a test."
	testFile := filepath.Join(tempDir, "test.txt")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create context with mock validator
	ctx := &tools.Context{
		Validator: &mockMultiEditValidator{allowedPath: testFile},
	}

	// Create the tool
	tool := CreateMultiEditTool(ctx)

	if tool.Tool.Name != "MultiEdit" {
		t.Errorf("Expected tool name 'MultiEdit', got '%s'", tool.Tool.Name)
	}

	// Test successful multi-edit through the core function
	edits := []MultiEditOperation{
		{OldString: "Hello", NewString: "Hi"},
		{OldString: "test", NewString: "example"},
	}

	result, err := performMultiEdit(testFile, edits)
	if err != nil {
		t.Errorf("Tool function failed: %v", err)
	}

	if !strings.Contains(result, "Successfully applied") {
		t.Errorf("Expected success message, got: %s", result)
	}

	// Verify file was modified
	newContent, err := os.ReadFile(testFile)
	if err != nil {
		t.Errorf("Failed to read modified file: %v", err)
	}

	expectedContent := "Hi world. This is a example."
	if string(newContent) != expectedContent {
		t.Errorf("Expected '%s', got '%s'", expectedContent, string(newContent))
	}
}

func TestMultiEditAtomicity(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "multiedit_atomic_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	originalContent := "line1\nline2\nline3\nline4"
	testFile := filepath.Join(tempDir, "atomic_test.txt")
	if err := os.WriteFile(testFile, []byte(originalContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	t.Run("all edits succeed atomically", func(t *testing.T) {
		edits := []MultiEditOperation{
			{OldString: "line1", NewString: "first"},
			{OldString: "line2", NewString: "second"},
			{OldString: "line3", NewString: "third"},
		}

		result, err := performMultiEdit(testFile, edits)
		if err != nil {
			t.Errorf("Multi-edit failed: %v", err)
			return
		}

		if !strings.Contains(result, "3 edits") {
			t.Errorf("Expected 3 edits in result: %s", result)
		}

		// Verify all changes applied
		content, err := os.ReadFile(testFile)
		if err != nil {
			t.Errorf("Failed to read file: %v", err)
			return
		}

		expected := "first\nsecond\nthird\nline4"
		if string(content) != expected {
			t.Errorf("Expected:\n%s\nGot:\n%s", expected, string(content))
		}
	})

	t.Run("partial failure rolls back all changes", func(t *testing.T) {
		// Reset file
		if err := os.WriteFile(testFile, []byte(originalContent), 0644); err != nil {
			t.Fatalf("Failed to reset test file: %v", err)
		}

		edits := []MultiEditOperation{
			{OldString: "line1", NewString: "first"},
			{OldString: "line2", NewString: "second"},
			{OldString: "nonexistent", NewString: "fail"}, // This will fail
		}

		_, err := performMultiEdit(testFile, edits)
		if err == nil {
			t.Error("Expected error for nonexistent string")
			return
		}

		// Verify file is completely unchanged
		content, err := os.ReadFile(testFile)
		if err != nil {
			t.Errorf("Failed to read file after failed multi-edit: %v", err)
			return
		}

		if string(content) != originalContent {
			t.Errorf("File should be unchanged after failed multi-edit.\nExpected:\n%s\nGot:\n%s",
				originalContent, string(content))
		}
	})
}

func TestMultiEditEdgeCases(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "multiedit_edge_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	tests := []struct {
		name            string
		content         string
		edits           []MultiEditOperation
		expectedContent string
	}{
		{
			name:    "edit creating dependencies",
			content: "abc def ghi",
			edits: []MultiEditOperation{
				{OldString: "abc", NewString: "xyz"},
				{OldString: "xyz def", NewString: "new text"},
			},
			expectedContent: "new text ghi",
		},
		{
			name:    "edit with empty replacement",
			content: "remove this text",
			edits: []MultiEditOperation{
				{OldString: " this", NewString: ""},
			},
			expectedContent: "remove text",
		},
		{
			name:    "edit with special characters",
			content: "normal text",
			edits: []MultiEditOperation{
				{OldString: "text", NewString: "tëxt with ñ and 東"},
			},
			expectedContent: "normal tëxt with ñ and 東",
		},
		{
			name:    "edit entire file content",
			content: "old content",
			edits: []MultiEditOperation{
				{OldString: "old content", NewString: "completely new content"},
			},
			expectedContent: "completely new content",
		},
		{
			name:    "multiple edits on same line",
			content: "foo bar foo baz",
			edits: []MultiEditOperation{
				{OldString: "foo", NewString: "qux"},
			},
			expectedContent: "", // This should fail due to multiple matches
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testFile := filepath.Join(tempDir, "edge_test_"+tt.name+".txt")
			if err := os.WriteFile(testFile, []byte(tt.content), 0644); err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}

			_, err := performMultiEdit(testFile, tt.edits)

			// Special case for "multiple edits on same line" which should fail
			if tt.name == "multiple edits on same line" {
				if err == nil {
					t.Error("Expected error for multiple matches")
				}
				return
			}

			if err != nil {
				t.Errorf("Multi-edit failed: %v", err)
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

// Mock validator for testing
type mockMultiEditValidator struct {
	allowedPath string
}

func (m *mockMultiEditValidator) SanitizePath(path string) (string, error) {
	if path == m.allowedPath {
		return path, nil
	}
	if strings.Contains(path, "invalid") {
		return "", errors.New("invalid path")
	}
	return path, nil
}

func (m *mockMultiEditValidator) ValidatePath(path string) error {
	if path == m.allowedPath {
		return nil
	}
	if strings.Contains(path, "forbidden") {
		return errors.New("forbidden path")
	}
	return nil
}

func (m *mockMultiEditValidator) ValidateContent(content []byte) error {
	return nil
}

func (m *mockMultiEditValidator) ValidateCommand(cmd string, args []string) error {
	return nil
}

func (m *mockMultiEditValidator) ValidateURL(url string) error {
	return nil
}
