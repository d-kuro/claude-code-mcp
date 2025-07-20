// Package file provides file operation tools using the MCP SDK patterns.
package file

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/d-kuro/claude-code-mcp/internal/prompts"
	"github.com/d-kuro/claude-code-mcp/internal/tools"
)

// GlobArgs represents the arguments for the Glob tool.
type GlobArgs struct {
	Pattern string  `json:"pattern"`
	Path    *string `json:"path,omitempty"`
}

// CreateGlobTool creates the Glob tool using MCP SDK patterns.
func CreateGlobTool(ctx *tools.Context) *tools.ServerTool {
	handler := func(ctxReq context.Context, session *mcp.ServerSession, params *mcp.CallToolParamsFor[GlobArgs]) (*mcp.CallToolResultFor[any], error) {
		args := params.Arguments

		searchPath := "."
		if args.Path != nil && *args.Path != "" {
			searchPath = *args.Path
		}

		var absSearchPath string
		var err error
		if filepath.IsAbs(searchPath) {
			absSearchPath = searchPath
		} else {
			cwd, err := os.Getwd()
			if err != nil {
				return &mcp.CallToolResultFor[any]{
					Content: []mcp.Content{&mcp.TextContent{Text: "Error: Failed to get current working directory: " + err.Error()}},
					IsError: true,
				}, nil
			}
			absSearchPath = filepath.Join(cwd, searchPath)
		}

		sanitizedPath, err := ctx.Validator.SanitizePath(absSearchPath)
		if err != nil {
			return &mcp.CallToolResultFor[any]{
				Content: []mcp.Content{&mcp.TextContent{Text: "Error: Invalid search path: " + err.Error()}},
				IsError: true,
			}, nil
		}

		if err := ctx.Validator.ValidatePath(sanitizedPath); err != nil {
			return &mcp.CallToolResultFor[any]{
				Content: []mcp.Content{&mcp.TextContent{Text: "Error: Path validation failed: " + err.Error()}},
				IsError: true,
			}, nil
		}

		if args.Pattern == "" {
			return &mcp.CallToolResultFor[any]{
				Content: []mcp.Content{&mcp.TextContent{Text: "Error: Pattern cannot be empty"}},
				IsError: true,
			}, nil
		}

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

// globFilesWithFind performs glob pattern matching using find command and returns sorted results.
func globFilesWithFind(searchPath, pattern string) (string, error) {
	stat, err := os.Stat(searchPath)
	if err != nil {
		return "", fmt.Errorf("failed to stat search path: %w", err)
	}

	if !stat.IsDir() {
		return "", fmt.Errorf("search path is not a directory")
	}

	findPath, err := FindBinary("find")
	if err != nil {
		return "", fmt.Errorf("find command not found: %w", err)
	}

	executor := NewCommandExecutor(30 * time.Second)
	findPattern := convertGlobToFindPattern(pattern)

	args := []string{
		searchPath,
		"-type", "f",
		"-name", findPattern,
	}

	if strings.Contains(pattern, "**/") {
		args = []string{
			searchPath,
			"-type", "f",
			"-path", "*/" + strings.TrimPrefix(pattern, "**/"),
		}
	}

	if err := executor.ValidateCommand("find", args); err != nil {
		return "", fmt.Errorf("command validation failed: %w", err)
	}

	result, err := executor.Execute(context.Background(), findPath, args...)
	if err != nil {
		return "", fmt.Errorf("failed to execute find: %w", err)
	}

	if result.ExitCode != 0 {
		return "", fmt.Errorf("find command failed with exit code %d: %s", result.ExitCode, result.Stderr)
	}

	if strings.TrimSpace(result.Stdout) == "" {
		return fmt.Sprintf("No files found matching pattern '%s' in directory '%s'", pattern, searchPath), nil
	}

	lines := strings.Split(strings.TrimSpace(result.Stdout), "\n")
	matches := make([]FileMatchInfo, 0, len(lines))

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if stat, err := os.Stat(line); err == nil {
			matches = append(matches, FileMatchInfo{
				Path:    line,
				ModTime: stat.ModTime(),
			})
		} else {
			matches = append(matches, FileMatchInfo{
				Path:    line,
				ModTime: time.Time{},
			})
		}
	}

	sort.Slice(matches, func(i, j int) bool {
		return matches[i].ModTime.After(matches[j].ModTime)
	})

	var output strings.Builder
	output.WriteString(fmt.Sprintf("Found %d file(s) matching pattern '%s' in directory '%s':\n", len(matches), pattern, searchPath))

	for _, match := range matches {
		output.WriteString(match.Path + "\n")
	}

	return strings.TrimSuffix(output.String(), "\n"), nil
}

// convertGlobToFindPattern converts a glob pattern to a find-compatible pattern.
func convertGlobToFindPattern(pattern string) string {
	if strings.HasPrefix(pattern, "**/") {
		return pattern[3:]
	}
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
