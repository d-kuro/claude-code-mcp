// Package tools provides unified file operation utilities for consolidating
// duplicate file handling patterns across the MCP server.
package tools

import (
	"fmt"
	"os"
	"strings"

	"github.com/d-kuro/claude-code-mcp/internal/security"
)

// FileOps provides unified file operation utilities with security validation,
// backup creation, and atomic writes.
type FileOps struct {
	validator security.Validator
}

// NewFileOps creates a new FileOps instance with the given validator.
func NewFileOps(validator security.Validator) *FileOps {
	return &FileOps{
		validator: validator,
	}
}

// FileOpInfo contains metadata about a file operation.
type FileOpInfo struct {
	Path         string
	OriginalPath string
	Mode         os.FileMode
	IsDir        bool
	Size         int64
}

// ContentTransformer defines a function that transforms file content.
type ContentTransformer func(content string) (string, error)

// ValidateAndSanitizePath validates and sanitizes a file path using the security validator.
func (f *FileOps) ValidateAndSanitizePath(path string) (string, error) {
	sanitizedPath, err := f.validator.SanitizePath(path)
	if err != nil {
		return "", fmt.Errorf("invalid file path: %w", err)
	}

	if err := f.validator.ValidatePath(sanitizedPath); err != nil {
		return "", fmt.Errorf("path validation failed: %w", err)
	}

	return sanitizedPath, nil
}

// GetFileInfo retrieves file information and performs basic validation.
func (f *FileOps) GetFileInfo(filePath string) (*FileOpInfo, error) {
	stat, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	if stat.IsDir() {
		return nil, fmt.Errorf("path is a directory, not a file")
	}

	return &FileOpInfo{
		Path:         filePath,
		OriginalPath: filePath,
		Mode:         stat.Mode(),
		IsDir:        stat.IsDir(),
		Size:         stat.Size(),
	}, nil
}

// ReadFileContent safely reads file content with proper error handling.
func (f *FileOps) ReadFileContent(filePath string) ([]byte, *FileOpInfo, error) {
	info, err := f.GetFileInfo(filePath)
	if err != nil {
		return nil, nil, err
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read file: %w", err)
	}

	return content, info, nil
}

// CreateBackup creates a backup file with the original content and permissions.
func (f *FileOps) CreateBackup(filePath string, content []byte, mode os.FileMode) (string, error) {
	backupPath := filePath + ".backup"
	if err := os.WriteFile(backupPath, content, mode); err != nil {
		return "", fmt.Errorf("failed to create backup file: %w", err)
	}
	return backupPath, nil
}

// AtomicWrite writes content to a file atomically with backup and rollback support.
func (f *FileOps) AtomicWrite(filePath string, newContent []byte, info *FileOpInfo, backupPath string) error {
	if err := os.WriteFile(filePath, newContent, info.Mode); err != nil {
		// Attempt to restore backup on write failure
		if backupPath != "" {
			if restoreErr := os.Rename(backupPath, filePath); restoreErr != nil {
				return fmt.Errorf("failed to write file and failed to restore backup: write error: %w, restore error: %v", err, restoreErr)
			}
			return fmt.Errorf("failed to write file (backup restored): %w", err)
		}
		return fmt.Errorf("failed to write file: %w", err)
	}
	return nil
}

// CleanupBackup removes a backup file, ignoring errors.
func (f *FileOps) CleanupBackup(backupPath string) {
	_ = os.Remove(backupPath)
}

// SafeFileUpdate performs a complete safe file update operation with backup and rollback.
func (f *FileOps) SafeFileUpdate(filePath string, transformer ContentTransformer) (string, error) {
	// Read original content and get file info
	originalContent, info, err := f.ReadFileContent(filePath)
	if err != nil {
		return "", err
	}

	// Create backup
	backupPath, err := f.CreateBackup(filePath, originalContent, info.Mode)
	if err != nil {
		return "", err
	}

	// Transform content
	newContent, err := transformer(string(originalContent))
	if err != nil {
		f.CleanupBackup(backupPath)
		return "", err
	}

	// Write new content atomically
	if err := f.AtomicWrite(filePath, []byte(newContent), info, backupPath); err != nil {
		return "", err
	}

	// Clean up backup on success
	f.CleanupBackup(backupPath)

	return newContent, nil
}

// StringReplacement represents a string replacement operation.
type StringReplacement struct {
	OldString  string
	NewString  string
	ReplaceAll bool
}

// ValidateStringReplacement validates a string replacement operation.
func (f *FileOps) ValidateStringReplacement(replacement StringReplacement, operationIndex int) error {
	if replacement.OldString == replacement.NewString {
		if operationIndex >= 0 {
			return fmt.Errorf("edit %d: old_string and new_string must be different", operationIndex+1)
		}
		return fmt.Errorf("old_string and new_string must be different")
	}

	if replacement.OldString == "" {
		if operationIndex >= 0 {
			return fmt.Errorf("edit %d: old_string cannot be empty", operationIndex+1)
		}
		return fmt.Errorf("old_string cannot be empty")
	}

	return nil
}

// PerformStringReplacement performs a single string replacement on content.
func (f *FileOps) PerformStringReplacement(content string, replacement StringReplacement, operationIndex int) (string, int, error) {
	if err := f.ValidateStringReplacement(replacement, operationIndex); err != nil {
		return "", 0, err
	}

	var modifiedContent string
	var replacementCount int

	if replacement.ReplaceAll {
		modifiedContent = strings.ReplaceAll(content, replacement.OldString, replacement.NewString)
		replacementCount = strings.Count(content, replacement.OldString)
	} else {
		occurrenceCount := strings.Count(content, replacement.OldString)
		if occurrenceCount == 0 {
			if operationIndex >= 0 {
				return "", 0, fmt.Errorf("edit %d: old_string not found in file", operationIndex+1)
			}
			return "", 0, fmt.Errorf("old_string not found in file")
		}
		if occurrenceCount > 1 {
			if operationIndex >= 0 {
				return "", 0, fmt.Errorf("edit %d: old_string appears %d times in file - use replace_all=true or provide more context to make it unique", operationIndex+1, occurrenceCount)
			}
			return "", 0, fmt.Errorf("old_string appears %d times in file - use replace_all=true or provide more context to make it unique", occurrenceCount)
		}

		modifiedContent = strings.Replace(content, replacement.OldString, replacement.NewString, 1)
		replacementCount = 1
	}

	if replacementCount == 0 {
		if operationIndex >= 0 {
			return "", 0, fmt.Errorf("edit %d: old_string not found in file", operationIndex+1)
		}
		return "", 0, fmt.Errorf("old_string not found in file")
	}

	return modifiedContent, replacementCount, nil
}

// SingleStringReplace performs a single string replacement operation on a file.
func (f *FileOps) SingleStringReplace(filePath string, replacement StringReplacement) (string, error) {
	var totalReplacements int

	_, err := f.SafeFileUpdate(filePath, func(content string) (string, error) {
		result, count, err := f.PerformStringReplacement(content, replacement, -1)
		totalReplacements = count
		return result, err
	})

	if err != nil {
		return "", err
	}

	if replacement.ReplaceAll {
		return fmt.Sprintf("Successfully replaced %d occurrences in %s", totalReplacements, filePath), nil
	}
	return fmt.Sprintf("Successfully replaced 1 occurrence in %s", filePath), nil
}

// MultiStringReplace performs multiple string replacement operations atomically on a file.
func (f *FileOps) MultiStringReplace(filePath string, replacements []StringReplacement) (string, error) {
	// Validate all replacements first
	for i, replacement := range replacements {
		if err := f.ValidateStringReplacement(replacement, i); err != nil {
			return "", err
		}
	}

	var totalReplacements int

	_, err := f.SafeFileUpdate(filePath, func(content string) (string, error) {
		currentContent := content
		operationCount := 0

		for i, replacement := range replacements {
			result, count, err := f.PerformStringReplacement(currentContent, replacement, i)
			if err != nil {
				return "", err
			}
			currentContent = result
			operationCount += count
		}

		totalReplacements = operationCount
		return currentContent, nil
	})

	if err != nil {
		return "", err
	}

	return fmt.Sprintf("Successfully applied %d edits with %d total replacements in %s", len(replacements), totalReplacements, filePath), nil
}
