// Package tools provides tool registry for managing MCP tools.
package tools

import (
	"fmt"
	"sort"
	"sync"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Registry manages the collection of available tools.
type Registry struct {
	mu    sync.RWMutex
	tools map[string]Tool
	ctx   *Context
}

// NewRegistry creates a new tool registry with the given context.
func NewRegistry(ctx *Context) *Registry {
	return &Registry{
		tools: make(map[string]Tool),
		ctx:   ctx,
	}
}

// Register registers a tool with the registry.
func (r *Registry) Register(tool Tool) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := tool.Name()
	if name == "" {
		return fmt.Errorf("tool name cannot be empty")
	}

	if _, exists := r.tools[name]; exists {
		return fmt.Errorf("tool %s is already registered", name)
	}

	r.tools[name] = tool
	return nil
}

// Get retrieves a tool by name.
func (r *Registry) Get(name string) (Tool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tool, exists := r.tools[name]
	return tool, exists
}

// List returns all registered tool names in sorted order.
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		names = append(names, name)
	}

	sort.Strings(names)
	return names
}

// GetMCPTools returns MCP tool schemas for all registered tools.
func (r *Registry) GetMCPTools() []*mcp.Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tools := make([]*mcp.Tool, 0, len(r.tools))
	for _, tool := range r.tools {
		tools = append(tools, tool.Schema())
	}

	return tools
}

// CreateHandlerMap creates a map of tool handlers for MCP server registration.
func (r *Registry) CreateHandlerMap() map[string]mcp.ToolHandler {
	r.mu.RLock()
	defer r.mu.RUnlock()

	handlers := make(map[string]mcp.ToolHandler)
	for name, tool := range r.tools {
		handlers[name] = tool.Handler()
	}

	return handlers
}

// Count returns the number of registered tools.
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.tools)
}

// Unregister removes a tool from the registry.
func (r *Registry) Unregister(name string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.tools[name]; !exists {
		return false
	}

	delete(r.tools, name)
	return true
}

// Clear removes all tools from the registry.
func (r *Registry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.tools = make(map[string]Tool)
}

// GetToolsByCategory returns tools filtered by category.
// Categories: file, system, web, notebook, todo, workflow
func (r *Registry) GetToolsByCategory(category string) []Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var categoryTools []Tool

	for _, tool := range r.tools {
		if r.getToolCategory(tool.Name()) == category {
			categoryTools = append(categoryTools, tool)
		}
	}

	return categoryTools
}

// getToolCategory determines the category of a tool based on its name.
func (r *Registry) getToolCategory(toolName string) string {
	switch toolName {
	case "Read", "Write", "Edit", "MultiEdit", "LS", "Glob", "Grep":
		return "file"
	case "Bash", "Task":
		return "system"
	case "WebFetch", "WebSearch":
		return "web"
	case "NotebookRead", "NotebookEdit":
		return "notebook"
	case "TodoRead", "TodoWrite":
		return "todo"
	case "exit_plan_mode":
		return "workflow"
	default:
		return "unknown"
	}
}

// GetCategories returns all available tool categories.
func (r *Registry) GetCategories() []string {
	return []string{"file", "system", "web", "notebook", "todo", "workflow"}
}

// Validate checks if all registered tools are properly configured.
func (r *Registry) Validate() error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for name, tool := range r.tools {
		if tool.Name() != name {
			return fmt.Errorf("tool name mismatch: registered as %s but reports name %s", name, tool.Name())
		}

		if tool.Description() == "" {
			return fmt.Errorf("tool %s has empty description", name)
		}

		if tool.Schema() == nil {
			return fmt.Errorf("tool %s has nil schema", name)
		}

		if tool.Handler() == nil {
			return fmt.Errorf("tool %s has nil handler", name)
		}
	}

	return nil
}
