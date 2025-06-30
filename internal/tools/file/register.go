// Package file provides registration for file operation tools.
package file

import (
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/d-kuro/claude-code-mcp/internal/tools"
)

// CreateFileTools creates all file operation tools using MCP SDK patterns.
func CreateFileTools(ctx *tools.Context) []*mcp.ServerTool {
	return []*mcp.ServerTool{
		CreateReadTool(ctx),
		CreateWriteTool(ctx),
		CreateEditTool(ctx),
		CreateMultiEditTool(ctx),
		CreateLSTool(ctx),
		CreateGlobTool(ctx),
		CreateGrepTool(ctx),
	}
}
