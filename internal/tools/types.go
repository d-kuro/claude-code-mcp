// Package tools provides tool registry and common types for MCP tools.
package tools

import (
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Tool represents a Claude Code tool that can be registered with the MCP server.
type Tool interface {
	// Name returns the tool name.
	Name() string

	// Description returns the tool description.
	Description() string

	// Schema returns the MCP tool schema for registration.
	Schema() *mcp.Tool

	// Handler returns the MCP tool handler function.
	Handler() mcp.ToolHandler

	// Validate validates tool arguments before execution.
	Validate(args map[string]any) error
}

// Context contains common dependencies needed by tools.
type Context struct {
	Logger    Logger
	Validator Validator
}

// Logger defines the logging interface for tools.
type Logger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
	WithTool(toolName string) Logger
	WithSession(sessionID string) Logger
}

// Validator defines the security validation interface.
type Validator interface {
	ValidatePath(path string) error
	ValidateCommand(cmd string, args []string) error
	ValidateURL(url string) error
	SanitizePath(path string) (string, error)
}

// BaseTool provides common functionality for all tools.
type BaseTool struct {
	name        string
	description string
	ctx         *Context
}

// NewBaseTool creates a new base tool with the given context.
func NewBaseTool(name, description string, ctx *Context) *BaseTool {
	return &BaseTool{
		name:        name,
		description: description,
		ctx:         ctx,
	}
}

// Name returns the tool name.
func (t *BaseTool) Name() string {
	return t.name
}

// Description returns the tool description.
func (t *BaseTool) Description() string {
	return t.description
}

// Context returns the tool context.
func (t *BaseTool) Context() *Context {
	return t.ctx
}

// CreateErrorResult creates an error result for MCP tool responses.
func CreateErrorResult(message string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: "Error: " + message},
		},
		IsError: true,
	}
}

// CreateTextResult creates a text result for MCP tool responses.
func CreateTextResult(text string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: text},
		},
		IsError: false,
	}
}

// CreateResultWithMeta creates a result with metadata for MCP tool responses.
func CreateResultWithMeta(text string, meta map[string]any) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: text},
		},
		Meta:    meta,
		IsError: false,
	}
}

// ExecutionResult represents the result of a tool execution.
type ExecutionResult struct {
	Success  bool           `json:"success"`
	Output   string         `json:"output"`
	Error    string         `json:"error,omitempty"`
	Metadata map[string]any `json:"metadata,omitempty"`
	Duration string         `json:"duration,omitempty"`
}

// FileInfo represents file information.
type FileInfo struct {
	Path        string `json:"path"`
	Size        int64  `json:"size"`
	Modified    string `json:"modified"`
	IsDirectory bool   `json:"is_directory"`
	Permissions string `json:"permissions"`
}

// SearchResult represents a search result.
type SearchResult struct {
	File    string `json:"file"`
	Line    int    `json:"line"`
	Column  int    `json:"column"`
	Match   string `json:"match"`
	Context string `json:"context"`
}
