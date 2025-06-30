// Package workflow provides registration for development workflow tools.
package workflow

import (
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/d-kuro/claude-code-mcp/internal/tools"
)

// CreateWorkflowTools creates all workflow tools using MCP SDK patterns.
func CreateWorkflowTools(ctx *tools.Context) []*mcp.ServerTool {
	return []*mcp.ServerTool{
		CreateExitPlanModeTool(ctx),
	}
}
