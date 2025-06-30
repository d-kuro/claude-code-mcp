// Package todo provides registration for todo management tools.
package todo

import (
	"github.com/d-kuro/claude-code-mcp/internal/tools"
)

// CreateTodoTools creates all todo management tools using MCP SDK patterns.
func CreateTodoTools(ctx *tools.Context) []*tools.ServerTool {
	return []*tools.ServerTool{
		CreateTodoReadTool(ctx),
		CreateTodoWriteTool(ctx),
	}
}
