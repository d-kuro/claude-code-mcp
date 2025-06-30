// Package web provides registration for web operation tools.
package web

import (
	"github.com/d-kuro/claude-code-mcp/internal/tools"
)

// CreateWebTools creates all web operation tools using MCP SDK patterns.
func CreateWebTools(ctx *tools.Context) []*tools.ServerTool {
	return []*tools.ServerTool{
		CreateWebFetchTool(ctx),
		CreateWebSearchTool(ctx),
	}
}
