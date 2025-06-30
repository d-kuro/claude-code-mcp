// Package todo provides task management tools using the MCP SDK patterns.
package todo

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/d-kuro/claude-code-mcp/internal/prompts"
	"github.com/d-kuro/claude-code-mcp/internal/tools"
)

// TodoStatus represents the status of a todo item.
type TodoStatus string

const (
	StatusPending    TodoStatus = "pending"
	StatusInProgress TodoStatus = "in_progress"
	StatusCompleted  TodoStatus = "completed"
)

// TodoPriority represents the priority of a todo item.
type TodoPriority string

const (
	PriorityHigh   TodoPriority = "high"
	PriorityMedium TodoPriority = "medium"
	PriorityLow    TodoPriority = "low"
)

// TodoItem represents a single todo item.
type TodoItem struct {
	ID       string       `json:"id"`
	Content  string       `json:"content"`
	Status   TodoStatus   `json:"status"`
	Priority TodoPriority `json:"priority"`
}

// TodoWriteArgs represents the arguments for the TodoWrite tool.
type TodoWriteArgs struct {
	Todos []TodoItem `json:"todos"`
}

// CreateTodoReadTool creates the TodoRead tool using MCP SDK patterns.
func CreateTodoReadTool(ctx *tools.Context) *tools.ServerTool {
	handler := func(ctxReq context.Context, session *mcp.ServerSession, params *mcp.CallToolParamsFor[map[string]any]) (*mcp.CallToolResultFor[any], error) {
		// No permission check needed for reading todos
		// No arguments needed for TodoRead

		todos := GetSessionTodos(session)

		if len(todos) == 0 {
			return &mcp.CallToolResultFor[any]{
				Content: []mcp.Content{&mcp.TextContent{Text: "No todos found for this session."}},
			}, nil
		}

		// Format todos as JSON for consistent output
		todosJSON, err := json.MarshalIndent(todos, "", "  ")
		if err != nil {
			return &mcp.CallToolResultFor[any]{
				Content: []mcp.Content{&mcp.TextContent{Text: "Error: Failed to format todos: " + err.Error()}},
				IsError: true,
			}, nil
		}

		// Count by status
		statusCounts := make(map[TodoStatus]int)
		for _, todo := range todos {
			statusCounts[todo.Status]++
		}

		output := fmt.Sprintf("Found %d todo(s) for this session:\n\nStatus Summary:\n- Pending: %d\n- In Progress: %d\n- Completed: %d\n\nTodos:\n%s",
			len(todos),
			statusCounts[StatusPending],
			statusCounts[StatusInProgress],
			statusCounts[StatusCompleted],
			string(todosJSON))

		return &mcp.CallToolResultFor[any]{
			Content: []mcp.Content{&mcp.TextContent{Text: output}},
		}, nil
	}

	tool := &mcp.Tool{
		Name:        "TodoRead",
		Description: prompts.TodoReadToolDoc,
	}

	return &tools.ServerTool{
		Tool: tool,
		RegisterFunc: func(server *mcp.Server) {
			mcp.AddTool(server, tool, handler)
		},
	}
}

// CreateTodoWriteTool creates the TodoWrite tool using MCP SDK patterns.
func CreateTodoWriteTool(ctx *tools.Context) *tools.ServerTool {
	typedHandler := func(ctxReq context.Context, session *mcp.ServerSession, params *mcp.CallToolParamsFor[TodoWriteArgs]) (*mcp.CallToolResultFor[any], error) {
		args := params.Arguments

		// Validate todos
		if len(args.Todos) == 0 {
			return &mcp.CallToolResultFor[any]{
				Content: []mcp.Content{&mcp.TextContent{Text: "Error: todos array cannot be empty"}},
				IsError: true,
			}, nil
		}

		// Validate each todo item
		seenIDs := make(map[string]bool)
		for i, todo := range args.Todos {
			// Validate ID
			if todo.ID == "" {
				return &mcp.CallToolResultFor[any]{
					Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Error: todo %d: ID cannot be empty", i+1)}},
					IsError: true,
				}, nil
			}

			// Check for duplicate IDs
			if seenIDs[todo.ID] {
				return &mcp.CallToolResultFor[any]{
					Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Error: todo %d: duplicate ID '%s'", i+1, todo.ID)}},
					IsError: true,
				}, nil
			}
			seenIDs[todo.ID] = true

			// Validate content
			if todo.Content == "" {
				return &mcp.CallToolResultFor[any]{
					Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Error: todo %d: content cannot be empty", i+1)}},
					IsError: true,
				}, nil
			}

			// Validate status
			if !isValidStatus(todo.Status) {
				return &mcp.CallToolResultFor[any]{
					Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Error: todo %d: invalid status '%s'. Must be one of: pending, in_progress, completed", i+1, todo.Status)}},
					IsError: true,
				}, nil
			}

			// Validate priority
			if !isValidPriority(todo.Priority) {
				return &mcp.CallToolResultFor[any]{
					Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("Error: todo %d: invalid priority '%s'. Must be one of: high, medium, low", i+1, todo.Priority)}},
					IsError: true,
				}, nil
			}
		}

		// Check that only one todo is in_progress at a time
		inProgressCount := 0
		for _, todo := range args.Todos {
			if todo.Status == StatusInProgress {
				inProgressCount++
			}
		}
		if inProgressCount > 1 {
			return &mcp.CallToolResultFor[any]{
				Content: []mcp.Content{&mcp.TextContent{Text: "Error: only one todo can be in 'in_progress' status at a time"}},
				IsError: true,
			}, nil
		}

		// Update session todos
		SetSessionTodos(session, args.Todos)

		// Count by status
		statusCounts := make(map[TodoStatus]int)
		for _, todo := range args.Todos {
			statusCounts[todo.Status]++
		}

		output := fmt.Sprintf("Successfully updated todo list with %d item(s):\n- Pending: %d\n- In Progress: %d\n- Completed: %d",
			len(args.Todos),
			statusCounts[StatusPending],
			statusCounts[StatusInProgress],
			statusCounts[StatusCompleted])

		return &mcp.CallToolResultFor[any]{
			Content: []mcp.Content{&mcp.TextContent{Text: output}},
		}, nil
	}

	// Create a wrapper handler that converts from map[string]any to typed args
	wrapperHandler := func(ctx context.Context, session *mcp.ServerSession, params *mcp.CallToolParamsFor[map[string]any]) (*mcp.CallToolResultFor[any], error) {
		// Convert map[string]any to typed args
		var args TodoWriteArgs
		data, err := json.Marshal(params.Arguments)
		if err != nil {
			return &mcp.CallToolResultFor[any]{
				Content: []mcp.Content{&mcp.TextContent{Text: "Error: Failed to marshal arguments: " + err.Error()}},
				IsError: true,
			}, nil
		}

		if err := json.Unmarshal(data, &args); err != nil {
			return &mcp.CallToolResultFor[any]{
				Content: []mcp.Content{&mcp.TextContent{Text: "Error: Failed to unmarshal arguments: " + err.Error()}},
				IsError: true,
			}, nil
		}

		typedParams := &mcp.CallToolParamsFor[TodoWriteArgs]{
			Name:      params.Name,
			Arguments: args,
		}

		return typedHandler(ctx, session, typedParams)
	}

	tool := &mcp.Tool{
		Name:        "TodoWrite",
		Description: prompts.TodoWriteToolDoc,
	}

	return &tools.ServerTool{
		Tool: tool,
		RegisterFunc: func(server *mcp.Server) {
			mcp.AddTool(server, tool, wrapperHandler)
		},
	}
}

// isValidStatus checks if the given status is valid.
func isValidStatus(status TodoStatus) bool {
	switch status {
	case StatusPending, StatusInProgress, StatusCompleted:
		return true
	default:
		return false
	}
}

// isValidPriority checks if the given priority is valid.
func isValidPriority(priority TodoPriority) bool {
	switch priority {
	case PriorityHigh, PriorityMedium, PriorityLow:
		return true
	default:
		return false
	}
}
