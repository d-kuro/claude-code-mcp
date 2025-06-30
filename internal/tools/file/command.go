// Package file provides command execution utilities for file operations.
package file

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// CommandExecutor provides secure command execution with validation and timeouts.
type CommandExecutor struct {
	timeout time.Duration
}

// NewCommandExecutor creates a new command executor with the specified timeout.
func NewCommandExecutor(timeout time.Duration) *CommandExecutor {
	return &CommandExecutor{
		timeout: timeout,
	}
}

// CommandResult represents the result of a command execution.
type CommandResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
	Duration time.Duration
}

// Execute runs a shell command with the specified arguments and returns the result.
func (e *CommandExecutor) Execute(ctx context.Context, name string, args ...string) (*CommandResult, error) {
	start := time.Now()

	// Create context with timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, e.timeout)
	defer cancel()

	// Create command
	cmd := exec.CommandContext(timeoutCtx, name, args...)

	// Set working directory to current directory
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current working directory: %w", err)
	}
	cmd.Dir = cwd

	// Execute command
	stdout, err := cmd.Output()
	stderr := ""
	exitCode := 0

	if err != nil {
		// Handle different types of errors
		if exitError, ok := err.(*exec.ExitError); ok {
			// Command executed but returned non-zero exit code
			stderr = string(exitError.Stderr)
			exitCode = exitError.ExitCode()
		} else {
			// Command failed to execute
			return nil, fmt.Errorf("failed to execute command: %w", err)
		}
	}

	duration := time.Since(start)

	return &CommandResult{
		Stdout:   string(stdout),
		Stderr:   stderr,
		ExitCode: exitCode,
		Duration: duration,
	}, nil
}

// ExecuteInDir runs a command in the specified directory.
func (e *CommandExecutor) ExecuteInDir(ctx context.Context, dir string, name string, args ...string) (*CommandResult, error) {
	start := time.Now()

	// Create context with timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, e.timeout)
	defer cancel()

	// Validate directory
	if !filepath.IsAbs(dir) {
		return nil, fmt.Errorf("directory must be absolute path")
	}

	stat, err := os.Stat(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to stat directory: %w", err)
	}

	if !stat.IsDir() {
		return nil, fmt.Errorf("path is not a directory")
	}

	// Create command
	cmd := exec.CommandContext(timeoutCtx, name, args...)
	cmd.Dir = dir

	// Execute command
	stdout, err := cmd.Output()
	stderr := ""
	exitCode := 0

	if err != nil {
		// Handle different types of errors
		if exitError, ok := err.(*exec.ExitError); ok {
			// Command executed but returned non-zero exit code
			stderr = string(exitError.Stderr)
			exitCode = exitError.ExitCode()
		} else {
			// Command failed to execute
			return nil, fmt.Errorf("failed to execute command: %w", err)
		}
	}

	duration := time.Since(start)

	return &CommandResult{
		Stdout:   string(stdout),
		Stderr:   stderr,
		ExitCode: exitCode,
		Duration: duration,
	}, nil
}

// ValidateCommand performs basic validation on command name and arguments.
func (e *CommandExecutor) ValidateCommand(name string, args []string) error {
	// Check if command name is empty
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("command name cannot be empty")
	}

	// Check for dangerous characters
	dangerousChars := []string{";", "&", "|", ">", "<", "`", "$", "(", ")", "{", "}", "[", "]"}
	for _, char := range dangerousChars {
		if strings.Contains(name, char) {
			return fmt.Errorf("command name contains dangerous character: %s", char)
		}
		for _, arg := range args {
			if strings.Contains(arg, char) && !isAllowedCharInArg(arg, char) {
				return fmt.Errorf("argument contains dangerous character: %s", char)
			}
		}
	}

	return nil
}

// isAllowedCharInArg checks if a character is allowed in a specific argument context.
func isAllowedCharInArg(arg, char string) bool {
	// Allow some characters in specific contexts
	// For example, { and } are allowed in glob patterns
	switch char {
	case "{", "}", "[", "]":
		// Allow in glob patterns
		if strings.Contains(arg, "*") || strings.Contains(arg, "?") {
			return true
		}
	case ">", "<":
		// Never allow redirection operators
		return false
	case "$":
		// Never allow variable expansion
		return false
	case "`":
		// Never allow command substitution
		return false
	}

	return false
}

// FindBinary searches for a binary in the system PATH.
func FindBinary(name string) (string, error) {
	path, err := exec.LookPath(name)
	if err != nil {
		return "", fmt.Errorf("binary %s not found in PATH: %w", name, err)
	}
	return path, nil
}
