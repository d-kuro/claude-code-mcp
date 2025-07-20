// Package file provides file operation tools using the MCP SDK patterns.
package file

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/d-kuro/claude-code-mcp/internal/prompts"
	"github.com/d-kuro/claude-code-mcp/internal/tools"
)

const (
	// Default buffer size for file reading (64KB)
	DefaultBufferSize = 64 * 1024
	// Large file threshold - files larger than this use streaming (10MB)
	LargeFileThreshold = 10 * 1024 * 1024
	// Maximum memory usage for in-memory reads (50MB)
	MaxMemoryUsage = 50 * 1024 * 1024
	// Default maximum lines to read
	DefaultMaxLines = 2000
	// Maximum line length before truncation
	MaxLineLength = 2000
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
// Uses optimized strategies based on file size for better performance.
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

	fileSize := stat.Size()

	// Handle empty files
	if fileSize == 0 {
		return "<system-reminder>\nWARNING: This file exists but has empty contents.\n</system-reminder>", nil
	}

	startOffset := 0
	if offset != nil {
		startOffset = *offset
	}

	maxLines := DefaultMaxLines
	if limit != nil {
		maxLines = *limit
	}

	// Choose strategy based on file size and memory constraints
	if fileSize > LargeFileThreshold || (int64(maxLines)*MaxLineLength) > MaxMemoryUsage {
		return readLargeFile(file, startOffset, maxLines)
	}

	return readSmallFile(file, startOffset, maxLines)
}

// readSmallFile optimally reads smaller files into memory using strings.Builder
func readSmallFile(file *os.File, startOffset, maxLines int) (string, error) {
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, DefaultBufferSize), DefaultBufferSize)

	var builder strings.Builder
	lineNumber := 1
	currentOffset := 0
	linesRead := 0

	// Pre-allocate buffer with estimated size
	builder.Grow(maxLines * 100) // Estimate 100 chars per line

	for scanner.Scan() && linesRead < maxLines {
		if currentOffset >= startOffset {
			line := scanner.Text()
			if len(line) > MaxLineLength {
				line = line[:MaxLineLength] + "... (truncated)"
			}

			if linesRead > 0 {
				builder.WriteByte('\n')
			}

			// Optimized line formatting using direct writes
			writeFormattedLine(&builder, lineNumber, line)
			linesRead++
		}
		lineNumber++
		currentOffset++
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error reading file: %w", err)
	}

	return builder.String(), nil
}

// readLargeFile uses streaming approach for large files with controlled memory usage
func readLargeFile(file *os.File, startOffset, maxLines int) (string, error) {
	reader := bufio.NewReaderSize(file, DefaultBufferSize)
	var builder strings.Builder

	lineNumber := 1
	currentOffset := 0
	linesRead := 0

	// Pre-allocate with conservative estimate
	builder.Grow(maxLines * 80)

	for linesRead < maxLines {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				// Handle last line without newline
				if len(line) > 0 && currentOffset >= startOffset {
					if len(line) > MaxLineLength {
						line = line[:MaxLineLength] + "... (truncated)"
					}

					if linesRead > 0 {
						builder.WriteByte('\n')
					}
					writeFormattedLine(&builder, lineNumber, line)
				}
				break
			}
			return "", fmt.Errorf("error reading file: %w", err)
		}

		// Remove trailing newline for processing
		if len(line) > 0 && line[len(line)-1] == '\n' {
			line = line[:len(line)-1]
		}

		if currentOffset >= startOffset {
			if len(line) > MaxLineLength {
				line = line[:MaxLineLength] + "... (truncated)"
			}

			if linesRead > 0 {
				builder.WriteByte('\n')
			}

			writeFormattedLine(&builder, lineNumber, line)
			linesRead++
		}

		lineNumber++
		currentOffset++
	}

	return builder.String(), nil
}

// writeFormattedLine efficiently writes a formatted line to the builder
// Optimized to avoid fmt.Sprintf allocations in tight loops
func writeFormattedLine(builder *strings.Builder, lineNumber int, line string) {
	// Convert line number to string efficiently
	lineNumStr := strconv.Itoa(lineNumber)

	// Calculate padding for right-alignment (5 characters total)
	padding := 5 - len(lineNumStr)
	for i := 0; i < padding; i++ {
		builder.WriteByte(' ')
	}

	builder.WriteString(lineNumStr)
	builder.WriteString("→")
	builder.WriteString(line)
}
