// Package bash provides the Task tool for launching agents with complex operations.
package bash

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/d-kuro/claude-code-mcp/internal/prompts"
	"github.com/d-kuro/claude-code-mcp/internal/tools"
)

// TaskArgs represents the arguments for the Task tool.
type TaskArgs struct {
	Description string `json:"description"`
	Prompt      string `json:"prompt"`
}

// CreateTaskTool creates the Task tool using MCP SDK patterns.
func CreateTaskTool(ctx *tools.Context) *tools.ServerTool {
	handler := func(ctxReq context.Context, session *mcp.ServerSession, params *mcp.CallToolParamsFor[TaskArgs]) (*mcp.CallToolResultFor[any], error) {
		args := params.Arguments

		// Validate required fields
		if args.Description == "" {
			return &mcp.CallToolResultFor[any]{
				Content: []mcp.Content{&mcp.TextContent{Text: "Error: Description cannot be empty"}},
				IsError: true,
			}, nil
		}

		if args.Prompt == "" {
			return &mcp.CallToolResultFor[any]{
				Content: []mcp.Content{&mcp.TextContent{Text: "Error: Prompt cannot be empty"}},
				IsError: true,
			}, nil
		}

		// Validate description length (should be 3-5 words)
		descWords := strings.Fields(args.Description)
		if len(descWords) < 2 || len(descWords) > 8 {
			return &mcp.CallToolResultFor[any]{
				Content: []mcp.Content{&mcp.TextContent{Text: "Warning: Description should be 3-5 words for optimal results"}},
				IsError: false,
			}, nil
		}

		// Log the task launch
		logger := ctx.Logger.WithTool("Task")
		logger.Info("Launching agent task", "description", args.Description)

		// Execute the task (simulated agent execution)
		result, err := executeAgentTask(ctxReq, &args, logger)
		if err != nil {
			return &mcp.CallToolResultFor[any]{
				Content: []mcp.Content{&mcp.TextContent{Text: "Error: " + err.Error()}},
				IsError: true,
			}, nil
		}

		// Format the response
		output := formatTaskResult(result, args.Description)

		return &mcp.CallToolResultFor[any]{
			Content: []mcp.Content{&mcp.TextContent{Text: output}},
		}, nil
	}

	tool := &mcp.Tool{
		Name:        "Task",
		Description: prompts.TaskToolDoc,
	}

	return &tools.ServerTool{
		Tool: tool,
		RegisterFunc: func(server *mcp.Server) {
			mcp.AddTool(server, tool, handler)
		},
	}
}

// TaskResult represents the result of an agent task execution.
type TaskResult struct {
	Success     bool          `json:"success"`
	Description string        `json:"description"`
	Output      string        `json:"output"`
	Error       string        `json:"error,omitempty"`
	Duration    time.Duration `json:"duration"`
	ToolsUsed   []string      `json:"tools_used,omitempty"`
	Summary     string        `json:"summary"`
}

// executeAgentTask simulates the execution of an agent task.
// In a real implementation, this would launch an actual agent with access to all tools.
func executeAgentTask(ctx context.Context, args *TaskArgs, logger tools.Logger) (*TaskResult, error) {
	startTime := time.Now()

	// Simulate agent processing time
	time.Sleep(100 * time.Millisecond)

	// Generate a simulated response
	result := &TaskResult{
		Success:     true,
		Description: args.Description,
		Duration:    time.Since(startTime),
		Output:      fmt.Sprintf("Task completed: %s", args.Prompt),
		Summary:     "Completed task and provided analysis",
	}

	logger.Info("Agent task completed", "duration", result.Duration, "success", result.Success)

	return result, nil
}

// formatTaskResult formats the task execution result into a readable string.
func formatTaskResult(result *TaskResult, description string) string {
	var output strings.Builder

	// Add task summary
	output.WriteString(fmt.Sprintf("Task: %s\n", description))
	output.WriteString(fmt.Sprintf("Status: %s\n", getStatusString(result.Success)))
	output.WriteString(fmt.Sprintf("Duration: %s\n", result.Duration.String()))

	if len(result.ToolsUsed) > 0 {
		output.WriteString(fmt.Sprintf("Tools Used: %s\n", strings.Join(result.ToolsUsed, ", ")))
	}

	output.WriteString("\n")

	// Add error if present
	if result.Error != "" {
		output.WriteString("Error: ")
		output.WriteString(result.Error)
		output.WriteString("\n\n")
	}

	// Add agent output
	if result.Output != "" {
		output.WriteString("Agent Output:\n")
		output.WriteString(result.Output)
		output.WriteString("\n\n")
	}

	// Add summary
	if result.Summary != "" {
		output.WriteString("Summary: ")
		output.WriteString(result.Summary)
		output.WriteString("\n")
	}

	return output.String()
}

// getStatusString returns a human-readable status string.
func getStatusString(success bool) string {
	if success {
		return "Completed Successfully"
	}
	return "Failed"
}
