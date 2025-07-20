// Package file provides file operation tools using the MCP SDK patterns.
package file

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/d-kuro/claude-code-mcp/internal/prompts"
	"github.com/d-kuro/claude-code-mcp/internal/tools"
)

// ReadArgs represents the arguments for the Read tool.
type ReadArgs struct {
	FilePath string `json:"file_path"`
	Offset   *int   `json:"offset,omitempty"`
	Limit    *int   `json:"limit,omitempty"`
}

// CreateReadTool creates the Read tool using MCP SDK patterns.
func CreateReadTool(ctx *tools.Context) *tools.ServerTool {
	handler := func(ctxReq context.Context, session *mcp.ServerSession, params *mcp.CallToolParamsFor[ReadArgs]) (*mcp.CallToolResultFor[any], error) {
		args := params.Arguments

		sanitizedPath, err := ctx.Validator.SanitizePath(args.FilePath)
		if err != nil {
			return &mcp.CallToolResultFor[any]{
				Content: []mcp.Content{&mcp.TextContent{Text: "Error: Invalid file path: " + err.Error()}},
				IsError: true,
			}, nil
		}

		if err := ctx.Validator.ValidatePath(sanitizedPath); err != nil {
			return &mcp.CallToolResultFor[any]{
				Content: []mcp.Content{&mcp.TextContent{Text: "Error: Path validation failed: " + err.Error()}},
				IsError: true,
			}, nil
		}

		content, err := readFileContent(sanitizedPath, args.Offset, args.Limit)
		if err != nil {
			return &mcp.CallToolResultFor[any]{
				Content: []mcp.Content{&mcp.TextContent{Text: "Error: " + err.Error()}},
				IsError: true,
			}, nil
		}

		return &mcp.CallToolResultFor[any]{
			Content: []mcp.Content{&mcp.TextContent{Text: content}},
		}, nil
	}

	tool := &mcp.Tool{
		Name:        "Read",
		Description: prompts.ReadToolDoc,
	}

	return &tools.ServerTool{
		Tool: tool,
		RegisterFunc: func(server *mcp.Server) {
			mcp.AddTool(server, tool, handler)
		},
	}
}

// WriteArgs represents the arguments for the Write tool.
type WriteArgs struct {
	FilePath string `json:"file_path"`
	Content  string `json:"content"`
}

// CreateWriteTool creates the Write tool using MCP SDK patterns.
func CreateWriteTool(ctx *tools.Context) *tools.ServerTool {
	handler := func(ctxReq context.Context, session *mcp.ServerSession, params *mcp.CallToolParamsFor[WriteArgs]) (*mcp.CallToolResultFor[any], error) {
		args := params.Arguments

		sanitizedPath, err := ctx.Validator.SanitizePath(args.FilePath)
		if err != nil {
			return &mcp.CallToolResultFor[any]{
				Content: []mcp.Content{&mcp.TextContent{Text: "Error: Invalid file path: " + err.Error()}},
				IsError: true,
			}, nil
		}

		if err := ctx.Validator.ValidatePath(sanitizedPath); err != nil {
			return &mcp.CallToolResultFor[any]{
				Content: []mcp.Content{&mcp.TextContent{Text: "Error: Path validation failed: " + err.Error()}},
				IsError: true,
			}, nil
		}

		bytesWritten, err := writeFileContent(sanitizedPath, args.Content)
		if err != nil {
			return &mcp.CallToolResultFor[any]{
				Content: []mcp.Content{&mcp.TextContent{Text: "Error: " + err.Error()}},
				IsError: true,
			}, nil
		}

		return &mcp.CallToolResultFor[any]{
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("File written successfully to %s (%d bytes)", sanitizedPath, bytesWritten)}},
		}, nil
	}

	tool := &mcp.Tool{
		Name:        "Write",
		Description: prompts.WriteToolDoc,
	}

	return &tools.ServerTool{
		Tool: tool,
		RegisterFunc: func(server *mcp.Server) {
			mcp.AddTool(server, tool, handler)
		},
	}
}

// LSArgs represents the arguments for the LS tool.
type LSArgs struct {
	Path   string   `json:"path"`
	Ignore []string `json:"ignore,omitempty"`
}

// CreateLSTool creates the LS tool using MCP SDK patterns.
func CreateLSTool(ctx *tools.Context) *tools.ServerTool {
	handler := func(ctxReq context.Context, session *mcp.ServerSession, params *mcp.CallToolParamsFor[LSArgs]) (*mcp.CallToolResultFor[any], error) {
		args := params.Arguments

		sanitizedPath, err := ctx.Validator.SanitizePath(args.Path)
		if err != nil {
			return &mcp.CallToolResultFor[any]{
				Content: []mcp.Content{&mcp.TextContent{Text: "Error: Invalid path: " + err.Error()}},
				IsError: true,
			}, nil
		}

		if err := ctx.Validator.ValidatePath(sanitizedPath); err != nil {
			return &mcp.CallToolResultFor[any]{
				Content: []mcp.Content{&mcp.TextContent{Text: "Error: Path validation failed: " + err.Error()}},
				IsError: true,
			}, nil
		}

		content, err := listDirectoryWithLS(sanitizedPath, args.Ignore)
		if err != nil {
			return &mcp.CallToolResultFor[any]{
				Content: []mcp.Content{&mcp.TextContent{Text: "Error: " + err.Error()}},
				IsError: true,
			}, nil
		}

		return &mcp.CallToolResultFor[any]{
			Content: []mcp.Content{&mcp.TextContent{Text: content}},
		}, nil
	}

	tool := &mcp.Tool{
		Name:        "LS",
		Description: prompts.LSToolDoc,
	}

	return &tools.ServerTool{
		Tool: tool,
		RegisterFunc: func(server *mcp.Server) {
			mcp.AddTool(server, tool, handler)
		},
	}
}

// GlobArgs represents the arguments for the Glob tool.
type GlobArgs struct {
	Pattern string  `json:"pattern"`
	Path    *string `json:"path,omitempty"`
}

// CreateGlobTool creates the Glob tool using MCP SDK patterns.
func CreateGlobTool(ctx *tools.Context) *tools.ServerTool {
	handler := func(ctxReq context.Context, session *mcp.ServerSession, params *mcp.CallToolParamsFor[GlobArgs]) (*mcp.CallToolResultFor[any], error) {
		args := params.Arguments

		// Determine search path
		searchPath := "."
		if args.Path != nil && *args.Path != "" {
			searchPath = *args.Path
		}

		// Get absolute search path
		var absSearchPath string
		var err error
		if filepath.IsAbs(searchPath) {
			absSearchPath = searchPath
		} else {
			// Convert relative path to absolute
			cwd, err := os.Getwd()
			if err != nil {
				return &mcp.CallToolResultFor[any]{
					Content: []mcp.Content{&mcp.TextContent{Text: "Error: Failed to get current working directory: " + err.Error()}},
					IsError: true,
				}, nil
			}
			absSearchPath = filepath.Join(cwd, searchPath)
		}

		// Sanitize path
		sanitizedPath, err := ctx.Validator.SanitizePath(absSearchPath)
		if err != nil {
			return &mcp.CallToolResultFor[any]{
				Content: []mcp.Content{&mcp.TextContent{Text: "Error: Invalid search path: " + err.Error()}},
				IsError: true,
			}, nil
		}

		// Validate path
		if err := ctx.Validator.ValidatePath(sanitizedPath); err != nil {
			return &mcp.CallToolResultFor[any]{
				Content: []mcp.Content{&mcp.TextContent{Text: "Error: Path validation failed: " + err.Error()}},
				IsError: true,
			}, nil
		}

		// Validate pattern
		if args.Pattern == "" {
			return &mcp.CallToolResultFor[any]{
				Content: []mcp.Content{&mcp.TextContent{Text: "Error: Pattern cannot be empty"}},
				IsError: true,
			}, nil
		}

		// Perform glob matching using find command
		content, err := globFilesWithFind(sanitizedPath, args.Pattern)
		if err != nil {
			return &mcp.CallToolResultFor[any]{
				Content: []mcp.Content{&mcp.TextContent{Text: "Error: " + err.Error()}},
				IsError: true,
			}, nil
		}

		return &mcp.CallToolResultFor[any]{
			Content: []mcp.Content{&mcp.TextContent{Text: content}},
		}, nil
	}

	tool := &mcp.Tool{
		Name:        "Glob",
		Description: prompts.GlobToolDoc,
	}

	return &tools.ServerTool{
		Tool: tool,
		RegisterFunc: func(server *mcp.Server) {
			mcp.AddTool(server, tool, handler)
		},
	}
}

// GrepArgs represents the arguments for the Grep tool.
type GrepArgs struct {
	Pattern string  `json:"pattern"`
	Path    *string `json:"path,omitempty"`
	Include *string `json:"include,omitempty"`
}

// CreateGrepTool creates the Grep tool using MCP SDK patterns.
func CreateGrepTool(ctx *tools.Context) *tools.ServerTool {
	handler := func(ctxReq context.Context, session *mcp.ServerSession, params *mcp.CallToolParamsFor[GrepArgs]) (*mcp.CallToolResultFor[any], error) {
		args := params.Arguments

		// Determine search path
		searchPath := "."
		if args.Path != nil && *args.Path != "" {
			searchPath = *args.Path
		}

		// Get absolute search path
		var absSearchPath string
		var err error
		if filepath.IsAbs(searchPath) {
			absSearchPath = searchPath
		} else {
			// Convert relative path to absolute
			cwd, err := os.Getwd()
			if err != nil {
				return &mcp.CallToolResultFor[any]{
					Content: []mcp.Content{&mcp.TextContent{Text: "Error: Failed to get current working directory: " + err.Error()}},
					IsError: true,
				}, nil
			}
			absSearchPath = filepath.Join(cwd, searchPath)
		}

		// Sanitize path
		sanitizedPath, err := ctx.Validator.SanitizePath(absSearchPath)
		if err != nil {
			return &mcp.CallToolResultFor[any]{
				Content: []mcp.Content{&mcp.TextContent{Text: "Error: Invalid search path: " + err.Error()}},
				IsError: true,
			}, nil
		}

		// Validate path
		if err := ctx.Validator.ValidatePath(sanitizedPath); err != nil {
			return &mcp.CallToolResultFor[any]{
				Content: []mcp.Content{&mcp.TextContent{Text: "Error: Path validation failed: " + err.Error()}},
				IsError: true,
			}, nil
		}

		// Validate pattern
		if args.Pattern == "" {
			return &mcp.CallToolResultFor[any]{
				Content: []mcp.Content{&mcp.TextContent{Text: "Error: Pattern cannot be empty"}},
				IsError: true,
			}, nil
		}

		// Validate regex pattern by attempting to compile it
		if _, err := regexp.Compile(args.Pattern); err != nil {
			return &mcp.CallToolResultFor[any]{
				Content: []mcp.Content{&mcp.TextContent{Text: "Error: Invalid regular expression: " + err.Error()}},
				IsError: true,
			}, nil
		}

		// Perform content search using ripgrep
		content, err := grepFilesWithRipgrep(sanitizedPath, args.Pattern, args.Include)
		if err != nil {
			return &mcp.CallToolResultFor[any]{
				Content: []mcp.Content{&mcp.TextContent{Text: "Error: " + err.Error()}},
				IsError: true,
			}, nil
		}

		return &mcp.CallToolResultFor[any]{
			Content: []mcp.Content{&mcp.TextContent{Text: content}},
		}, nil
	}

	tool := &mcp.Tool{
		Name:        "Grep",
		Description: prompts.GrepToolDoc,
	}

	return &tools.ServerTool{
		Tool: tool,
		RegisterFunc: func(server *mcp.Server) {
			mcp.AddTool(server, tool, handler)
		},
	}
}

// Helper functions for file operations

func readFileContent(filePath string, offset *int, limit *int) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer func() {
		_ = file.Close()
	}()

	stat, err := file.Stat()
	if err != nil {
		return "", fmt.Errorf("failed to get file info: %w", err)
	}

	if stat.IsDir() {
		return "", fmt.Errorf("path is a directory, not a file")
	}

	startOffset := 0
	if offset != nil {
		startOffset = *offset
	}

	maxLines := 2000
	if limit != nil {
		maxLines = *limit
	}

	scanner := bufio.NewScanner(file)
	var lines []string
	lineNumber := 1
	currentOffset := 0
	maxLineLength := 2000

	for scanner.Scan() {
		if currentOffset >= startOffset {
			if len(lines) >= maxLines {
				break
			}

			line := scanner.Text()
			if len(line) > maxLineLength {
				line = line[:maxLineLength] + "... (truncated)"
			}

			formattedLine := fmt.Sprintf("%5d→%s", lineNumber, line)
			lines = append(lines, formattedLine)
		}
		lineNumber++
		currentOffset++
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error reading file: %w", err)
	}

	content := strings.Join(lines, "\n")
	if len(lines) == 0 && stat.Size() == 0 {
		content = "<system-reminder>\nWARNING: This file exists but has empty contents.\n</system-reminder>"
	}

	return content, nil
}

func writeFileContent(filePath, content string) (int, error) {
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return 0, fmt.Errorf("failed to create directory: %w", err)
	}

	file, err := os.Create(filePath)
	if err != nil {
		return 0, fmt.Errorf("failed to create file: %w", err)
	}
	defer func() {
		_ = file.Close()
	}()

	bytesWritten, err := file.WriteString(content)
	if err != nil {
		return 0, fmt.Errorf("failed to write content: %w", err)
	}

	if err := file.Sync(); err != nil {
		return 0, fmt.Errorf("failed to sync file: %w", err)
	}

	return bytesWritten, nil
}

func listDirectoryWithLS(dirPath string, ignorePatterns []string) (string, error) {
	stat, err := os.Stat(dirPath)
	if err != nil {
		return "", fmt.Errorf("failed to stat path: %w", err)
	}

	if !stat.IsDir() {
		return "", fmt.Errorf("path is not a directory")
	}

	// Check if ls command is available
	lsPath, err := FindBinary("ls")
	if err != nil {
		return "", fmt.Errorf("ls command not found: %w", err)
	}

	// Create command executor with 10 second timeout
	executor := NewCommandExecutor(10 * time.Second)

	// Build ls command arguments
	args := []string{
		"-1", // One entry per line
		"-A", // Show hidden files but not . and ..
		"-F", // Add indicators to show file types
		dirPath,
	}

	// Validate command before execution
	if err := executor.ValidateCommand("ls", args); err != nil {
		return "", fmt.Errorf("command validation failed: %w", err)
	}

	// Execute ls command
	result, err := executor.Execute(context.Background(), lsPath, args...)
	if err != nil {
		return "", fmt.Errorf("failed to execute ls: %w", err)
	}

	// Handle ls exit codes
	if result.ExitCode != 0 {
		return "", fmt.Errorf("ls command failed with exit code %d: %s", result.ExitCode, result.Stderr)
	}

	// Parse output
	if strings.TrimSpace(result.Stdout) == "" {
		return fmt.Sprintf("- %s/\n  (empty directory)", dirPath), nil
	}

	lines := strings.Split(strings.TrimSpace(result.Stdout), "\n")
	var output strings.Builder
	output.WriteString(fmt.Sprintf("- %s/\n", dirPath))

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Extract filename and type indicator
		name := line
		isDir := strings.HasSuffix(line, "/")
		if isDir {
			name = strings.TrimSuffix(line, "/")
		}

		// Apply ignore patterns
		if shouldIgnoreFile(name, ignorePatterns) {
			continue
		}

		// Format output
		if isDir {
			output.WriteString(fmt.Sprintf("  - %s/\n", name))
		} else {
			// Remove file type indicators that ls -F adds
			name = strings.TrimSuffix(name, "*") // Executable
			name = strings.TrimSuffix(name, "@") // Symlink
			name = strings.TrimSuffix(name, "|") // FIFO
			name = strings.TrimSuffix(name, "=") // Socket
			output.WriteString(fmt.Sprintf("  - %s\n", name))
		}
	}

	return strings.TrimSuffix(output.String(), "\n"), nil
}

// EditArgs represents the arguments for the Edit tool.
type EditArgs struct {
	FilePath   string `json:"file_path"`
	OldString  string `json:"old_string"`
	NewString  string `json:"new_string"`
	ReplaceAll *bool  `json:"replace_all,omitempty"`
}

// CreateEditTool creates the Edit tool using MCP SDK patterns.
func CreateEditTool(ctx *tools.Context) *tools.ServerTool {
	handler := func(ctxReq context.Context, session *mcp.ServerSession, params *mcp.CallToolParamsFor[EditArgs]) (*mcp.CallToolResultFor[any], error) {
		args := params.Arguments

		// Sanitize path
		sanitizedPath, err := ctx.Validator.SanitizePath(args.FilePath)
		if err != nil {
			return &mcp.CallToolResultFor[any]{
				Content: []mcp.Content{&mcp.TextContent{Text: "Error: Invalid file path: " + err.Error()}},
				IsError: true,
			}, nil
		}

		// Validate path
		if err := ctx.Validator.ValidatePath(sanitizedPath); err != nil {
			return &mcp.CallToolResultFor[any]{
				Content: []mcp.Content{&mcp.TextContent{Text: "Error: Path validation failed: " + err.Error()}},
				IsError: true,
			}, nil
		}

		// Validate arguments
		if args.OldString == args.NewString {
			return &mcp.CallToolResultFor[any]{
				Content: []mcp.Content{&mcp.TextContent{Text: "Error: old_string and new_string must be different"}},
				IsError: true,
			}, nil
		}

		if args.OldString == "" {
			return &mcp.CallToolResultFor[any]{
				Content: []mcp.Content{&mcp.TextContent{Text: "Error: old_string cannot be empty"}},
				IsError: true,
			}, nil
		}

		// Perform the edit
		result, err := editFileContent(sanitizedPath, args.OldString, args.NewString, args.ReplaceAll)
		if err != nil {
			return &mcp.CallToolResultFor[any]{
				Content: []mcp.Content{&mcp.TextContent{Text: "Error: " + err.Error()}},
				IsError: true,
			}, nil
		}

		return &mcp.CallToolResultFor[any]{
			Content: []mcp.Content{&mcp.TextContent{Text: result}},
		}, nil
	}

	tool := &mcp.Tool{
		Name:        "Edit",
		Description: prompts.EditToolDoc,
	}

	return &tools.ServerTool{
		Tool: tool,
		RegisterFunc: func(server *mcp.Server) {
			mcp.AddTool(server, tool, handler)
		},
	}
}

func shouldIgnoreFile(filename string, ignorePatterns []string) bool {
	for _, pattern := range ignorePatterns {
		if matched, err := filepath.Match(pattern, filename); err == nil && matched {
			return true
		}
	}
	return false
}

// MultiEditOperation represents a single edit operation in a MultiEdit.
type MultiEditOperation struct {
	OldString  string `json:"old_string"`
	NewString  string `json:"new_string"`
	ReplaceAll *bool  `json:"replace_all,omitempty"`
}

// MultiEditArgs represents the arguments for the MultiEdit tool.
type MultiEditArgs struct {
	FilePath string               `json:"file_path"`
	Edits    []MultiEditOperation `json:"edits"`
}

// CreateMultiEditTool creates the MultiEdit tool using MCP SDK patterns.
func CreateMultiEditTool(ctx *tools.Context) *tools.ServerTool {
	handler := func(ctxReq context.Context, session *mcp.ServerSession, params *mcp.CallToolParamsFor[MultiEditArgs]) (*mcp.CallToolResultFor[any], error) {
		args := params.Arguments

		// Sanitize path
		sanitizedPath, err := ctx.Validator.SanitizePath(args.FilePath)
		if err != nil {
			return &mcp.CallToolResultFor[any]{
				Content: []mcp.Content{&mcp.TextContent{Text: "Error: Invalid file path: " + err.Error()}},
				IsError: true,
			}, nil
		}

		// Validate path
		if err := ctx.Validator.ValidatePath(sanitizedPath); err != nil {
			return &mcp.CallToolResultFor[any]{
				Content: []mcp.Content{&mcp.TextContent{Text: "Error: Path validation failed: " + err.Error()}},
				IsError: true,
			}, nil
		}

		// Validate edits array
		if len(args.Edits) == 0 {
			return &mcp.CallToolResultFor[any]{
				Content: []mcp.Content{&mcp.TextContent{Text: "Error: edits array cannot be empty"}},
				IsError: true,
			}, nil
		}

		// Validate each edit operation
		for i, edit := range args.Edits {
			if edit.OldString == edit.NewString {
				return &mcp.CallToolResultFor[any]{
					Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Error: edit %d: old_string and new_string must be different", i+1)}},
					IsError: true,
				}, nil
			}

			if edit.OldString == "" {
				return &mcp.CallToolResultFor[any]{
					Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Error: edit %d: old_string cannot be empty", i+1)}},
					IsError: true,
				}, nil
			}
		}

		// Perform the multi-edit operation
		result, err := performMultiEdit(sanitizedPath, args.Edits)
		if err != nil {
			return &mcp.CallToolResultFor[any]{
				Content: []mcp.Content{&mcp.TextContent{Text: "Error: " + err.Error()}},
				IsError: true,
			}, nil
		}

		return &mcp.CallToolResultFor[any]{
			Content: []mcp.Content{&mcp.TextContent{Text: result}},
		}, nil
	}

	tool := &mcp.Tool{
		Name:        "MultiEdit",
		Description: prompts.MultiEditToolDoc,
	}

	return &tools.ServerTool{
		Tool: tool,
		RegisterFunc: func(server *mcp.Server) {
			mcp.AddTool(server, tool, handler)
		},
	}
}

// performMultiEdit performs multiple edits atomically on a file.
func performMultiEdit(filePath string, edits []MultiEditOperation) (string, error) {
	// Check if file exists
	stat, err := os.Stat(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to stat file: %w", err)
	}

	if stat.IsDir() {
		return "", fmt.Errorf("path is a directory, not a file")
	}

	// Read the original file content
	originalContent, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	// Create backup of original file
	backupPath := filePath + ".backup"
	if err := os.WriteFile(backupPath, originalContent, stat.Mode()); err != nil {
		return "", fmt.Errorf("failed to create backup file: %w", err)
	}

	// Apply edits sequentially
	currentContent := string(originalContent)
	totalReplacements := 0

	for i, edit := range edits {
		// Determine if we should replace all occurrences
		shouldReplaceAll := edit.ReplaceAll != nil && *edit.ReplaceAll

		var modifiedContent string
		var replacementCount int

		if shouldReplaceAll {
			// Replace all occurrences
			modifiedContent = strings.ReplaceAll(currentContent, edit.OldString, edit.NewString)
			replacementCount = strings.Count(currentContent, edit.OldString)
		} else {
			// Check if the old string appears exactly once in current content
			occurrenceCount := strings.Count(currentContent, edit.OldString)
			if occurrenceCount == 0 {
				// Restore backup and return error
				_ = os.Rename(backupPath, filePath)
				return "", fmt.Errorf("edit %d: old_string not found in file", i+1)
			}
			if occurrenceCount > 1 {
				// Restore backup and return error
				_ = os.Rename(backupPath, filePath)
				return "", fmt.Errorf("edit %d: old_string appears %d times in file - use replace_all=true or provide more context to make it unique", i+1, occurrenceCount)
			}

			// Replace the single occurrence
			modifiedContent = strings.Replace(currentContent, edit.OldString, edit.NewString, 1)
			replacementCount = 1
		}

		// Check if any replacements were made
		if replacementCount == 0 {
			// Restore backup and return error
			_ = os.Rename(backupPath, filePath)
			return "", fmt.Errorf("edit %d: old_string not found in file", i+1)
		}

		// Update current content for next iteration
		currentContent = modifiedContent
		totalReplacements += replacementCount
	}

	// Write the final modified content back to the file
	if err := os.WriteFile(filePath, []byte(currentContent), stat.Mode()); err != nil {
		// Try to restore from backup if write fails
		if restoreErr := os.Rename(backupPath, filePath); restoreErr != nil {
			return "", fmt.Errorf("failed to write file and failed to restore backup: write error: %w, restore error: %v", err, restoreErr)
		}
		return "", fmt.Errorf("failed to write file (backup restored): %w", err)
	}

	// Clean up backup file on success
	_ = os.Remove(backupPath)

	// Return success message
	return fmt.Sprintf("Successfully applied %d edits with %d total replacements in %s", len(edits), totalReplacements, filePath), nil
}

// editFileContent performs string replacement on a file.
func editFileContent(filePath, oldString, newString string, replaceAll *bool) (string, error) {
	// Check if file exists
	stat, err := os.Stat(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to stat file: %w", err)
	}

	if stat.IsDir() {
		return "", fmt.Errorf("path is a directory, not a file")
	}

	// Read the file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	originalContent := string(content)

	// Determine if we should replace all occurrences
	shouldReplaceAll := replaceAll != nil && *replaceAll

	var modifiedContent string
	var replacementCount int

	if shouldReplaceAll {
		// Replace all occurrences
		modifiedContent = strings.ReplaceAll(originalContent, oldString, newString)
		replacementCount = strings.Count(originalContent, oldString)
	} else {
		// Check if the old string appears exactly once
		occurrenceCount := strings.Count(originalContent, oldString)
		if occurrenceCount == 0 {
			return "", fmt.Errorf("old_string not found in file")
		}
		if occurrenceCount > 1 {
			return "", fmt.Errorf("old_string appears %d times in file - use replace_all=true or provide more context to make it unique", occurrenceCount)
		}

		// Replace the single occurrence
		modifiedContent = strings.Replace(originalContent, oldString, newString, 1)
		replacementCount = 1
	}

	// Check if any replacements were made
	if replacementCount == 0 {
		return "", fmt.Errorf("old_string not found in file")
	}

	// Create backup of original file
	backupPath := filePath + ".backup"
	if err := os.WriteFile(backupPath, content, stat.Mode()); err != nil {
		return "", fmt.Errorf("failed to create backup file: %w", err)
	}

	// Write the modified content back to the file
	if err := os.WriteFile(filePath, []byte(modifiedContent), stat.Mode()); err != nil {
		// Try to restore from backup if write fails
		if restoreErr := os.Rename(backupPath, filePath); restoreErr != nil {
			return "", fmt.Errorf("failed to write file and failed to restore backup: write error: %w, restore error: %v", err, restoreErr)
		}
		return "", fmt.Errorf("failed to write file (backup restored): %w", err)
	}

	// Clean up backup file on success
	_ = os.Remove(backupPath)

	// Return success message
	if shouldReplaceAll {
		return fmt.Sprintf("Successfully replaced %d occurrences in %s", replacementCount, filePath), nil
	} else {
		return fmt.Sprintf("Successfully replaced 1 occurrence in %s", filePath), nil
	}
}

// FileMatchInfo represents a file with its modification time for sorting.
type FileMatchInfo struct {
	Path    string
	ModTime time.Time
}

// globFilesWithFind performs glob pattern matching using find command and returns sorted results.
func globFilesWithFind(searchPath, pattern string) (string, error) {
	// Ensure search path is a directory
	stat, err := os.Stat(searchPath)
	if err != nil {
		return "", fmt.Errorf("failed to stat search path: %w", err)
	}

	if !stat.IsDir() {
		return "", fmt.Errorf("search path is not a directory")
	}

	// Check if find command is available
	findPath, err := FindBinary("find")
	if err != nil {
		return "", fmt.Errorf("find command not found: %w", err)
	}

	// Create command executor with 30 second timeout
	executor := NewCommandExecutor(30 * time.Second)

	// Convert glob pattern to find-compatible pattern
	findPattern := convertGlobToFindPattern(pattern)

	// Build find command arguments
	args := []string{
		searchPath,
		"-type", "f", // Only files, not directories
		"-name", findPattern,
	}

	// Handle recursive patterns
	if strings.Contains(pattern, "**/") {
		// For recursive patterns, use -path instead of -name
		args = []string{
			searchPath,
			"-type", "f",
			"-path", "*/" + strings.TrimPrefix(pattern, "**/"),
		}
	}

	// Validate command before execution
	if err := executor.ValidateCommand("find", args); err != nil {
		return "", fmt.Errorf("command validation failed: %w", err)
	}

	// Execute find command
	result, err := executor.Execute(context.Background(), findPath, args...)
	if err != nil {
		return "", fmt.Errorf("failed to execute find: %w", err)
	}

	// Handle find exit codes
	if result.ExitCode != 0 {
		return "", fmt.Errorf("find command failed with exit code %d: %s", result.ExitCode, result.Stderr)
	}

	if strings.TrimSpace(result.Stdout) == "" {
		return fmt.Sprintf("No files found matching pattern '%s' in directory '%s'", pattern, searchPath), nil
	}

	// Parse and sort results
	lines := strings.Split(strings.TrimSpace(result.Stdout), "\n")
	matches := make([]FileMatchInfo, 0, len(lines))

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Get file modification time for sorting
		if stat, err := os.Stat(line); err == nil {
			matches = append(matches, FileMatchInfo{
				Path:    line,
				ModTime: stat.ModTime(),
			})
		} else {
			// If we can't stat the file, add it anyway without mod time
			matches = append(matches, FileMatchInfo{
				Path:    line,
				ModTime: time.Time{}, // Zero time will sort last
			})
		}
	}

	// Sort by modification time (most recent first)
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].ModTime.After(matches[j].ModTime)
	})

	// Format output
	var output strings.Builder
	output.WriteString(fmt.Sprintf("Found %d file(s) matching pattern '%s' in directory '%s':\n", len(matches), pattern, searchPath))

	for _, match := range matches {
		output.WriteString(match.Path + "\n")
	}

	return strings.TrimSuffix(output.String(), "\n"), nil
}

// convertGlobToFindPattern converts a glob pattern to a find-compatible pattern.
func convertGlobToFindPattern(pattern string) string {
	// Handle recursive patterns
	if strings.HasPrefix(pattern, "**/") {
		// Remove the **/ prefix for find -name usage
		return pattern[3:]
	}

	// For simple patterns, use as-is
	return pattern
}

// matchGlobPattern matches a file path against a glob pattern.
// Supports ** for recursive directory matching and standard glob patterns.
func matchGlobPattern(pattern, path string) (bool, error) {
	// Handle ** patterns for recursive matching
	if strings.Contains(pattern, "**") {
		// Split pattern on ** to handle recursive matching
		parts := strings.Split(pattern, "**")

		if len(parts) == 1 {
			// No ** in pattern, use standard matching
			return filepath.Match(pattern, path)
		}

		// For patterns with **, we need custom logic
		return matchRecursivePattern(pattern, path)
	}

	// Use standard filepath.Match for non-recursive patterns
	return filepath.Match(pattern, path)
}

// matchRecursivePattern handles patterns with ** for recursive directory matching.
func matchRecursivePattern(pattern, path string) (bool, error) {
	// Convert pattern to a regular expression approach
	// This is a simplified implementation for common cases

	// For simple matching, check prefix and suffix
	if strings.HasPrefix(pattern, "**/") {
		// Pattern like "**/*.go"
		suffix := pattern[3:] // Remove "**/"
		if strings.Contains(suffix, "/") {
			// Complex pattern, fall back to basic matching
			return strings.HasSuffix(path, suffix[strings.LastIndex(suffix, "/"):]), nil
		}
		// Simple suffix pattern like "**/*.go"
		return filepath.Match(suffix, filepath.Base(path))
	}

	if strings.HasSuffix(pattern, "/**") {
		// Pattern like "src/**"
		prefix := pattern[:len(pattern)-3] // Remove "/**"
		return strings.HasPrefix(path, prefix+"/") || path == prefix, nil
	}

	if strings.Contains(pattern, "**/") {
		// Pattern like "src/**/*.go"
		parts := strings.Split(pattern, "**/")
		if len(parts) == 2 {
			prefix := parts[0]
			suffix := parts[1]

			// Check if path starts with prefix and matches suffix pattern
			if prefix != "" && !strings.HasPrefix(path, prefix) {
				return false, nil
			}

			// Find the part after the prefix
			remainingPath := path
			if prefix != "" {
				if len(path) <= len(prefix) {
					return false, nil
				}
				remainingPath = path[len(prefix):]
				// Remove leading slash if present
				remainingPath = strings.TrimPrefix(remainingPath, "/")
			}

			// Check suffix pattern against remaining path or just the filename
			if strings.Contains(suffix, "/") {
				return filepath.Match(suffix, remainingPath)
			}
			return filepath.Match(suffix, filepath.Base(remainingPath))
		}
	}

	// Fallback to basic pattern matching
	return filepath.Match(pattern, path)
}

// grepFilesWithRipgrep performs content search using ripgrep command and returns sorted results.
func grepFilesWithRipgrep(searchPath, pattern string, includePattern *string) (string, error) {
	// Ensure search path is a directory
	stat, err := os.Stat(searchPath)
	if err != nil {
		return "", fmt.Errorf("failed to stat search path: %w", err)
	}

	if !stat.IsDir() {
		return "", fmt.Errorf("search path is not a directory")
	}

	// Check if ripgrep is available
	rgPath, err := FindBinary("rg")
	if err != nil {
		return "", fmt.Errorf("ripgrep (rg) not found: %w - please install ripgrep for optimal performance", err)
	}

	// Create command executor with 30 second timeout
	executor := NewCommandExecutor(30 * time.Second)

	// Build ripgrep arguments
	args := []string{
		"--files-with-matches", // Only show file names, not matches
		"--no-heading",         // Don't group matches by file
		"--no-line-number",     // Don't show line numbers
		"--color=never",        // No color output
		"--hidden",             // Search hidden files
		"--follow",             // Follow symlinks
		"--case-sensitive",     // Case sensitive by default
	}

	// Add include pattern if specified
	if includePattern != nil && *includePattern != "" {
		// Convert include pattern to ripgrep glob
		globPattern := convertIncludePatternToGlob(*includePattern)
		args = append(args, "--glob", globPattern)
	}

	// Add the search pattern and path
	args = append(args, pattern, searchPath)

	// Validate command before execution
	if err := executor.ValidateCommand("rg", args); err != nil {
		return "", fmt.Errorf("command validation failed: %w", err)
	}

	// Execute ripgrep
	result, err := executor.Execute(context.Background(), rgPath, args...)
	if err != nil {
		return "", fmt.Errorf("failed to execute ripgrep: %w", err)
	}

	// Handle ripgrep exit codes
	// Exit code 0: matches found
	// Exit code 1: no matches found
	// Exit code 2: error occurred
	if result.ExitCode == 2 {
		return "", fmt.Errorf("ripgrep error: %s", result.Stderr)
	}

	if result.ExitCode == 1 || strings.TrimSpace(result.Stdout) == "" {
		return fmt.Sprintf("No files found containing pattern '%s' in directory '%s'", pattern, searchPath), nil
	}

	// Parse and sort results
	lines := strings.Split(strings.TrimSpace(result.Stdout), "\n")
	matches := make([]FileMatchInfo, 0, len(lines))

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Get file modification time for sorting
		if stat, err := os.Stat(line); err == nil {
			matches = append(matches, FileMatchInfo{
				Path:    line,
				ModTime: stat.ModTime(),
			})
		} else {
			// If we can't stat the file, add it anyway without mod time
			matches = append(matches, FileMatchInfo{
				Path:    line,
				ModTime: time.Time{}, // Zero time will sort last
			})
		}
	}

	// Sort by modification time (most recent first)
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].ModTime.After(matches[j].ModTime)
	})

	// Format output
	var output strings.Builder
	output.WriteString(fmt.Sprintf("Found %d file(s) containing pattern '%s' in directory '%s':\n", len(matches), pattern, searchPath))

	for _, match := range matches {
		output.WriteString(match.Path + "\n")
	}

	return strings.TrimSuffix(output.String(), "\n"), nil
}

// convertIncludePatternToGlob converts a Claude Code include pattern to a ripgrep glob pattern.
func convertIncludePatternToGlob(includePattern string) string {
	// Handle brace expansion patterns like "*.{ts,tsx}"
	if strings.Contains(includePattern, "{") && strings.Contains(includePattern, "}") {
		// ripgrep supports brace expansion natively
		return includePattern
	}

	// For simple patterns like "*.js", use as-is
	return includePattern
}

// searchFileContent searches for regex pattern in file content.
func searchFileContent(filePath string, regex *regexp.Regexp) (bool, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return false, err
	}
	defer func() {
		_ = file.Close()
	}()

	// Check if file is binary by reading first 512 bytes
	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil && err.Error() != "EOF" {
		return false, err
	}

	// Simple binary file detection
	if isBinaryContent(buffer[:n]) {
		return false, nil
	}

	// Reset file pointer to beginning
	if _, err := file.Seek(0, 0); err != nil {
		return false, err
	}

	// Search file content line by line
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if regex.MatchString(line) {
			return true, nil
		}
	}

	if err := scanner.Err(); err != nil {
		return false, err
	}

	return false, nil
}

// isBinaryContent checks if content appears to be binary (non-text).
func isBinaryContent(data []byte) bool {
	// Simple heuristic: if we find null bytes or high percentage of non-printable characters
	nullBytes := 0
	nonPrintable := 0

	for _, b := range data {
		if b == 0 {
			nullBytes++
		}
		// Check for non-printable characters (excluding common whitespace)
		if b < 32 && b != 9 && b != 10 && b != 13 {
			nonPrintable++
		}
	}

	// If more than 1% null bytes or more than 30% non-printable, consider binary
	if len(data) > 0 && (float64(nullBytes)/float64(len(data)) > 0.01 || float64(nonPrintable)/float64(len(data)) > 0.30) {
		return true
	}

	return false
}

// matchIncludePattern matches a filename against an include pattern.
// Supports patterns like "*.js", "*.{ts,tsx}", etc.
func matchIncludePattern(pattern, fileName string) (bool, error) {
	// Handle brace expansion patterns like "*.{ts,tsx}"
	if strings.Contains(pattern, "{") && strings.Contains(pattern, "}") {
		return matchBracePattern(pattern, fileName)
	}

	// Use standard filepath.Match for simple patterns
	return filepath.Match(pattern, fileName)
}

// matchBracePattern handles brace expansion patterns like "*.{ts,tsx}".
func matchBracePattern(pattern, fileName string) (bool, error) {
	// Find brace expression
	start := strings.Index(pattern, "{")
	end := strings.Index(pattern, "}")

	if start == -1 || end == -1 || end <= start {
		// No valid brace expression, fall back to standard matching
		return filepath.Match(pattern, fileName)
	}

	// Extract parts
	prefix := pattern[:start]
	suffix := pattern[end+1:]
	braceContent := pattern[start+1 : end]

	// Split brace content by comma
	alternatives := strings.Split(braceContent, ",")

	// Test each alternative
	for _, alt := range alternatives {
		testPattern := prefix + strings.TrimSpace(alt) + suffix
		matched, err := filepath.Match(testPattern, fileName)
		if err != nil {
			continue
		}
		if matched {
			return true, nil
		}
	}

	return false, nil
}
