// Package prompts provides centralized management for all system prompts
// used throughout the Claude Code MCP server.
package prompts

// ToolPrompts contains all prompts for MCP tools
type ToolPrompts struct {
	// Bash tool prompts
	Task TaskPrompts

	// Web tool prompts
	WebFetch   string
	WebSearch  string
	WebProcess string

	// Workflow tool prompts
	ExitPlanMode string

	// Todo tool prompts
	TodoRead  string
	TodoWrite string
}

// TaskPrompts contains prompts specific to the Task tool
type TaskPrompts struct {
	Description string
	Template    string
}

// Default returns the default prompts configuration
func Default() *ToolPrompts {
	return &ToolPrompts{
		Task: TaskPrompts{
			Description: TaskToolDescription,
			Template:    TaskGenericResponseTemplate,
		},
		WebFetch:     WebFetchToolDescription,
		WebSearch:    WebSearchToolDescription,
		WebProcess:   WebFetchContentProcessingTemplate,
		ExitPlanMode: ExitPlanModeToolDescription,
		TodoRead:     TodoReadToolDescription,
		TodoWrite:    TodoWriteToolDescription,
	}
}

// GetTaskTemplate returns the appropriate template based on task type
func GetTaskTemplate(taskType string) string {
	switch taskType {
	case "search":
		return TaskSearchResponseTemplate
	case "analysis":
		return TaskAnalysisResponseTemplate
	case "exploration":
		return TaskExplorationResponseTemplate
	default:
		return TaskGenericResponseTemplate
	}
}
