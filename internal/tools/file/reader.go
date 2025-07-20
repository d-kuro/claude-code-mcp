// Package file provides file operation tools using the MCP SDK patterns.
package file

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

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

// readFileContent reads file content with support for offset and limit.
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
