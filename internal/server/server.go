// Package server implements the MCP server for Claude Code tools.
package server

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/d-kuro/claude-code-mcp/internal/collections"
	"github.com/d-kuro/claude-code-mcp/internal/logging"
	"github.com/d-kuro/claude-code-mcp/internal/security"
	"github.com/d-kuro/claude-code-mcp/internal/tools"
	"github.com/d-kuro/claude-code-mcp/internal/tools/bash"
	"github.com/d-kuro/claude-code-mcp/internal/tools/file"
	"github.com/d-kuro/claude-code-mcp/internal/tools/notebook"
	"github.com/d-kuro/claude-code-mcp/internal/tools/todo"
	"github.com/d-kuro/claude-code-mcp/internal/tools/web"
	"github.com/d-kuro/claude-code-mcp/internal/version"
)

// loggerAdapter wraps logging.Logger to implement tools.Logger interface.
// This avoids circular dependency between logging and tools packages.
type loggerAdapter struct {
	*logging.Logger
}

// WithTool implements tools.Logger interface.
func (a *loggerAdapter) WithTool(toolName string) tools.Logger {
	return &loggerAdapter{Logger: a.Logger.WithTool(toolName)}
}

// WithSession implements tools.Logger interface.
func (a *loggerAdapter) WithSession(sessionID string) tools.Logger {
	return &loggerAdapter{Logger: a.Logger.WithSession(sessionID)}
}

// Server represents the Claude Code MCP server.
type Server struct {
	mcpServer *mcp.Server
	registry  *tools.Registry
	logger    *logging.Logger
	validator security.Validator
}

// Options configures the server instance.
type Options struct {
	Logger    *logging.Logger
	Validator security.Validator
}

// New creates a new Claude Code MCP server with the given options.
func New(opts *Options) (*Server, error) {
	if opts.Logger == nil {
		logLevel := os.Getenv("LOG_LEVEL")
		if logLevel == "" {
			logLevel = "info"
		}
		opts.Logger = logging.NewLogger(logLevel)
	}

	if opts.Validator == nil {
		opts.Validator = security.NewDefaultValidator()
	}

	toolCtx := &tools.Context{
		Logger:    &loggerAdapter{Logger: opts.Logger},
		Validator: opts.Validator,
	}

	registry := tools.NewRegistry(toolCtx)

	mcpServer := mcp.NewServer(&mcp.Implementation{
		Name:    "claude-code-mcp",
		Version: version.GetVersion().Version,
	}, nil)

	server := &Server{
		mcpServer: mcpServer,
		registry:  registry,
		logger:    opts.Logger,
		validator: opts.Validator,
	}

	if err := server.registerTools(); err != nil {
		return nil, fmt.Errorf("failed to register tools: %w", err)
	}

	return server, nil
}

// Start starts the MCP server.
func (s *Server) Start(ctx context.Context) error {
	s.logger.Info("Starting Claude Code MCP server",
		slog.String("version", version.GetVersion().Version),
		slog.Int("tools", s.registry.Count()),
	)

	if err := s.registry.Validate(); err != nil {
		return fmt.Errorf("tool registry validation failed: %w", err)
	}

	return nil
}

// Stop stops the MCP server gracefully.
func (s *Server) Stop(ctx context.Context) error {
	s.logger.Info("Stopping Claude Code MCP server")

	// TODO: Add cleanup logic for any running operations
	// For now, we just log the stop event

	select {
	case <-ctx.Done():
		s.logger.Warn("Server stop timed out")
		return ctx.Err()
	default:
		s.logger.Info("Server stopped successfully")
		return nil
	}
}

// GetRegistry returns the tool registry.
func (s *Server) GetRegistry() *tools.Registry {
	return s.registry
}

// registerTools registers all Claude Code tools with the server.
func (s *Server) registerTools() error {
	s.logger.Debug("Registering tools with MCP server")

	toolCtx := &tools.Context{
		Logger:    &loggerAdapter{Logger: s.logger},
		Validator: s.validator,
	}

	// Create file operation tools
	fileTools := file.CreateFileTools(toolCtx)

	// Create system operation tools
	bashTools := bash.CreateBashTools(toolCtx)

	// Create notebook operation tools
	notebookTools := notebook.CreateNotebookTools(toolCtx)

	// Create web operation tools
	webTools := web.CreateWebTools(toolCtx)

	// Create todo management tools
	todoTools := todo.CreateTodoTools(toolCtx)

	// Combine all tools
	allTools := collections.Concat(
		fileTools,
		bashTools,
		notebookTools,
		webTools,
		todoTools,
	)

	// Register tools with MCP server
	var toolNames []string
	for _, tool := range allTools {
		// Use the RegisterFunc to register the tool with proper type inference
		tool.RegisterFunc(s.mcpServer)
		toolNames = append(toolNames, tool.Tool.Name)

		s.logger.Debug("Registered tool", "name", tool.Tool.Name)
	}

	s.logger.Info("Successfully registered tools",
		slog.Int("count", len(allTools)),
		slog.Any("tools", toolNames),
	)

	// All core tools are now registered

	return nil
}

// Serve runs the MCP server with the specified transport.
// It connects the MCP server to the transport and waits for either
// the session to complete or the context to be cancelled.
func (s *Server) Serve(ctx context.Context, transport mcp.Transport) error {
	s.logger.Info("Starting MCP server transport",
		slog.String("transport", fmt.Sprintf("%T", transport)),
	)

	// Connect the MCP server to the transport
	session, err := s.mcpServer.Connect(ctx, transport)
	if err != nil {
		return fmt.Errorf("failed to connect MCP server: %w", err)
	}

	// Wait for either the session to finish or context cancellation
	sessionDone := make(chan error, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				s.logger.Error("MCP session goroutine panicked",
					slog.Any("panic", r))
				sessionDone <- fmt.Errorf("session panicked: %v", r)
			}
		}()
		sessionDone <- session.Wait()
	}()

	select {
	case err := <-sessionDone:
		s.logger.Info("MCP session finished")
		return err
	case <-ctx.Done():
		s.logger.Info("MCP server shutting down due to context cancellation")
		return ctx.Err()
	}
}
