// Package bash provides registration for bash command execution tools.
package bash

import (
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/d-kuro/claude-code-mcp/internal/tools"
)

// CreateBashTools creates all bash operation tools using MCP SDK patterns.
func CreateBashTools(ctx *tools.Context) []*mcp.ServerTool {
	return []*mcp.ServerTool{
		CreateBashTool(ctx),
		CreateTaskTool(ctx),
	}
}
