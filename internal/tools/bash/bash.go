// Package bash provides command execution tools with persistent sessions.
package bash

import (
	"context"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/d-kuro/claude-code-mcp/internal/prompts"
	"github.com/d-kuro/claude-code-mcp/internal/tools"
)

// BashArgs represents the arguments for the Bash tool.
type BashArgs struct {
	Command     string  `json:"command"`
	Description *string `json:"description,omitempty"`
	Timeout     *int    `json:"timeout,omitempty"`
}

// CreateBashTool creates the Bash tool using MCP SDK patterns.
func CreateBashTool(ctx *tools.Context) *tools.ServerTool {
	handler := func(ctxReq context.Context, session *mcp.ServerSession, params *mcp.CallToolParamsFor[BashArgs]) (*mcp.CallToolResultFor[any], error) {
		args := params.Arguments

		// Validate command is not empty
		if args.Command == "" {
			return &mcp.CallToolResultFor[any]{
				Content: []mcp.Content{&mcp.TextContent{Text: "Error: Command cannot be empty"}},
				IsError: true,
			}, nil
		}

		// Validate command security
		if err := ctx.Validator.ValidateCommand(args.Command, nil); err != nil {
			return &mcp.CallToolResultFor[any]{
				Content: []mcp.Content{&mcp.TextContent{Text: "Error: Command validation failed: " + err.Error()}},
				IsError: true,
			}, nil
		}

		// Determine timeout (default 120s, max 600s)
		timeout := 120 * time.Second
		if args.Timeout != nil {
			requestedTimeout := time.Duration(*args.Timeout) * time.Millisecond
			if requestedTimeout > 600*time.Second {
				return &mcp.CallToolResultFor[any]{
					Content: []mcp.Content{&mcp.TextContent{Text: "Error: Maximum timeout is 600000ms (10 minutes)"}},
					IsError: true,
				}, nil
			}
			if requestedTimeout > 0 {
				timeout = requestedTimeout
			}
		}

		// Get or create session manager
		sessionManager := GetSessionManager()

		// Execute command in persistent session
		result, err := sessionManager.ExecuteCommand(ctxReq, args.Command, timeout)
		if err != nil {
			return &mcp.CallToolResultFor[any]{
				Content: []mcp.Content{&mcp.TextContent{Text: "Error: " + err.Error()}},
				IsError: true,
			}, nil
		}

		// Format output
		output := formatCommandResult(result, args.Description)

		return &mcp.CallToolResultFor[any]{
			Content: []mcp.Content{&mcp.TextContent{Text: output}},
		}, nil
	}

	tool := &mcp.Tool{
		Name:        "Bash",
		Description: prompts.BashToolDoc,
	}

	return &tools.ServerTool{
		Tool: tool,
		RegisterFunc: func(server *mcp.Server) {
			mcp.AddTool(server, tool, handler)
		},
	}
}

// formatCommandResult formats the command execution result into a readable string.
func formatCommandResult(result *CommandResult, description *string) string {
	var output string

	// Add description if provided
	if description != nil && *description != "" {
		output += fmt.Sprintf("Description: %s\n\n", *description)
	}

	// Add command execution summary
	output += fmt.Sprintf("Command executed successfully (exit code: %d, duration: %s)\n\n", result.ExitCode, result.Duration)

	// Add stdout if present
	if result.Stdout != "" {
		output += "Output:\n"
		// Truncate output if too long (30000 characters)
		if len(result.Stdout) > 30000 {
			output += result.Stdout[:30000] + "\n... (output truncated)\n"
		} else {
			output += result.Stdout + "\n"
		}
	}

	// Add stderr if present
	if result.Stderr != "" {
		output += "\nError output:\n" + result.Stderr + "\n"
	}

	// Add working directory info
	if result.WorkingDirectory != "" {
		output += "\nCurrent working directory: " + result.WorkingDirectory
	}

	return output
}
