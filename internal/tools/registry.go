// Package tools provides tool registry and unified registration framework for MCP tools.
package tools

import (
	"context"
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
	case "Bash":
		return "system"
	case "WebFetch", "WebSearch":
		return "web"
	case "NotebookRead", "NotebookEdit":
		return "notebook"
	case "TodoRead", "TodoWrite":
		return "todo"
	default:
		return "unknown"
	}
}

// GetCategories returns all available tool categories.
func (r *Registry) GetCategories() []string {
	return []string{"file", "system", "web", "notebook", "todo"}
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

// =============================================================================
// Unified Tool Registration Framework
// =============================================================================

// ToolFactory is a function that creates a ServerTool given a context.
// This standardizes the CreateXXXTool pattern used throughout the codebase.
type ToolFactory func(*Context) *ServerTool

// ToolGroupFactory is a function that creates multiple ServerTools given a context.
// This standardizes the CreateXXXTools pattern used throughout the codebase.
type ToolGroupFactory func(*Context) []*ServerTool

// ToolDefinition contains metadata and factory for a tool.
type ToolDefinition struct {
	Name        string
	Description string
	Category    string
	Factory     ToolFactory
}

// ToolGroupDefinition contains metadata and factory for a group of tools.
type ToolGroupDefinition struct {
	Name        string
	Description string
	Category    string
	Factory     ToolGroupFactory
}

// ToolBuilder provides a fluent interface for building tools with type safety.
type ToolBuilder[T any] struct {
	name        string
	description string
	category    string
	handler     func(context.Context, *mcp.ServerSession, *mcp.CallToolParamsFor[T]) (*mcp.CallToolResultFor[any], error)
	ctx         *Context
}

// NewToolBuilder creates a new tool builder with type-safe parameter validation.
func NewToolBuilder[T any](name, description string, ctx *Context) *ToolBuilder[T] {
	return &ToolBuilder[T]{
		name:        name,
		description: description,
		category:    "unknown",
		ctx:         ctx,
	}
}

// WithCategory sets the tool category for organization.
func (b *ToolBuilder[T]) WithCategory(category string) *ToolBuilder[T] {
	b.category = category
	return b
}

// WithHandler sets the tool handler function with proper MCP SDK typing.
func (b *ToolBuilder[T]) WithHandler(handler func(context.Context, *mcp.ServerSession, *mcp.CallToolParamsFor[T]) (*mcp.CallToolResultFor[any], error)) *ToolBuilder[T] {
	b.handler = handler
	return b
}

// Build creates the ServerTool with all configured options.
func (b *ToolBuilder[T]) Build() *ServerTool {
	if b.handler == nil {
		panic(fmt.Sprintf("handler not set for tool %s", b.name))
	}

	tool := &mcp.Tool{
		Name:        b.name,
		Description: b.description,
	}

	return &ServerTool{
		Tool: tool,
		RegisterFunc: func(server *mcp.Server) {
			mcp.AddTool(server, tool, b.handler)
		},
	}
}

// ToolRegistry provides advanced tool registration and management.
type ToolRegistry struct {
	mu          sync.RWMutex
	definitions map[string]*ToolDefinition
	groups      map[string]*ToolGroupDefinition
	ctx         *Context
}

// NewToolRegistry creates a new advanced tool registry.
func NewToolRegistry(ctx *Context) *ToolRegistry {
	return &ToolRegistry{
		definitions: make(map[string]*ToolDefinition),
		groups:      make(map[string]*ToolGroupDefinition),
		ctx:         ctx,
	}
}

// RegisterTool registers a single tool definition.
func (tr *ToolRegistry) RegisterTool(def *ToolDefinition) error {
	tr.mu.Lock()
	defer tr.mu.Unlock()

	if def.Name == "" {
		return fmt.Errorf("tool name cannot be empty")
	}

	if def.Factory == nil {
		return fmt.Errorf("tool factory cannot be nil")
	}

	if _, exists := tr.definitions[def.Name]; exists {
		return fmt.Errorf("tool %s is already registered", def.Name)
	}

	tr.definitions[def.Name] = def
	return nil
}

// RegisterToolGroup registers a group of tools.
func (tr *ToolRegistry) RegisterToolGroup(def *ToolGroupDefinition) error {
	tr.mu.Lock()
	defer tr.mu.Unlock()

	if def.Name == "" {
		return fmt.Errorf("tool group name cannot be empty")
	}

	if def.Factory == nil {
		return fmt.Errorf("tool group factory cannot be nil")
	}

	if _, exists := tr.groups[def.Name]; exists {
		return fmt.Errorf("tool group %s is already registered", def.Name)
	}

	tr.groups[def.Name] = def
	return nil
}

// CreateAllTools creates all registered tools and tool groups.
func (tr *ToolRegistry) CreateAllTools() []*ServerTool {
	tr.mu.RLock()
	defer tr.mu.RUnlock()

	var allTools []*ServerTool

	// Create individual tools
	for _, def := range tr.definitions {
		tool := def.Factory(tr.ctx)
		allTools = append(allTools, tool)
	}

	// Create tool groups
	for _, def := range tr.groups {
		tools := def.Factory(tr.ctx)
		allTools = append(allTools, tools...)
	}

	return allTools
}

// CreateToolsByCategory creates tools filtered by category.
func (tr *ToolRegistry) CreateToolsByCategory(category string) []*ServerTool {
	tr.mu.RLock()
	defer tr.mu.RUnlock()

	var categoryTools []*ServerTool

	// Create individual tools in category
	for _, def := range tr.definitions {
		if def.Category == category {
			tool := def.Factory(tr.ctx)
			categoryTools = append(categoryTools, tool)
		}
	}

	// Create tool groups in category
	for _, def := range tr.groups {
		if def.Category == category {
			tools := def.Factory(tr.ctx)
			categoryTools = append(categoryTools, tools...)
		}
	}

	return categoryTools
}

// GetDefinitions returns all tool definitions.
func (tr *ToolRegistry) GetDefinitions() map[string]*ToolDefinition {
	tr.mu.RLock()
	defer tr.mu.RUnlock()

	result := make(map[string]*ToolDefinition)
	for k, v := range tr.definitions {
		result[k] = v
	}
	return result
}

// GetGroups returns all tool group definitions.
func (tr *ToolRegistry) GetGroups() map[string]*ToolGroupDefinition {
	tr.mu.RLock()
	defer tr.mu.RUnlock()

	result := make(map[string]*ToolGroupDefinition)
	for k, v := range tr.groups {
		result[k] = v
	}
	return result
}

// ListCategories returns all unique categories across tools and groups.
func (tr *ToolRegistry) ListCategories() []string {
	tr.mu.RLock()
	defer tr.mu.RUnlock()

	categorySet := make(map[string]bool)

	for _, def := range tr.definitions {
		categorySet[def.Category] = true
	}

	for _, def := range tr.groups {
		categorySet[def.Category] = true
	}

	categories := make([]string, 0, len(categorySet))
	for category := range categorySet {
		categories = append(categories, category)
	}

	sort.Strings(categories)
	return categories
}

// RegisterAllTools registers common tool patterns for simplified bulk registration.
func RegisterAllTools(registry *ToolRegistry) error {
	// This would be used in the future to register all standard tools
	// For now, it's a placeholder that demonstrates the pattern

	// Example usage (not implemented yet):
	// err := registry.RegisterToolGroup(&ToolGroupDefinition{
	//     Name:        "file_tools",
	//     Description: "File operation tools",
	//     Category:    "file",
	//     Factory:     file.CreateFileTools,
	// })
	// if err != nil {
	//     return fmt.Errorf("failed to register file tools: %w", err)
	// }

	return nil
}

// =============================================================================
// Usage Examples and Patterns
// =============================================================================

/*
Example 1: Using ToolBuilder for a simple tool

func CreateMyTool(ctx *tools.Context) *tools.ServerTool {
    return tools.NewToolBuilder[MyToolArgs]("MyTool", "Description", ctx).
        WithCategory("file").
        WithHandler(func(ctxReq context.Context, session *mcp.ServerSession, params *mcp.CallToolParamsFor[MyToolArgs]) (*mcp.CallToolResultFor[any], error) {
            args := params.Arguments
            // Tool implementation here
            return tools.CreateStandardSuccessResult("Success", nil)
        }).
        Build()
}

Example 2: Using ToolRegistry to register existing tool groups

func registerAllTools(ctx *tools.Context) error {
    registry := tools.NewToolRegistry(ctx)

    // Register existing tool groups
    err := registry.RegisterToolGroup(&tools.ToolGroupDefinition{
        Name:        "file_tools",
        Description: "File operation tools",
        Category:    "file",
        Factory:     file.CreateFileTools,
    })
    if err != nil {
        return err
    }

    err = registry.RegisterToolGroup(&tools.ToolGroupDefinition{
        Name:        "bash_tools",
        Description: "Command execution tools",
        Category:    "system",
        Factory:     bash.CreateBashTools,
    })
    if err != nil {
        return err
    }

    // Create all tools and register with server
    allTools := registry.CreateAllTools()
    for _, tool := range allTools {
        tool.RegisterFunc(mcpServer)
    }

    return nil
}

Example 3: Using ArgsValidator for common validation patterns

func myToolHandler(ctxReq context.Context, session *mcp.ServerSession, params *mcp.CallToolParamsFor[MyArgs]) (*mcp.CallToolResultFor[any], error) {
    validator := tools.NewArgsValidator(ctx)

    // Validate file path
    sanitizedPath, err := validator.ValidateFilePath(params.Arguments.FilePath)
    if err != nil {
        return tools.CreateStandardErrorResult(err.Error(), nil), nil
    }

    // Tool implementation
    return tools.CreateStandardSuccessResult("Operation completed", map[string]any{
        "path": sanitizedPath,
    }), nil
}

Benefits of the Unified Framework:
1. Type-safe tool parameter validation with generics
2. Standardized error and success result creation
3. Common validation patterns for paths, commands, URLs
4. Flexible registration with categories and metadata
5. Fluent builder interface for clean tool creation
6. Compatible with existing MCP SDK patterns
7. Eliminates boilerplate in CreateXXXTools functions
*/

// CreateStandardErrorResult creates a standardized error result for tools.
func CreateStandardErrorResult(message string, details map[string]any) *mcp.CallToolResultFor[any] {
	result := &mcp.CallToolResultFor[any]{
		Content: []mcp.Content{&mcp.TextContent{Text: "Error: " + message}},
		IsError: true,
	}

	if details != nil {
		result.Meta = details
	}

	return result
}

// CreateStandardSuccessResult creates a standardized success result for tools.
func CreateStandardSuccessResult(message string, metadata map[string]any) *mcp.CallToolResultFor[any] {
	result := &mcp.CallToolResultFor[any]{
		Content: []mcp.Content{&mcp.TextContent{Text: message}},
		IsError: false,
	}

	if metadata != nil {
		result.Meta = metadata
	}

	return result
}

// ValidateAndSanitizeArgs provides common argument validation and sanitization.
type ArgsValidator struct {
	ctx *Context
}

// NewArgsValidator creates a new argument validator.
func NewArgsValidator(ctx *Context) *ArgsValidator {
	return &ArgsValidator{ctx: ctx}
}

// ValidateFilePath validates and sanitizes a file path argument.
func (v *ArgsValidator) ValidateFilePath(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("file path cannot be empty")
	}

	sanitized, err := v.ctx.Validator.SanitizePath(path)
	if err != nil {
		return "", fmt.Errorf("invalid file path: %w", err)
	}

	if err := v.ctx.Validator.ValidatePath(sanitized); err != nil {
		return "", fmt.Errorf("path validation failed: %w", err)
	}

	return sanitized, nil
}

// ValidateCommand validates a command and its arguments.
func (v *ArgsValidator) ValidateCommand(cmd string, args []string) error {
	if cmd == "" {
		return fmt.Errorf("command cannot be empty")
	}

	return v.ctx.Validator.ValidateCommand(cmd, args)
}

// ValidateURL validates a URL argument.
func (v *ArgsValidator) ValidateURL(url string) error {
	if url == "" {
		return fmt.Errorf("URL cannot be empty")
	}

	return v.ctx.Validator.ValidateURL(url)
}
