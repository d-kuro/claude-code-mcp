// Package main implements the Claude Code MCP server executable.
// It provides a Model Context Protocol server that exposes Claude Code's
// built-in tools as MCP tools for external applications.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"

	"github.com/d-kuro/claude-code-mcp/internal/logging"
	"github.com/d-kuro/claude-code-mcp/internal/server"
	"github.com/d-kuro/claude-code-mcp/pkg/version"
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "claude-code-mcp",
	Short: "Claude Code MCP server",
	Long: `Claude Code MCP server provides a Model Context Protocol server that exposes 
Claude Code's built-in tools as MCP tools for external applications.`,
	RunE: runServer,
}

// serverFlags holds the flags for the server command
type serverFlags struct {
	httpAddr string
}

var serverOpts = &serverFlags{}

func init() {
	// Add version flag
	rootCmd.Flags().BoolP("version", "v", false, "Print version information and exit")

	// Add server flags
	rootCmd.Flags().StringVar(&serverOpts.httpAddr, "http", "", "HTTP server address (e.g., :8080)")

	// Add subcommands
	rootCmd.AddCommand(googleLoginCmd)
	rootCmd.AddCommand(googleLogoutCmd)
	rootCmd.AddCommand(googleStatusCmd)
}

// runServer starts the MCP server
func runServer(cmd *cobra.Command, args []string) error {
	// Check version flag
	if versionFlag, _ := cmd.Flags().GetBool("version"); versionFlag {
		fmt.Println(version.GetVersion().String())
		return nil
	}

	// Get log level from environment variable, default to "info"
	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info"
	}

	// Initialize logger with log level
	logger := logging.NewLogger(logLevel)

	opts := &server.Options{}

	srv, err := server.New(opts)
	if err != nil {
		logger.Error("Failed to create server", slog.Any("error", err))
		return fmt.Errorf("failed to create server: %w", err)
	}

	// Set up signal handling
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	if err := srv.Start(ctx); err != nil {
		logger.Error("Failed to start server", slog.Any("error", err))
		return fmt.Errorf("failed to start server: %w", err)
	}

	var transport mcp.Transport
	if serverOpts.httpAddr != "" {
		// TODO: Implement HTTP/SSE transport
		logger.Warn("HTTP transport not yet implemented, using stdio",
			slog.String("requested_addr", serverOpts.httpAddr))
		transport = mcp.NewStdioTransport()
	} else {
		transport = mcp.NewStdioTransport()
	}

	logger.Info("Claude Code MCP Server starting",
		slog.String("version", version.GetVersion().Version),
		slog.String("transport", fmt.Sprintf("%T", transport)),
		slog.Int("tools_available", srv.GetRegistry().Count()))

	// Start server in a goroutine so we can handle signals
	serverDone := make(chan error, 1)
	go func() {
		serverDone <- srv.Serve(ctx, transport)
	}()

	// Wait for either the server to finish or a signal
	select {
	case err := <-serverDone:
		if err != nil && !errors.Is(err, context.Canceled) {
			logger.Error("Server error", slog.Any("error", err))
		}
	case <-ctx.Done():
		logger.Info("Shutdown signal received")
	}

	// Create a new context for shutdown with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := srv.Stop(shutdownCtx); err != nil {
		logger.Error("Error stopping server", slog.Any("error", err))
	}

	logger.Info("Claude Code MCP Server stopped")
	return nil
}
