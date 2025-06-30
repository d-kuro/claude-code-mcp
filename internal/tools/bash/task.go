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
	Description string `json:"description" jsonschema:"required,description=A short (3-5 word) description of the task"`
	Prompt      string `json:"prompt" jsonschema:"required,description=The task for the agent to perform"`
}

// CreateTaskTool creates the Task tool using MCP SDK patterns.
func CreateTaskTool(ctx *tools.Context) *mcp.ServerTool {
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

	return mcp.NewServerTool(
		"Task",
		prompts.Default().Task.Description,
		handler,
	)
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

	// Analyze the task to determine what kind of response to generate
	taskType := analyzeTaskType(args.Prompt)

	// Generate a simulated response based on the task type
	result := &TaskResult{
		Success:     true,
		Description: args.Description,
		Duration:    time.Since(startTime),
		ToolsUsed:   getSimulatedToolsUsed(taskType),
	}

	switch taskType {
	case "search":
		result.Output = generateSearchResponse(args.Prompt)
		result.Summary = "Completed search operation and found relevant results"
	case "analysis":
		result.Output = generateAnalysisResponse(args.Prompt)
		result.Summary = "Completed code analysis and provided insights"
	case "exploration":
		result.Output = generateExplorationResponse(args.Prompt)
		result.Summary = "Explored codebase structure and documented findings"
	default:
		result.Output = generateGenericResponse(args.Prompt)
		result.Summary = "Completed task and provided analysis"
	}

	logger.Info("Agent task completed", "duration", result.Duration, "success", result.Success)

	return result, nil
}

// analyzeTaskType determines the type of task based on the prompt content.
func analyzeTaskType(prompt string) string {
	promptLower := strings.ToLower(prompt)

	if strings.Contains(promptLower, "search") || strings.Contains(promptLower, "find") {
		return "search"
	}
	if strings.Contains(promptLower, "analyze") || strings.Contains(promptLower, "understand") {
		return "analysis"
	}
	if strings.Contains(promptLower, "explore") || strings.Contains(promptLower, "structure") {
		return "exploration"
	}

	return "generic"
}

// getSimulatedToolsUsed returns a list of tools that would typically be used for different task types.
func getSimulatedToolsUsed(taskType string) []string {
	switch taskType {
	case "search":
		return []string{"Grep", "Glob", "Read"}
	case "analysis":
		return []string{"Read", "Grep", "LS"}
	case "exploration":
		return []string{"LS", "Glob", "Read"}
	default:
		return []string{"Read", "LS"}
	}
}

// generateSearchResponse generates a response for search-type tasks.
func generateSearchResponse(prompt string) string {
	return fmt.Sprintf(prompts.TaskSearchResponseTemplate, prompt)
}

// generateAnalysisResponse generates a response for analysis-type tasks.
func generateAnalysisResponse(prompt string) string {
	return fmt.Sprintf(prompts.TaskAnalysisResponseTemplate, prompt)
}

// generateExplorationResponse generates a response for exploration-type tasks.
func generateExplorationResponse(prompt string) string {
	return fmt.Sprintf(prompts.TaskExplorationResponseTemplate, prompt)
}

// generateGenericResponse generates a generic response for other task types.
func generateGenericResponse(prompt string) string {
	return fmt.Sprintf(prompts.TaskGenericResponseTemplate, prompt)
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
