// Package bash provides shell command execution with persistent state.
package bash

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// ShellExecutor handles execution of shell commands with persistent session state.
type ShellExecutor struct{}

// NewShellExecutor creates a new shell executor.
func NewShellExecutor() *ShellExecutor {
	return &ShellExecutor{}
}

// ExecuteInSession executes a command within a persistent session context.
func (e *ShellExecutor) ExecuteInSession(ctx context.Context, session *ShellSession, command string, timeout time.Duration) (*CommandResult, error) {
	start := time.Now()

	// Create context with timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Parse the command to handle session state changes
	if err := e.preprocessCommand(session, command); err != nil {
		return nil, fmt.Errorf("command preprocessing failed: %w", err)
	}

	// Execute the command
	result, err := e.executeCommand(timeoutCtx, session, command)
	if err != nil {
		// Check for timeout first, before checking other error types
		if timeoutCtx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("command timed out after %v", timeout)
		}
		return nil, err
	}

	// Also check for timeout in case the command completed but the context was cancelled
	if timeoutCtx.Err() == context.DeadlineExceeded {
		return nil, fmt.Errorf("command timed out after %v", timeout)
	}

	// Update session state based on command execution
	if err := e.postprocessCommand(session, command, result); err != nil {
		// Log warning but don't fail the command
		// In a real implementation, this would use the logger from context
		fmt.Fprintf(os.Stderr, "Warning: session state update failed: %v\n", err)
	}

	result.Duration = time.Since(start)
	result.WorkingDirectory = session.WorkingDirectory

	return result, nil
}

// preprocessCommand handles commands that change session state before execution.
func (e *ShellExecutor) preprocessCommand(session *ShellSession, command string) error {
	trimmedCmd := strings.TrimSpace(command)

	// Handle cd commands to update working directory
	if strings.HasPrefix(trimmedCmd, "cd ") || trimmedCmd == "cd" {
		return e.handleCdCommand(session, trimmedCmd)
	}

	// Handle export commands to update environment
	if strings.HasPrefix(trimmedCmd, "export ") {
		return e.handleExportCommand(session, trimmedCmd)
	}

	return nil
}

// postprocessCommand handles session state updates after command execution.
func (e *ShellExecutor) postprocessCommand(session *ShellSession, command string, result *CommandResult) error {
	// If command was successful and might have changed working directory
	if result.ExitCode == 0 {
		// For certain commands, verify and update the working directory
		if e.commandMightChangeDirectory(command) {
			return e.updateWorkingDirectoryFromPwd(session)
		}
	}

	return nil
}

// executeCommand executes the actual shell command.
func (e *ShellExecutor) executeCommand(ctx context.Context, session *ShellSession, command string) (*CommandResult, error) {
	// Use bash as the shell for consistent behavior
	cmd := exec.CommandContext(ctx, "/bin/bash", "-c", command)

	// Set working directory
	cmd.Dir = session.WorkingDirectory

	// Set environment variables
	env := os.Environ()
	for key, value := range session.Environment {
		env = append(env, fmt.Sprintf("%s=%s", key, value))
	}
	cmd.Env = env

	// Execute command and capture both stdout and stderr
	stdout, stderr, err := e.runCommand(cmd)
	exitCode := 0

	if err != nil {
		// Check for context cancellation/timeout first
		if ctx.Err() == context.DeadlineExceeded {
			// Command timed out
			return nil, fmt.Errorf("command timed out")
		}
		// Handle different types of errors
		if exitError, ok := err.(*exec.ExitError); ok {
			// Command executed but returned non-zero exit code
			exitCode = exitError.ExitCode()
		} else {
			// Command failed to execute
			return nil, fmt.Errorf("failed to execute command: %w", err)
		}
	}

	return &CommandResult{
		Stdout:   stdout,
		Stderr:   stderr,
		ExitCode: exitCode,
	}, nil
}

// runCommand runs the command and captures both stdout and stderr separately.
func (e *ShellExecutor) runCommand(cmd *exec.Cmd) (stdout, stderr string, err error) {
	var stdoutBuf, stderrBuf strings.Builder
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	err = cmd.Run()
	stdout = stdoutBuf.String()
	stderr = stderrBuf.String()
	return
}

// handleCdCommand processes cd commands to update session working directory.
func (e *ShellExecutor) handleCdCommand(session *ShellSession, command string) error {
	parts := strings.Fields(command)

	var targetDir string
	if len(parts) == 1 {
		// cd with no arguments goes to home directory
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		targetDir = homeDir
	} else {
		// cd with directory argument
		targetDir = strings.Join(parts[1:], " ")

		// Remove quotes if present
		if (strings.HasPrefix(targetDir, "\"") && strings.HasSuffix(targetDir, "\"")) ||
			(strings.HasPrefix(targetDir, "'") && strings.HasSuffix(targetDir, "'")) {
			targetDir = targetDir[1 : len(targetDir)-1]
		}
	}

	// Convert relative path to absolute
	if !filepath.IsAbs(targetDir) {
		targetDir = filepath.Join(session.WorkingDirectory, targetDir)
	}

	// Clean the path
	targetDir = filepath.Clean(targetDir)

	// Verify directory exists
	stat, err := os.Stat(targetDir)
	if err != nil {
		return fmt.Errorf("directory does not exist: %s", targetDir)
	}

	if !stat.IsDir() {
		return fmt.Errorf("not a directory: %s", targetDir)
	}

	// Update session working directory
	session.WorkingDirectory = targetDir

	return nil
}

// handleExportCommand processes export commands to update session environment.
func (e *ShellExecutor) handleExportCommand(session *ShellSession, command string) error {
	// Parse export command: export VAR=value or export VAR="value"
	command = strings.TrimPrefix(command, "export ")
	command = strings.TrimSpace(command)

	// Find the = sign
	eqIndex := strings.Index(command, "=")
	if eqIndex == -1 {
		// export VAR (without value) - export existing environment variable
		varName := strings.TrimSpace(command)
		if value, exists := os.LookupEnv(varName); exists {
			session.Environment[varName] = value
		}
		return nil
	}

	// export VAR=value
	varName := strings.TrimSpace(command[:eqIndex])
	varValue := strings.TrimSpace(command[eqIndex+1:])

	// Remove quotes if present
	if (strings.HasPrefix(varValue, "\"") && strings.HasSuffix(varValue, "\"")) ||
		(strings.HasPrefix(varValue, "'") && strings.HasSuffix(varValue, "'")) {
		varValue = varValue[1 : len(varValue)-1]
	}

	// Update session environment
	session.Environment[varName] = varValue

	return nil
}

// commandMightChangeDirectory checks if a command might change the working directory.
func (e *ShellExecutor) commandMightChangeDirectory(command string) bool {
	trimmedCmd := strings.TrimSpace(command)

	// Commands that might change directory
	changeDirectoryCommands := []string{
		"cd ", "pushd ", "popd",
	}

	for _, cdCmd := range changeDirectoryCommands {
		if strings.HasPrefix(trimmedCmd, cdCmd) || trimmedCmd == strings.TrimSpace(cdCmd) {
			return true
		}
	}

	return false
}

// updateWorkingDirectoryFromPwd updates session working directory by running pwd.
func (e *ShellExecutor) updateWorkingDirectoryFromPwd(session *ShellSession) error {
	// Create a simple context for pwd command
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "/bin/bash", "-c", "pwd")
	cmd.Dir = session.WorkingDirectory

	// Set environment
	env := os.Environ()
	for key, value := range session.Environment {
		env = append(env, fmt.Sprintf("%s=%s", key, value))
	}
	cmd.Env = env

	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get current directory with pwd: %w", err)
	}

	newWorkingDir := strings.TrimSpace(string(output))
	if newWorkingDir != "" {
		session.WorkingDirectory = newWorkingDir
	}

	return nil
}

// ValidateCommand performs basic validation on the command.
func (e *ShellExecutor) ValidateCommand(command string) error {
	if strings.TrimSpace(command) == "" {
		return fmt.Errorf("command cannot be empty")
	}

	// Check for dangerous patterns
	dangerousPatterns := []string{
		"rm -rf /",
		":(){ :|:& };:",   // Fork bomb
		"dd if=/dev/zero", // Dangerous dd usage
		"mkfs",            // Filesystem creation
		"fdisk",           // Disk partitioning
	}

	lowerCmd := strings.ToLower(command)
	for _, pattern := range dangerousPatterns {
		if strings.Contains(lowerCmd, pattern) {
			return fmt.Errorf("command contains dangerous pattern: %s", pattern)
		}
	}

	return nil
}
