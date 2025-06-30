// Package file provides file operation tools using the MCP SDK patterns.
package file

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/d-kuro/claude-code-mcp/internal/prompts"
	"github.com/d-kuro/claude-code-mcp/internal/tools"
)

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

// listDirectoryWithLS lists directory contents using the ls command.
func listDirectoryWithLS(dirPath string, ignorePatterns []string) (string, error) {
	stat, err := os.Stat(dirPath)
	if err != nil {
		return "", fmt.Errorf("failed to stat path: %w", err)
	}

	if !stat.IsDir() {
		return "", fmt.Errorf("path is not a directory")
	}

	lsPath, err := FindBinary("ls")
	if err != nil {
		return "", fmt.Errorf("ls command not found: %w", err)
	}

	executor := NewCommandExecutor(10 * time.Second)

	args := []string{
		"-1", // One entry per line
		"-A", // Show hidden files but not . and ..
		"-F", // Add indicators to show file types
		dirPath,
	}

	if err := executor.ValidateCommand("ls", args); err != nil {
		return "", fmt.Errorf("command validation failed: %w", err)
	}

	result, err := executor.Execute(context.Background(), lsPath, args...)
	if err != nil {
		return "", fmt.Errorf("failed to execute ls: %w", err)
	}

	if result.ExitCode != 0 {
		return "", fmt.Errorf("ls command failed with exit code %d: %s", result.ExitCode, result.Stderr)
	}

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

		name := line
		isDir := strings.HasSuffix(line, "/")
		if isDir {
			name = strings.TrimSuffix(line, "/")
		}

		if shouldIgnoreFile(name, ignorePatterns) {
			continue
		}

		if isDir {
			output.WriteString(fmt.Sprintf("  - %s/\n", name))
		} else {
			name = strings.TrimSuffix(name, "*") // Executable
			name = strings.TrimSuffix(name, "@") // Symlink
			name = strings.TrimSuffix(name, "|") // FIFO
			name = strings.TrimSuffix(name, "=") // Socket
			output.WriteString(fmt.Sprintf("  - %s\n", name))
		}
	}

	return strings.TrimSuffix(output.String(), "\n"), nil
}

// shouldIgnoreFile checks if a filename matches any of the ignore patterns.
func shouldIgnoreFile(filename string, ignorePatterns []string) bool {
	for _, pattern := range ignorePatterns {
		if matched, err := filepath.Match(pattern, filename); err == nil && matched {
			return true
		}
	}
	return false
}
