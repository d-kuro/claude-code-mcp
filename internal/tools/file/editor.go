// Package file provides file operation tools using the MCP SDK patterns.
package file

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/d-kuro/claude-code-mcp/internal/prompts"
	"github.com/d-kuro/claude-code-mcp/internal/tools"
)

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

// editFileContent performs string replacement on a file.
func editFileContent(filePath, oldString, newString string, replaceAll *bool) (string, error) {
	stat, err := os.Stat(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to stat file: %w", err)
	}

	if stat.IsDir() {
		return "", fmt.Errorf("path is a directory, not a file")
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	originalContent := string(content)
	shouldReplaceAll := replaceAll != nil && *replaceAll

	var modifiedContent string
	var replacementCount int

	if shouldReplaceAll {
		modifiedContent = strings.ReplaceAll(originalContent, oldString, newString)
		replacementCount = strings.Count(originalContent, oldString)
	} else {
		occurrenceCount := strings.Count(originalContent, oldString)
		if occurrenceCount == 0 {
			return "", fmt.Errorf("old_string not found in file")
		}
		if occurrenceCount > 1 {
			return "", fmt.Errorf("old_string appears %d times in file - use replace_all=true or provide more context to make it unique", occurrenceCount)
		}

		modifiedContent = strings.Replace(originalContent, oldString, newString, 1)
		replacementCount = 1
	}

	if replacementCount == 0 {
		return "", fmt.Errorf("old_string not found in file")
	}

	backupPath := filePath + ".backup"
	if err := os.WriteFile(backupPath, content, stat.Mode()); err != nil {
		return "", fmt.Errorf("failed to create backup file: %w", err)
	}

	if err := os.WriteFile(filePath, []byte(modifiedContent), stat.Mode()); err != nil {
		if restoreErr := os.Rename(backupPath, filePath); restoreErr != nil {
			return "", fmt.Errorf("failed to write file and failed to restore backup: write error: %w, restore error: %v", err, restoreErr)
		}
		return "", fmt.Errorf("failed to write file (backup restored): %w", err)
	}

	_ = os.Remove(backupPath)

	if shouldReplaceAll {
		return fmt.Sprintf("Successfully replaced %d occurrences in %s", replacementCount, filePath), nil
	}
	return fmt.Sprintf("Successfully replaced 1 occurrence in %s", filePath), nil
}
