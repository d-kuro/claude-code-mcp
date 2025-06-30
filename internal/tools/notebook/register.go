// Package notebook provides registration for Jupyter notebook operation tools.
package notebook

import (
	"github.com/d-kuro/claude-code-mcp/internal/tools"
)

// CreateNotebookTools creates all notebook operation tools using MCP SDK patterns.
func CreateNotebookTools(ctx *tools.Context) []*tools.ServerTool {
	return []*tools.ServerTool{
		CreateNotebookReadTool(ctx),
		CreateNotebookEditTool(ctx),
	}
}
