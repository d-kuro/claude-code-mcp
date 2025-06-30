// Package workflow provides development workflow tools using the MCP SDK patterns.
package workflow

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/d-kuro/claude-code-mcp/internal/prompts"
	"github.com/d-kuro/claude-code-mcp/internal/tools"
)

// ExitPlanModeArgs represents the arguments for the exit_plan_mode tool.
type ExitPlanModeArgs struct {
	Plan string `json:"plan" jsonschema:"required,description=The plan you came up with, that you want to run by the user for approval. Supports markdown. The plan should be pretty concise."`
}

// CreateExitPlanModeTool creates the exit_plan_mode tool using MCP SDK patterns.
func CreateExitPlanModeTool(ctx *tools.Context) *mcp.ServerTool {
	handler := func(ctxReq context.Context, session *mcp.ServerSession, params *mcp.CallToolParamsFor[ExitPlanModeArgs]) (*mcp.CallToolResultFor[any], error) {
		args := params.Arguments

		// Validate plan content
		if args.Plan == "" {
			return &mcp.CallToolResultFor[any]{
				Content: []mcp.Content{&mcp.TextContent{Text: "Error: plan cannot be empty"}},
				IsError: true,
			}, nil
		}

		// Log the plan for debugging purposes
		ctx.Logger.WithTool("exit_plan_mode").Info("User requested to exit plan mode", "plan_length", len(args.Plan))

		// Format the output to indicate plan mode exit
		output := fmt.Sprintf(prompts.ExitPlanModeOutputTemplate, args.Plan)

		return &mcp.CallToolResultFor[any]{
			Content: []mcp.Content{&mcp.TextContent{Text: output}},
		}, nil
	}

	return mcp.NewServerTool(
		"exit_plan_mode",
		prompts.Default().ExitPlanMode,
		handler,
	)
}
