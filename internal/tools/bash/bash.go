// Package bash provides command execution tools with persistent sessions.
package bash

import (
	"context"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/d-kuro/claude-code-mcp/internal/tools"
)

// BashArgs represents the arguments for the Bash tool.
type BashArgs struct {
	Command     string  `json:"command" jsonschema:"required,description=The command to execute"`
	Description *string `json:"description,omitempty" jsonschema:"description=Clear concise description of what this command does in 5-10 words. Examples: Input: ls Output: Lists files in current directory"`
	Timeout     *int    `json:"timeout,omitempty" jsonschema:"description=Optional timeout in milliseconds (max 600000)"`
}

// CreateBashTool creates the Bash tool using MCP SDK patterns.
func CreateBashTool(ctx *tools.Context) *mcp.ServerTool {
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

	return mcp.NewServerTool(
		"Bash",
		"Executes a given bash command in a persistent shell session with optional timeout, ensuring proper handling and security measures.\n\nBefore executing the command, please follow these steps:\n\n1. Directory Verification:\n   - If the command will create new directories or files, first use the LS tool to verify the parent directory exists and is the correct location\n   - For example, before running \"mkdir foo/bar\", first use LS to check that \"foo\" exists and is the intended parent directory\n\n2. Command Execution:\n   - Always quote file paths that contain spaces with double quotes (e.g., cd \"path with spaces/file.txt\")\n   - Examples of proper quoting:\n     - cd \"/Users/name/My Documents\" (correct)\n     - cd /Users/name/My Documents (incorrect - will fail)\n     - python \"/path/with spaces/script.py\" (correct)\n     - python /path/with spaces/script.py (incorrect - will fail)\n   - After ensuring proper quoting, execute the command.\n   - Capture the output of the command.\n\nUsage notes:\n  - The command argument is required.\n  - You can specify an optional timeout in milliseconds (up to 600000ms / 10 minutes). If not specified, commands will timeout after 120000ms (2 minutes).\n  - It is very helpful if you write a clear, concise description of what this command does in 5-10 words.\n  - If the output exceeds 30000 characters, output will be truncated before being returned to you.\n  - VERY IMPORTANT: You MUST avoid using search commands like `find` and `grep`. Instead use Grep, Glob, or Task to search. You MUST avoid read tools like `cat`, `head`, `tail`, and `ls`, and use Read and LS to read files.\n - If you _still_ need to run `grep`, STOP. ALWAYS USE ripgrep at `rg` first, which all users have pre-installed.\n  - When issuing multiple commands, use the ';' or '&&' operator to separate them. DO NOT use newlines (newlines are ok in quoted strings).\n  - Try to maintain your current working directory throughout the session by using absolute paths and avoiding usage of `cd`. You may use `cd` if the User explicitly requests it.\n    <good-example>\n    pytest /foo/bar/tests\n    </good-example>\n    <bad-example>\n    cd /foo/bar && pytest tests\n    </bad-example>",
		handler,
	)
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
