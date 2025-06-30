// Package file provides registration for file operation tools.
package file

import (
	"github.com/d-kuro/claude-code-mcp/internal/tools"
)

// CreateFileTools creates all file operation tools using MCP SDK patterns.
func CreateFileTools(ctx *tools.Context) []*tools.ServerTool {
	return []*tools.ServerTool{
		CreateReadTool(ctx),
		CreateWriteTool(ctx),
		CreateEditTool(ctx),
		CreateMultiEditTool(ctx),
		CreateLSTool(ctx),
		CreateGlobTool(ctx),
		CreateGrepTool(ctx),
	}
}
