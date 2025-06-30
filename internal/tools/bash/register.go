// Package bash provides registration for bash command execution tools.
package bash

import (
	"github.com/d-kuro/claude-code-mcp/internal/tools"
)

// CreateBashTools creates all bash operation tools using MCP SDK patterns.
func CreateBashTools(ctx *tools.Context) []*tools.ServerTool {
	return []*tools.ServerTool{
		CreateBashTool(ctx),
	}
}
