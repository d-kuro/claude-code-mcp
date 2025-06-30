// Package file provides file operation tools using the MCP SDK patterns.
package file

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/d-kuro/claude-code-mcp/internal/prompts"
	"github.com/d-kuro/claude-code-mcp/internal/tools"
)

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

// writeFileContent writes content to a file, creating directories as needed.
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
