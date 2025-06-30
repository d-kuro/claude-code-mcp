// Package notebook provides registration for Jupyter notebook operation tools.
package notebook

import (
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/d-kuro/claude-code-mcp/internal/tools"
)

// CreateNotebookTools creates all notebook operation tools using MCP SDK patterns.
func CreateNotebookTools(ctx *tools.Context) []*mcp.ServerTool {
	return []*mcp.ServerTool{
		CreateNotebookReadTool(ctx),
		CreateNotebookEditTool(ctx),
	}
}
