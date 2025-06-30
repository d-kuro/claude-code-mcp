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

		if _, err := regexp.Compile(args.Pattern); err != nil {
			return &mcp.CallToolResultFor[any]{
				Content: []mcp.Content{&mcp.TextContent{Text: "Error: Invalid regular expression: " + err.Error()}},
				IsError: true,
			}, nil
		}

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

// grepFilesWithRipgrep performs content search using ripgrep command and returns sorted results.
func grepFilesWithRipgrep(searchPath, pattern string, includePattern *string) (string, error) {
	stat, err := os.Stat(searchPath)
	if err != nil {
		return "", fmt.Errorf("failed to stat search path: %w", err)
	}

	if !stat.IsDir() {
		return "", fmt.Errorf("search path is not a directory")
	}

	rgPath, err := FindBinary("rg")
	if err != nil {
		return "", fmt.Errorf("ripgrep (rg) not found: %w - please install ripgrep for optimal performance", err)
	}

	executor := NewCommandExecutor(30 * time.Second)

	args := []string{
		"--files-with-matches",
		"--no-heading",
		"--no-line-number",
		"--color=never",
		"--hidden",
		"--follow",
		"--case-sensitive",
	}

	if includePattern != nil && *includePattern != "" {
		globPattern := convertIncludePatternToGlob(*includePattern)
		args = append(args, "--glob", globPattern)
	}

	args = append(args, pattern, searchPath)

	if err := executor.ValidateCommand("rg", args); err != nil {
		return "", fmt.Errorf("command validation failed: %w", err)
	}

	result, err := executor.Execute(context.Background(), rgPath, args...)
	if err != nil {
		return "", fmt.Errorf("failed to execute ripgrep: %w", err)
	}

	if result.ExitCode == 2 {
		return "", fmt.Errorf("ripgrep error: %s", result.Stderr)
	}

	if result.ExitCode == 1 || strings.TrimSpace(result.Stdout) == "" {
		return fmt.Sprintf("No files found containing pattern '%s' in directory '%s'", pattern, searchPath), nil
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
	output.WriteString(fmt.Sprintf("Found %d file(s) containing pattern '%s' in directory '%s':\n", len(matches), pattern, searchPath))

	for _, match := range matches {
		output.WriteString(match.Path + "\n")
	}

	return strings.TrimSuffix(output.String(), "\n"), nil
}

// convertIncludePatternToGlob converts a Claude Code include pattern to a ripgrep glob pattern.
func convertIncludePatternToGlob(includePattern string) string {
	if strings.Contains(includePattern, "{") && strings.Contains(includePattern, "}") {
		return includePattern
	}
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

	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil && err.Error() != "EOF" {
		return false, err
	}

	if isBinaryContent(buffer[:n]) {
		return false, nil
	}

	if _, err := file.Seek(0, 0); err != nil {
		return false, err
	}

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
	nullBytes := 0
	nonPrintable := 0

	for _, b := range data {
		if b == 0 {
			nullBytes++
		}
		if b < 32 && b != 9 && b != 10 && b != 13 {
			nonPrintable++
		}
	}

	if len(data) > 0 && (float64(nullBytes)/float64(len(data)) > 0.01 || float64(nonPrintable)/float64(len(data)) > 0.30) {
		return true
	}

	return false
}

// matchIncludePattern matches a filename against an include pattern.
func matchIncludePattern(pattern, fileName string) (bool, error) {
	if strings.Contains(pattern, "{") && strings.Contains(pattern, "}") {
		return matchBracePattern(pattern, fileName)
	}

	return filepath.Match(pattern, fileName)
}

// matchBracePattern handles brace expansion patterns like "*.{ts,tsx}".
func matchBracePattern(pattern, fileName string) (bool, error) {
	start := strings.Index(pattern, "{")
	end := strings.Index(pattern, "}")

	if start == -1 || end == -1 || end <= start {
		return filepath.Match(pattern, fileName)
	}

	prefix := pattern[:start]
	suffix := pattern[end+1:]
	braceContent := pattern[start+1 : end]

	alternatives := strings.Split(braceContent, ",")

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
