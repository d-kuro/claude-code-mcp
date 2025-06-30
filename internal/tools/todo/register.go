// Package todo provides registration for todo management tools.
package todo

import (
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/d-kuro/claude-code-mcp/internal/tools"
)

// CreateTodoTools creates all todo management tools using MCP SDK patterns.
func CreateTodoTools(ctx *tools.Context) []*mcp.ServerTool {
	return []*mcp.ServerTool{
		CreateTodoReadTool(ctx),
		CreateTodoWriteTool(ctx),
	}
}
