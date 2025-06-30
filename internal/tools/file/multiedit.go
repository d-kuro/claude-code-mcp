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

		if len(args.Edits) == 0 {
			return &mcp.CallToolResultFor[any]{
				Content: []mcp.Content{&mcp.TextContent{Text: "Error: edits array cannot be empty"}},
				IsError: true,
			}, nil
		}

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
	stat, err := os.Stat(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to stat file: %w", err)
	}

	if stat.IsDir() {
		return "", fmt.Errorf("path is a directory, not a file")
	}

	originalContent, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	backupPath := filePath + ".backup"
	if err := os.WriteFile(backupPath, originalContent, stat.Mode()); err != nil {
		return "", fmt.Errorf("failed to create backup file: %w", err)
	}

	currentContent := string(originalContent)
	totalReplacements := 0

	for i, edit := range edits {
		shouldReplaceAll := edit.ReplaceAll != nil && *edit.ReplaceAll

		var modifiedContent string
		var replacementCount int

		if shouldReplaceAll {
			modifiedContent = strings.ReplaceAll(currentContent, edit.OldString, edit.NewString)
			replacementCount = strings.Count(currentContent, edit.OldString)
		} else {
			occurrenceCount := strings.Count(currentContent, edit.OldString)
			if occurrenceCount == 0 {
				_ = os.Rename(backupPath, filePath)
				return "", fmt.Errorf("edit %d: old_string not found in file", i+1)
			}
			if occurrenceCount > 1 {
				_ = os.Rename(backupPath, filePath)
				return "", fmt.Errorf("edit %d: old_string appears %d times in file - use replace_all=true or provide more context to make it unique", i+1, occurrenceCount)
			}

			modifiedContent = strings.Replace(currentContent, edit.OldString, edit.NewString, 1)
			replacementCount = 1
		}

		if replacementCount == 0 {
			_ = os.Rename(backupPath, filePath)
			return "", fmt.Errorf("edit %d: old_string not found in file", i+1)
		}

		currentContent = modifiedContent
		totalReplacements += replacementCount
	}

	if err := os.WriteFile(filePath, []byte(currentContent), stat.Mode()); err != nil {
		if restoreErr := os.Rename(backupPath, filePath); restoreErr != nil {
			return "", fmt.Errorf("failed to write file and failed to restore backup: write error: %w, restore error: %v", err, restoreErr)
		}
		return "", fmt.Errorf("failed to write file (backup restored): %w", err)
	}

	_ = os.Remove(backupPath)

	return fmt.Sprintf("Successfully applied %d edits with %d total replacements in %s", len(edits), totalReplacements, filePath), nil
}
