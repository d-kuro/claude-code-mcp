package file

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/d-kuro/claude-code-mcp/internal/security"
)

// TestFileOperationsIntegration tests end-to-end file operations using all tools together
func TestFileOperationsIntegration(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "integration_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Create test project structure
	projectDir := filepath.Join(tempDir, "test_project")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("Failed to create project dir: %v", err)
	}

	// Create initial files
	files := map[string]string{
		"main.go": `package main

import "fmt"

func main() {
	fmt.Println("Hello, World!")
}`,
		"config.json": `{
  "app": "test-app",
  "version": "1.0.0",
  "debug": true
}`,
		"README.md": `# Test Project

This is a test project for integration testing.

## Features
- Feature 1
- Feature 2
`,
	}

	for filename, content := range files {
		filePath := filepath.Join(projectDir, filename)
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create %s: %v", filename, err)
		}
	}

	// Create tools context
	validator := security.NewDefaultValidator()
	_ = validator // validator is available but not needed for these tests

	// Test 1: Read operations on all files
	t.Run("read_all_files", func(t *testing.T) {
		for filename := range files {
			filePath := filepath.Join(projectDir, filename)

			content, err := readFileContent(filePath, nil, nil)
			if err != nil {
				t.Errorf("Failed to read %s: %v", filename, err)
				continue
			}

			if !strings.Contains(content, "→") {
				t.Errorf("Expected formatted output for %s", filename)
			}

			// Verify content contains expected text
			if filename == "main.go" && !strings.Contains(content, "Hello, World!") {
				t.Errorf("main.go should contain 'Hello, World!'")
			}
		}
	})

	// Test 2: Single edit operations
	t.Run("single_edits", func(t *testing.T) {
		mainFile := filepath.Join(projectDir, "main.go")

		// Edit the greeting message
		result, err := editFileContent(mainFile, "Hello, World!", "Hello, Go!", nil)
		if err != nil {
			t.Errorf("Failed to edit main.go: %v", err)
			return
		}

		if !strings.Contains(result, "Successfully replaced 1 occurrence") {
			t.Errorf("Expected success message, got: %s", result)
		}

		// Verify the change
		content, err := readFileContent(mainFile, nil, nil)
		if err != nil {
			t.Errorf("Failed to read modified main.go: %v", err)
			return
		}

		if !strings.Contains(content, "Hello, Go!") {
			t.Errorf("Expected 'Hello, Go!' in modified content")
		}
	})

	// Test 3: Multi-edit operations
	t.Run("multi_edits", func(t *testing.T) {
		configFile := filepath.Join(projectDir, "config.json")

		edits := []MultiEditOperation{
			{OldString: "test-app", NewString: "production-app"},
			{OldString: "1.0.0", NewString: "2.0.0"},
			{OldString: "true", NewString: "false"},
		}

		result, err := performMultiEdit(configFile, edits)
		if err != nil {
			t.Errorf("Failed to perform multi-edit on config.json: %v", err)
			return
		}

		if !strings.Contains(result, "Successfully applied 3 edits") {
			t.Errorf("Expected 3 edits success message, got: %s", result)
		}

		// Verify all changes
		content, err := os.ReadFile(configFile)
		if err != nil {
			t.Errorf("Failed to read modified config.json: %v", err)
			return
		}

		contentStr := string(content)
		if !strings.Contains(contentStr, "production-app") {
			t.Error("Expected 'production-app' in config")
		}
		if !strings.Contains(contentStr, "2.0.0") {
			t.Error("Expected '2.0.0' in config")
		}
		if !strings.Contains(contentStr, "false") {
			t.Error("Expected 'false' in config")
		}
	})

	// Test 4: Complex workflow with dependencies
	t.Run("complex_workflow", func(t *testing.T) {
		readmeFile := filepath.Join(projectDir, "README.md")

		// Step 1: Add a new section
		_, err := editFileContent(readmeFile, "## Features", "## Installation\n\n```bash\ngo install\n```\n\n## Features", nil)
		if err != nil {
			t.Errorf("Failed to add installation section: %v", err)
			return
		}

		// Step 2: Update features with multi-edit
		edits := []MultiEditOperation{
			{OldString: "Feature 1", NewString: "Authentication system"},
			{OldString: "Feature 2", NewString: "API endpoints"},
		}

		_, err = performMultiEdit(readmeFile, edits)
		if err != nil {
			t.Errorf("Failed to update features: %v", err)
			return
		}

		// Step 3: Add more content
		_, err = editFileContent(readmeFile, "- API endpoints", "- API endpoints\n- Database integration\n- Unit testing", nil)
		if err != nil {
			t.Errorf("Failed to add more features: %v", err)
			return
		}

		// Verify final content
		content, err := readFileContent(readmeFile, nil, nil)
		if err != nil {
			t.Errorf("Failed to read final README: %v", err)
			return
		}

		expectedElements := []string{
			"Installation",
			"go install",
			"Authentication system",
			"API endpoints",
			"Database integration",
			"Unit testing",
		}

		for _, element := range expectedElements {
			if !strings.Contains(content, element) {
				t.Errorf("Expected '%s' in final README content", element)
			}
		}
	})

	// Test 5: Error handling and recovery
	t.Run("error_handling", func(t *testing.T) {
		testFile := filepath.Join(projectDir, "error_test.txt")
		if err := os.WriteFile(testFile, []byte("original content"), 0644); err != nil {
			t.Fatalf("Failed to create error test file: %v", err)
		}

		// Test failed multi-edit (should restore original content)
		edits := []MultiEditOperation{
			{OldString: "original", NewString: "modified"},
			{OldString: "nonexistent", NewString: "fail"}, // This will fail
		}

		_, err := performMultiEdit(testFile, edits)
		if err == nil {
			t.Error("Expected error for nonexistent string")
			return
		}

		// Verify content is restored
		content, err := os.ReadFile(testFile)
		if err != nil {
			t.Errorf("Failed to read file after failed edit: %v", err)
			return
		}

		if string(content) != "original content" {
			t.Errorf("Content should be restored after failed edit, got: %s", string(content))
		}
	})
}

// TestConcurrentFileOperations tests file operations under concurrent access
func TestConcurrentFileOperations(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "concurrent_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	testFile := filepath.Join(tempDir, "concurrent_test.txt")
	initialContent := "line1\nline2\nline3\nline4\nline5"
	if err := os.WriteFile(testFile, []byte(initialContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Test concurrent reads (should be safe)
	t.Run("concurrent_reads", func(t *testing.T) {
		done := make(chan bool, 10)
		errors := make(chan error, 10)

		for i := 0; i < 10; i++ {
			go func() {
				defer func() { done <- true }()

				_, err := readFileContent(testFile, nil, nil)
				if err != nil {
					errors <- err
					return
				}
			}()
		}

		// Wait for all reads to complete
		for i := 0; i < 10; i++ {
			<-done
		}

		// Check for errors
		select {
		case err := <-errors:
			t.Errorf("Concurrent read failed: %v", err)
		default:
			// No errors, good
		}
	})

	// Test that atomic operations maintain consistency
	t.Run("atomic_operation_consistency", func(t *testing.T) {
		// Reset file
		if err := os.WriteFile(testFile, []byte(initialContent), 0644); err != nil {
			t.Fatalf("Failed to reset test file: %v", err)
		}

		// Perform a multi-edit that should be atomic
		edits := []MultiEditOperation{
			{OldString: "line1", NewString: "first"},
			{OldString: "line2", NewString: "second"},
			{OldString: "line3", NewString: "third"},
		}

		_, err := performMultiEdit(testFile, edits)
		if err != nil {
			t.Errorf("Multi-edit failed: %v", err)
			return
		}

		// Verify all changes were applied consistently
		content, err := os.ReadFile(testFile)
		if err != nil {
			t.Errorf("Failed to read file after atomic operation: %v", err)
			return
		}

		expected := "first\nsecond\nthird\nline4\nline5"
		if string(content) != expected {
			t.Errorf("Atomic operation was not consistent.\nExpected:\n%s\nGot:\n%s", expected, string(content))
		}
	})
}

// TestLargeFileOperations tests operations on large files
func TestLargeFileOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping large file test in short mode")
	}

	tempDir, err := os.MkdirTemp("", "large_file_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Create a large file (>10MB to trigger large file strategy)
	largeFile := filepath.Join(tempDir, "large_test.txt")

	t.Run("create_large_file", func(t *testing.T) {
		file, err := os.Create(largeFile)
		if err != nil {
			t.Fatalf("Failed to create large file: %v", err)
		}
		defer func() { _ = file.Close() }()

		// Write ~15MB of content
		lineContent := "This is a test line that will be repeated many times to create a large file for testing purposes.\n"
		linesNeeded := (15 * 1024 * 1024) / len(lineContent) // ~15MB

		for i := 0; i < linesNeeded; i++ {
			if i%10000 == 0 {
				// Add some variety for testing
				_, _ = fmt.Fprintf(file, "MARKER_%d: %s", i/10000, lineContent)
			} else {
				_, _ = file.WriteString(lineContent)
			}
		}
	})

	t.Run("read_large_file_with_limit", func(t *testing.T) {
		start := time.Now()
		content, err := readFileContent(largeFile, nil, intPtrIntegration(100))
		duration := time.Since(start)

		if err != nil {
			t.Errorf("Failed to read large file: %v", err)
			return
		}

		lines := strings.Split(content, "\n")
		if lines[len(lines)-1] == "" {
			lines = lines[:len(lines)-1]
		}

		if len(lines) != 100 {
			t.Errorf("Expected 100 lines, got %d", len(lines))
		}

		// Should complete reasonably quickly even for large files
		if duration > 5*time.Second {
			t.Errorf("Large file read took too long: %v", duration)
		}

		t.Logf("Read 100 lines from large file in %v", duration)
	})

	t.Run("edit_large_file", func(t *testing.T) {
		start := time.Now()

		// Edit a marker that should exist
		result, err := editFileContent(largeFile, "MARKER_0:", "EDITED_MARKER_0:", nil)
		duration := time.Since(start)

		if err != nil {
			t.Errorf("Failed to edit large file: %v", err)
			return
		}

		if !strings.Contains(result, "Successfully replaced") {
			t.Errorf("Expected success message, got: %s", result)
		}

		t.Logf("Edited large file in %v", duration)

		// Verify the edit
		content, err := readFileContent(largeFile, nil, intPtrIntegration(10))
		if err != nil {
			t.Errorf("Failed to read edited large file: %v", err)
			return
		}

		if !strings.Contains(content, "EDITED_MARKER_0:") {
			t.Error("Edit was not applied to large file")
		}
	})
}

// TestFileOperationsWithDifferentEncodings tests handling of different file encodings
func TestFileOperationsWithDifferentEncodings(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "encoding_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	tests := []struct {
		name     string
		content  string
		encoding string
	}{
		{
			name:     "utf8_with_emoji",
			content:  "Hello 世界! 🌟 This is a test with emoji and unicode.",
			encoding: "UTF-8",
		},
		{
			name:     "special_characters",
			content:  "Café naïve résumé piñata Москва العربية 日本語 한국어",
			encoding: "UTF-8",
		},
		{
			name:     "mixed_newlines",
			content:  "Line 1\nLine 2\r\nLine 3\rLine 4",
			encoding: "Mixed newlines",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testFile := filepath.Join(tempDir, tt.name+".txt")

			// Write file with special content
			if err := os.WriteFile(testFile, []byte(tt.content), 0644); err != nil {
				t.Fatalf("Failed to create %s: %v", tt.name, err)
			}

			// Test reading
			content, err := readFileContent(testFile, nil, nil)
			if err != nil {
				t.Errorf("Failed to read %s: %v", tt.name, err)
				return
			}

			// Should contain the special characters
			if !strings.Contains(content, "世界") && strings.Contains(tt.content, "世界") {
				t.Errorf("Unicode characters not preserved in %s", tt.name)
			}

			// Test editing
			if strings.Contains(tt.content, "test") {
				_, err := editFileContent(testFile, "test", "edited", nil)
				if err != nil {
					t.Errorf("Failed to edit %s: %v", tt.name, err)
					return
				}

				// Verify edit preserved encoding
				newContent, err := os.ReadFile(testFile)
				if err != nil {
					t.Errorf("Failed to read edited %s: %v", tt.name, err)
					return
				}

				if !strings.Contains(string(newContent), "edited") {
					t.Errorf("Edit not applied to %s", tt.name)
				}

				// Check that special characters are still there
				if strings.Contains(tt.content, "世界") && !strings.Contains(string(newContent), "世界") {
					t.Errorf("Unicode characters lost during edit in %s", tt.name)
				}
			}
		})
	}
}

// TestFileOperationsErrorRecovery tests comprehensive error recovery scenarios
func TestFileOperationsErrorRecovery(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "error_recovery_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Test permission error recovery
	t.Run("permission_error_recovery", func(t *testing.T) {
		testFile := filepath.Join(tempDir, "readonly.txt")
		originalContent := "original content"

		if err := os.WriteFile(testFile, []byte(originalContent), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		// Make file readonly
		if err := os.Chmod(testFile, 0444); err != nil {
			t.Fatalf("Failed to make file readonly: %v", err)
		}
		defer func() { _ = os.Chmod(testFile, 0644) }() // Restore for cleanup

		// Try to edit (should fail gracefully)
		_, err := editFileContent(testFile, "original", "modified", nil)
		if err == nil {
			t.Error("Expected permission error")
			return
		}

		// Content should be unchanged
		content, err := os.ReadFile(testFile)
		if err != nil {
			t.Errorf("Failed to read file after permission error: %v", err)
			return
		}

		if string(content) != originalContent {
			t.Error("File content should be unchanged after permission error")
		}
	})

	// Test disk space simulation (using a very large edit)
	t.Run("large_content_handling", func(t *testing.T) {
		testFile := filepath.Join(tempDir, "large_content.txt")
		originalContent := "small content"

		if err := os.WriteFile(testFile, []byte(originalContent), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		// Try to replace with very large content
		largeContent := strings.Repeat("x", 100*1024*1024) // 100MB

		// This might fail due to memory or disk constraints, but should handle gracefully
		_, err := editFileContent(testFile, "small content", largeContent, nil)

		// Whether it succeeds or fails, the file should be in a valid state
		content, readErr := os.ReadFile(testFile)
		if readErr != nil {
			t.Errorf("File should be readable after large content edit attempt: %v", readErr)
			return
		}

		// File should either contain original content (if edit failed) or new content (if edit succeeded)
		contentStr := string(content)
		if contentStr != originalContent && contentStr != largeContent {
			t.Error("File is in invalid state after large content edit")
		}

		t.Logf("Large content edit result: err=%v, content_length=%d", err, len(content))
	})
}

// Helper function for integration tests
func intPtrIntegration(i int) *int {
	return &i
}
