// Package tools provides centralized response utilities for MCP tool handlers.
package tools

import (
	"encoding/json"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ErrorResponse creates a standardized error response for MCP tools.
func ErrorResponse(message string) *mcp.CallToolResultFor[any] {
	return &mcp.CallToolResultFor[any]{
		Content: []mcp.Content{&mcp.TextContent{Text: "Error: " + message}},
		IsError: true,
	}
}

// ErrorResponsef creates a standardized error response with formatted message.
func ErrorResponsef(format string, args ...any) *mcp.CallToolResultFor[any] {
	return ErrorResponse(fmt.Sprintf(format, args...))
}

// SuccessResponse creates a standardized success response with text content.
func SuccessResponse(message string) *mcp.CallToolResultFor[any] {
	return &mcp.CallToolResultFor[any]{
		Content: []mcp.Content{&mcp.TextContent{Text: message}},
		IsError: false,
	}
}

// SuccessResponsef creates a standardized success response with formatted message.
func SuccessResponsef(format string, args ...any) *mcp.CallToolResultFor[any] {
	return SuccessResponse(fmt.Sprintf(format, args...))
}

// JSONResponse creates a response with JSON content.
func JSONResponse(data any) *mcp.CallToolResultFor[any] {
	jsonBytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return ErrorResponsef("failed to marshal JSON: %v", err)
	}

	return &mcp.CallToolResultFor[any]{
		Content: []mcp.Content{&mcp.TextContent{Text: string(jsonBytes)}},
		IsError: false,
	}
}

// ResponseWithMeta creates a response with metadata.
func ResponseWithMeta(text string, meta map[string]any) *mcp.CallToolResultFor[any] {
	return &mcp.CallToolResultFor[any]{
		Content: []mcp.Content{&mcp.TextContent{Text: text}},
		Meta:    meta,
		IsError: false,
	}
}

// Common error response patterns

// InvalidPathError creates an error response for invalid file paths.
func InvalidPathError(err error) *mcp.CallToolResultFor[any] {
	return ErrorResponsef("Invalid file path: %v", err)
}

// PathValidationError creates an error response for path validation failures.
func PathValidationError(err error) *mcp.CallToolResultFor[any] {
	return ErrorResponsef("Path validation failed: %v", err)
}

// CommandValidationError creates an error response for command validation failures.
func CommandValidationError(err error) *mcp.CallToolResultFor[any] {
	return ErrorResponsef("Command validation failed: %v", err)
}

// FileOperationError creates an error response for file operation failures.
func FileOperationError(operation string, err error) *mcp.CallToolResultFor[any] {
	return ErrorResponsef("%s failed: %v", operation, err)
}

// ValidationError creates an error response for general validation failures.
func ValidationError(field, message string) *mcp.CallToolResultFor[any] {
	return ErrorResponsef("%s validation failed: %s", field, message)
}

// EmptyFieldError creates an error response for empty required fields.
func EmptyFieldError(fieldName string) *mcp.CallToolResultFor[any] {
	return ErrorResponsef("%s cannot be empty", fieldName)
}

// InvalidFieldError creates an error response for invalid field values.
func InvalidFieldError(fieldName, reason string) *mcp.CallToolResultFor[any] {
	return ErrorResponsef("Invalid %s: %s", fieldName, reason)
}

// TimeoutError creates an error response for timeout violations.
func TimeoutError(maxTimeout string) *mcp.CallToolResultFor[any] {
	return ErrorResponsef("Maximum timeout is %s", maxTimeout)
}

// NotFoundError creates an error response for missing resources.
func NotFoundError(resource string) *mcp.CallToolResultFor[any] {
	return ErrorResponsef("%s not found", resource)
}

// PermissionError creates an error response for permission issues.
func PermissionError(operation string) *mcp.CallToolResultFor[any] {
	return ErrorResponsef("Permission denied: %s", operation)
}

// ConflictError creates an error response for conflicts.
func ConflictError(message string) *mcp.CallToolResultFor[any] {
	return ErrorResponsef("Conflict: %s", message)
}

// Common success response patterns

// FileSuccessResponse creates a success response for file operations.
func FileSuccessResponse(operation, filePath string, details ...string) *mcp.CallToolResultFor[any] {
	message := fmt.Sprintf("%s successful for %s", operation, filePath)
	if len(details) > 0 {
		message += " (" + details[0] + ")"
	}
	return SuccessResponse(message)
}

// CountSuccessResponse creates a success response with count information.
func CountSuccessResponse(operation string, count int, target string) *mcp.CallToolResultFor[any] {
	return SuccessResponsef("%s: %d %s", operation, count, target)
}

// Helper functions for common response validation patterns

// ValidateNonEmpty validates that a field is not empty and returns an error response if it is.
func ValidateNonEmpty(value, fieldName string) *mcp.CallToolResultFor[any] {
	if value == "" {
		return EmptyFieldError(fieldName)
	}
	return nil
}

// ValidateEqual validates that two values are different and returns an error response if they are equal.
func ValidateNotEqual(value1, value2, field1, field2 string) *mcp.CallToolResultFor[any] {
	if value1 == value2 {
		return ErrorResponsef("%s and %s must be different", field1, field2)
	}
	return nil
}

// ValidatePathWithContext validates a path using the provided validator and returns appropriate error responses.
func ValidatePathWithContext(ctx *Context, filePath string) (string, *mcp.CallToolResultFor[any]) {
	sanitizedPath, err := ctx.Validator.SanitizePath(filePath)
	if err != nil {
		return "", InvalidPathError(err)
	}

	if err := ctx.Validator.ValidatePath(sanitizedPath); err != nil {
		return "", PathValidationError(err)
	}

	return sanitizedPath, nil
}

// ValidateCommandWithContext validates a command using the provided validator and returns appropriate error responses.
func ValidateCommandWithContext(ctx *Context, command string, args []string) *mcp.CallToolResultFor[any] {
	if err := ctx.Validator.ValidateCommand(command, args); err != nil {
		return CommandValidationError(err)
	}
	return nil
}

// WrapError wraps an error with additional context and returns an error response.
func WrapError(err error, context string) *mcp.CallToolResultFor[any] {
	return ErrorResponsef("%s: %v", context, err)
}

// ResponseBuilder provides a fluent interface for building responses.
type ResponseBuilder struct {
	content []mcp.Content
	meta    map[string]any
	isError bool
}

// NewResponse creates a new response builder.
func NewResponse() *ResponseBuilder {
	return &ResponseBuilder{
		content: make([]mcp.Content, 0),
		meta:    make(map[string]any),
		isError: false,
	}
}

// WithText adds text content to the response.
func (rb *ResponseBuilder) WithText(text string) *ResponseBuilder {
	rb.content = append(rb.content, &mcp.TextContent{Text: text})
	return rb
}

// WithTextf adds formatted text content to the response.
func (rb *ResponseBuilder) WithTextf(format string, args ...any) *ResponseBuilder {
	return rb.WithText(fmt.Sprintf(format, args...))
}

// WithMeta adds metadata to the response.
func (rb *ResponseBuilder) WithMeta(key string, value any) *ResponseBuilder {
	rb.meta[key] = value
	return rb
}

// AsError marks the response as an error.
func (rb *ResponseBuilder) AsError() *ResponseBuilder {
	rb.isError = true
	return rb
}

// Build creates the final MCP response.
func (rb *ResponseBuilder) Build() *mcp.CallToolResultFor[any] {
	response := &mcp.CallToolResultFor[any]{
		Content: rb.content,
		IsError: rb.isError,
	}

	if len(rb.meta) > 0 {
		response.Meta = rb.meta
	}

	return response
}
