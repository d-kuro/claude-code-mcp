package file

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestGlobFiles(t *testing.T) {
	// Create temporary test directory
	tempDir, err := os.MkdirTemp("", "globtest")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		_ = os.RemoveAll(tempDir)
	}()

	// Create test files
	testFiles := []string{
		"test.go",
		"main.go",
		"src/handler.go",
		"src/utils.go",
		"pkg/config.js",
		"pkg/lib.ts",
		"docs/readme.md",
		"scripts/build.sh",
	}

	for _, file := range testFiles {
		fullPath := filepath.Join(tempDir, file)
		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create dir %s: %v", dir, err)
		}
		if err := os.WriteFile(fullPath, []byte("test content"), 0644); err != nil {
			t.Fatalf("Failed to create file %s: %v", fullPath, err)
		}
		// Add a small delay to ensure different modification times
		time.Sleep(1 * time.Millisecond)
	}

	tests := []struct {
		name        string
		pattern     string
		expectFiles []string
	}{
		{
			name:        "match go files",
			pattern:     "*.go",
			expectFiles: []string{"test.go", "main.go"},
		},
		{
			name:        "match recursive js files",
			pattern:     "**/*.js",
			expectFiles: []string{"config.js"},
		},
		{
			name:        "recursive go files",
			pattern:     "**/*.go",
			expectFiles: []string{"test.go", "main.go", "handler.go", "utils.go"},
		},
		{
			name:        "no matches",
			pattern:     "*.nonexistent",
			expectFiles: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := globFilesWithFind(tempDir, tt.pattern)
			if err != nil {
				t.Fatalf("globFiles() error = %v", err)
			}

			if len(tt.expectFiles) == 0 {
				// Either empty result or "No files found" message is acceptable
				return
			}

			// Check that all expected files are mentioned in the result
			for _, expectedFile := range tt.expectFiles {
				if !strings.Contains(result, expectedFile) {
					t.Errorf("Expected to find '%s' in result: %s", expectedFile, result)
				}
			}
		})
	}
}

func TestMatchGlobPattern(t *testing.T) {
	tests := []struct {
		pattern string
		path    string
		want    bool
	}{
		// Basic patterns
		{"*.go", "main.go", true},
		{"*.go", "main.js", false},
		{"test*", "test.go", true},
		{"test*", "main.go", false},

		// Recursive patterns
		{"**/*.go", "main.go", true},
		{"**/*.go", "src/main.go", true},
		{"**/*.go", "src/deep/main.go", true},
		{"**/*.go", "main.js", false},

		// Directory patterns
		{"src/**", "src/main.go", true},
		{"src/**", "src/deep/main.go", true},
		{"src/**", "pkg/main.go", false},

		// Complex patterns
		{"src/**/*.go", "src/main.go", true},
		{"src/**/*.go", "src/deep/main.go", true},
		{"src/**/*.go", "pkg/main.go", false},
		{"src/**/*.go", "src/main.js", false},
	}

	for _, tt := range tests {
		t.Run(tt.pattern+"_"+tt.path, func(t *testing.T) {
			got, err := matchGlobPattern(tt.pattern, tt.path)
			if err != nil {
				t.Errorf("matchGlobPattern() error = %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("matchGlobPattern(%q, %q) = %v, want %v", tt.pattern, tt.path, got, tt.want)
			}
		})
	}
}
