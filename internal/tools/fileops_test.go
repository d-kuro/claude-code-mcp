package tools

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewFileOps(t *testing.T) {
	validator := &mockValidator{}
	fileOps := NewFileOps(validator)

	if fileOps == nil {
		t.Error("NewFileOps returned nil")
	}

	// FileOps validator is set correctly (interface comparison not needed)
}

func TestValidateAndSanitizePath(t *testing.T) {
	tests := []struct {
		name          string
		path          string
		mockBehavior  func(*mockValidator)
		expectError   bool
		errorContains string
	}{
		{
			name: "valid path",
			path: "/valid/path.txt",
			mockBehavior: func(m *mockValidator) {
				m.sanitizeResult = "/valid/path.txt"
				m.sanitizeError = nil
				m.validateError = nil
			},
		},
		{
			name: "sanitization error",
			path: "/invalid/path",
			mockBehavior: func(m *mockValidator) {
				m.sanitizeError = errors.New("invalid path")
			},
			expectError:   true,
			errorContains: "invalid file path",
		},
		{
			name: "validation error",
			path: "/forbidden/path",
			mockBehavior: func(m *mockValidator) {
				m.sanitizeResult = "/forbidden/path"
				m.sanitizeError = nil
				m.validateError = errors.New("forbidden path")
			},
			expectError:   true,
			errorContains: "path validation failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := &mockValidator{}
			tt.mockBehavior(validator)

			fileOps := NewFileOps(validator)
			result, err := fileOps.ValidateAndSanitizePath(tt.path)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				} else if !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error containing '%s', got: %v", tt.errorContains, err)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if result != validator.sanitizeResult {
				t.Errorf("Expected result '%s', got '%s'", validator.sanitizeResult, result)
			}
		})
	}
}

func TestGetFileInfo(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "fileops_info_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Create test file
	testFile := filepath.Join(tempDir, "test.txt")
	testContent := "test content"
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	validator := &mockValidator{}
	fileOps := NewFileOps(validator)

	t.Run("valid file", func(t *testing.T) {
		info, err := fileOps.GetFileInfo(testFile)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
			return
		}

		if info.Path != testFile {
			t.Errorf("Expected path '%s', got '%s'", testFile, info.Path)
		}

		if info.OriginalPath != testFile {
			t.Errorf("Expected original path '%s', got '%s'", testFile, info.OriginalPath)
		}

		if info.IsDir {
			t.Error("Expected IsDir to be false for file")
		}

		if info.Size != int64(len(testContent)) {
			t.Errorf("Expected size %d, got %d", len(testContent), info.Size)
		}

		if info.Mode == 0 {
			t.Error("Expected non-zero file mode")
		}
	})

	t.Run("nonexistent file", func(t *testing.T) {
		_, err := fileOps.GetFileInfo(filepath.Join(tempDir, "nonexistent.txt"))
		if err == nil {
			t.Error("Expected error for nonexistent file")
		}
		if !strings.Contains(err.Error(), "failed to stat file") {
			t.Errorf("Expected stat error, got: %v", err)
		}
	})

	t.Run("directory instead of file", func(t *testing.T) {
		testDir := filepath.Join(tempDir, "testdir")
		if err := os.Mkdir(testDir, 0755); err != nil {
			t.Fatalf("Failed to create test dir: %v", err)
		}

		_, err := fileOps.GetFileInfo(testDir)
		if err == nil {
			t.Error("Expected error for directory")
		}
		if !strings.Contains(err.Error(), "path is a directory") {
			t.Errorf("Expected directory error, got: %v", err)
		}
	})
}

func TestReadFileContent(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "fileops_read_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	testContent := "test file content\nwith multiple lines"
	testFile := filepath.Join(tempDir, "test.txt")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	validator := &mockValidator{}
	fileOps := NewFileOps(validator)

	content, info, err := fileOps.ReadFileContent(testFile)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}

	if string(content) != testContent {
		t.Errorf("Expected content '%s', got '%s'", testContent, string(content))
	}

	if info.Path != testFile {
		t.Errorf("Expected info path '%s', got '%s'", testFile, info.Path)
	}

	if info.Size != int64(len(testContent)) {
		t.Errorf("Expected size %d, got %d", len(testContent), info.Size)
	}
}

func TestCreateBackup(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "fileops_backup_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	testFile := filepath.Join(tempDir, "test.txt")
	testContent := []byte("original content")
	testMode := os.FileMode(0755)

	validator := &mockValidator{}
	fileOps := NewFileOps(validator)

	backupPath, err := fileOps.CreateBackup(testFile, testContent, testMode)
	if err != nil {
		t.Errorf("Failed to create backup: %v", err)
		return
	}

	expectedBackupPath := testFile + ".backup"
	if backupPath != expectedBackupPath {
		t.Errorf("Expected backup path '%s', got '%s'", expectedBackupPath, backupPath)
	}

	// Verify backup file exists and has correct content
	backupContent, err := os.ReadFile(backupPath)
	if err != nil {
		t.Errorf("Failed to read backup file: %v", err)
		return
	}

	if string(backupContent) != string(testContent) {
		t.Errorf("Expected backup content '%s', got '%s'", string(testContent), string(backupContent))
	}

	// Verify backup file has correct permissions
	stat, err := os.Stat(backupPath)
	if err != nil {
		t.Errorf("Failed to stat backup file: %v", err)
		return
	}

	if stat.Mode() != testMode {
		t.Errorf("Expected backup mode %v, got %v", testMode, stat.Mode())
	}
}

func TestAtomicWrite(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "fileops_atomic_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	testFile := filepath.Join(tempDir, "test.txt")
	originalContent := "original content"
	newContent := []byte("new content")

	// Create original file
	if err := os.WriteFile(testFile, []byte(originalContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	validator := &mockValidator{}
	fileOps := NewFileOps(validator)

	// Get file info
	info, err := fileOps.GetFileInfo(testFile)
	if err != nil {
		t.Fatalf("Failed to get file info: %v", err)
	}

	// Create backup
	backupPath, err := fileOps.CreateBackup(testFile, []byte(originalContent), info.Mode)
	if err != nil {
		t.Fatalf("Failed to create backup: %v", err)
	}

	t.Run("successful write", func(t *testing.T) {
		err := fileOps.AtomicWrite(testFile, newContent, info, backupPath)
		if err != nil {
			t.Errorf("AtomicWrite failed: %v", err)
			return
		}

		// Verify file content
		content, err := os.ReadFile(testFile)
		if err != nil {
			t.Errorf("Failed to read file after atomic write: %v", err)
			return
		}

		if string(content) != string(newContent) {
			t.Errorf("Expected content '%s', got '%s'", string(newContent), string(content))
		}
	})

	t.Run("write with backup restore", func(t *testing.T) {
		// Create a read-only directory to force write failure
		readOnlyDir := filepath.Join(tempDir, "readonly")
		if err := os.Mkdir(readOnlyDir, 0444); err != nil {
			t.Fatalf("Failed to create readonly dir: %v", err)
		}
		defer func() { _ = os.Chmod(readOnlyDir, 0755) }() // Ensure cleanup works

		readOnlyFile := filepath.Join(readOnlyDir, "readonly.txt")

		// For this test, we'll simulate a write failure scenario
		// by trying to write to a file in a readonly directory
		fakeInfo := &FileOpInfo{
			Path: readOnlyFile,
			Mode: 0644,
		}

		err := fileOps.AtomicWrite(readOnlyFile, newContent, fakeInfo, "")
		if err == nil {
			t.Error("Expected write to fail in readonly directory")
		}

		if !strings.Contains(err.Error(), "failed to write file") {
			t.Errorf("Expected write failure error, got: %v", err)
		}
	})
}

func TestCleanupBackup(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "fileops_cleanup_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	backupFile := filepath.Join(tempDir, "test.backup")
	if err := os.WriteFile(backupFile, []byte("backup content"), 0644); err != nil {
		t.Fatalf("Failed to create backup file: %v", err)
	}

	validator := &mockValidator{}
	fileOps := NewFileOps(validator)

	// Verify backup exists
	if _, err := os.Stat(backupFile); os.IsNotExist(err) {
		t.Fatal("Backup file should exist before cleanup")
	}

	fileOps.CleanupBackup(backupFile)

	// Verify backup is removed
	if _, err := os.Stat(backupFile); !os.IsNotExist(err) {
		t.Error("Backup file should be removed after cleanup")
	}

	// Test cleanup of nonexistent file (should not panic or error)
	fileOps.CleanupBackup(filepath.Join(tempDir, "nonexistent.backup"))
}

func TestSafeFileUpdate(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "fileops_safe_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	originalContent := "original content line 1\noriginal content line 2"
	testFile := filepath.Join(tempDir, "test.txt")
	if err := os.WriteFile(testFile, []byte(originalContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	validator := &mockValidator{}
	fileOps := NewFileOps(validator)

	t.Run("successful transformation", func(t *testing.T) {
		transformer := func(content string) (string, error) {
			return strings.ReplaceAll(content, "original", "modified"), nil
		}

		result, err := fileOps.SafeFileUpdate(testFile, transformer)
		if err != nil {
			t.Errorf("SafeFileUpdate failed: %v", err)
			return
		}

		expectedResult := "modified content line 1\nmodified content line 2"
		if result != expectedResult {
			t.Errorf("Expected result '%s', got '%s'", expectedResult, result)
		}

		// Verify file content
		newContent, err := os.ReadFile(testFile)
		if err != nil {
			t.Errorf("Failed to read file after update: %v", err)
			return
		}

		if string(newContent) != expectedResult {
			t.Errorf("Expected file content '%s', got '%s'", expectedResult, string(newContent))
		}

		// Verify backup was cleaned up
		backupPath := testFile + ".backup"
		if _, err := os.Stat(backupPath); !os.IsNotExist(err) {
			t.Error("Backup should be cleaned up after successful update")
		}
	})

	t.Run("transformation error", func(t *testing.T) {
		// Reset file content
		if err := os.WriteFile(testFile, []byte(originalContent), 0644); err != nil {
			t.Fatalf("Failed to reset test file: %v", err)
		}

		transformer := func(content string) (string, error) {
			return "", errors.New("error in transformation")
		}

		_, err := fileOps.SafeFileUpdate(testFile, transformer)
		if err == nil {
			t.Error("Expected transformation error")
			return
		}

		// Verify file content unchanged
		content, err := os.ReadFile(testFile)
		if err != nil {
			t.Errorf("Failed to read file after failed update: %v", err)
			return
		}

		if string(content) != originalContent {
			t.Error("File content should be unchanged after transformation error")
		}
	})
}

func TestValidateStringReplacement(t *testing.T) {
	validator := &mockValidator{}
	fileOps := NewFileOps(validator)

	tests := []struct {
		name           string
		replacement    StringReplacement
		operationIndex int
		expectError    bool
		errorContains  string
	}{
		{
			name: "valid replacement",
			replacement: StringReplacement{
				OldString:  "old",
				NewString:  "new",
				ReplaceAll: false,
			},
			operationIndex: -1,
		},
		{
			name: "same old and new string",
			replacement: StringReplacement{
				OldString: "same",
				NewString: "same",
			},
			operationIndex: 0,
			expectError:    true,
			errorContains:  "edit 1: old_string and new_string must be different",
		},
		{
			name: "same old and new string (no operation index)",
			replacement: StringReplacement{
				OldString: "same",
				NewString: "same",
			},
			operationIndex: -1,
			expectError:    true,
			errorContains:  "old_string and new_string must be different",
		},
		{
			name: "empty old string",
			replacement: StringReplacement{
				OldString: "",
				NewString: "new",
			},
			operationIndex: 1,
			expectError:    true,
			errorContains:  "edit 2: old_string cannot be empty",
		},
		{
			name: "empty old string (no operation index)",
			replacement: StringReplacement{
				OldString: "",
				NewString: "new",
			},
			operationIndex: -1,
			expectError:    true,
			errorContains:  "old_string cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := fileOps.ValidateStringReplacement(tt.replacement, tt.operationIndex)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				} else if !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error containing '%s', got: %v", tt.errorContains, err)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestPerformStringReplacement(t *testing.T) {
	validator := &mockValidator{}
	fileOps := NewFileOps(validator)

	tests := []struct {
		name            string
		content         string
		replacement     StringReplacement
		operationIndex  int
		expectedContent string
		expectedCount   int
		expectError     bool
		errorContains   string
	}{
		{
			name:    "single replacement",
			content: "Hello world",
			replacement: StringReplacement{
				OldString: "world",
				NewString: "Go",
			},
			operationIndex:  -1,
			expectedContent: "Hello Go",
			expectedCount:   1,
		},
		{
			name:    "replace all occurrences",
			content: "foo bar foo baz foo",
			replacement: StringReplacement{
				OldString:  "foo",
				NewString:  "qux",
				ReplaceAll: true,
			},
			operationIndex:  -1,
			expectedContent: "qux bar qux baz qux",
			expectedCount:   3,
		},
		{
			name:    "string not found",
			content: "Hello world",
			replacement: StringReplacement{
				OldString: "missing",
				NewString: "found",
			},
			operationIndex: 0,
			expectError:    true,
			errorContains:  "edit 1: old_string not found",
		},
		{
			name:    "multiple occurrences without replace all",
			content: "foo bar foo baz",
			replacement: StringReplacement{
				OldString: "foo",
				NewString: "qux",
			},
			operationIndex: 1,
			expectError:    true,
			errorContains:  "edit 2: old_string appears 2 times",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, count, err := fileOps.PerformStringReplacement(
				tt.content, tt.replacement, tt.operationIndex)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				} else if !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error containing '%s', got: %v", tt.errorContains, err)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if result != tt.expectedContent {
				t.Errorf("Expected content '%s', got '%s'", tt.expectedContent, result)
			}

			if count != tt.expectedCount {
				t.Errorf("Expected count %d, got %d", tt.expectedCount, count)
			}
		})
	}
}

func TestSingleStringReplace(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "fileops_single_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	originalContent := "Hello world"
	testFile := filepath.Join(tempDir, "test.txt")
	if err := os.WriteFile(testFile, []byte(originalContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	validator := &mockValidator{}
	fileOps := NewFileOps(validator)

	replacement := StringReplacement{
		OldString:  "world",
		NewString:  "Go",
		ReplaceAll: false,
	}

	result, err := fileOps.SingleStringReplace(testFile, replacement)
	if err != nil {
		t.Errorf("SingleStringReplace failed: %v", err)
		return
	}

	expectedResult := "Successfully replaced 1 occurrence in " + testFile
	if result != expectedResult {
		t.Errorf("Expected result '%s', got '%s'", expectedResult, result)
	}

	// Verify file content
	newContent, err := os.ReadFile(testFile)
	if err != nil {
		t.Errorf("Failed to read file after replacement: %v", err)
		return
	}

	if string(newContent) != "Hello Go" {
		t.Errorf("Expected file content 'Hello Go', got '%s'", string(newContent))
	}
}

func TestMultiStringReplace(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "fileops_multi_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	originalContent := "Hello world. This is a test."
	testFile := filepath.Join(tempDir, "test.txt")
	if err := os.WriteFile(testFile, []byte(originalContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	validator := &mockValidator{}
	fileOps := NewFileOps(validator)

	replacements := []StringReplacement{
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
	}

	result, err := fileOps.MultiStringReplace(testFile, replacements)
	if err != nil {
		t.Errorf("MultiStringReplace failed: %v", err)
		return
	}

	expectedPattern := "Successfully applied 3 edits with 3 total replacements in " + testFile
	if result != expectedPattern {
		t.Errorf("Expected result '%s', got '%s'", expectedPattern, result)
	}

	// Verify file content
	newContent, err := os.ReadFile(testFile)
	if err != nil {
		t.Errorf("Failed to read file after replacements: %v", err)
		return
	}

	expectedContent := "Hi Go. This is a example."
	if string(newContent) != expectedContent {
		t.Errorf("Expected file content '%s', got '%s'", expectedContent, string(newContent))
	}
}

// Mock validator for testing
type mockValidator struct {
	sanitizeResult string
	sanitizeError  error
	validateError  error
}

func (m *mockValidator) SanitizePath(path string) (string, error) {
	return m.sanitizeResult, m.sanitizeError
}

func (m *mockValidator) ValidatePath(path string) error {
	return m.validateError
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
