package bash

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestShellExecutor_ValidateCommand(t *testing.T) {
	executor := NewShellExecutor()

	tests := []struct {
		name    string
		command string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid simple command",
			command: "echo hello",
			wantErr: false,
		},
		{
			name:    "valid complex command",
			command: "ls -la | grep test",
			wantErr: false,
		},
		{
			name:    "empty command",
			command: "",
			wantErr: true,
			errMsg:  "command cannot be empty",
		},
		{
			name:    "whitespace only command",
			command: "   \n\t  ",
			wantErr: true,
			errMsg:  "command cannot be empty",
		},
		{
			name:    "dangerous rm command",
			command: "rm -rf /",
			wantErr: true,
			errMsg:  "dangerous pattern",
		},
		{
			name:    "fork bomb",
			command: ":(){ :|:& };:",
			wantErr: true,
			errMsg:  "dangerous pattern",
		},
		{
			name:    "dangerous dd command",
			command: "dd if=/dev/zero of=/dev/sda",
			wantErr: true,
			errMsg:  "dangerous pattern",
		},
		{
			name:    "mkfs command",
			command: "mkfs.ext4 /dev/sdb1",
			wantErr: true,
			errMsg:  "dangerous pattern",
		},
		{
			name:    "fdisk command",
			command: "fdisk /dev/sda",
			wantErr: true,
			errMsg:  "dangerous pattern",
		},
		{
			name:    "case insensitive dangerous command",
			command: "RM -RF /",
			wantErr: true,
			errMsg:  "dangerous pattern",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := executor.ValidateCommand(tt.command)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidateCommand() expected error but got none")
				} else if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("ValidateCommand() error = %v, want error containing %q", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateCommand() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestShellExecutor_ExecuteInSession_BasicCommands(t *testing.T) {
	executor := NewShellExecutor()
	session := createTestSession()

	tests := []struct {
		name          string
		command       string
		wantExitCode  int
		wantStdoutSub string
		wantStderrSub string
		expectError   bool
	}{
		{
			name:          "echo command",
			command:       "echo hello world",
			wantExitCode:  0,
			wantStdoutSub: "hello world",
		},
		{
			name:          "pwd command",
			command:       "pwd",
			wantExitCode:  0,
			wantStdoutSub: "/",
		},
		{
			name:          "ls root directory",
			command:       "ls /",
			wantExitCode:  0,
			wantStdoutSub: "",
		},
		{
			name:          "false command",
			command:       "false",
			wantExitCode:  1,
			wantStdoutSub: "",
		},
		{
			name:          "command with stderr",
			command:       "bash -c \"echo 'to stderr' >&2\"",
			wantExitCode:  0,
			wantStderrSub: "to stderr",
		},
		{
			name:         "nonexistent command",
			command:      "nonexistent_command_xyz",
			expectError:  false, // Should not error, but should have non-zero exit code
			wantExitCode: 127,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			result, err := executor.ExecuteInSession(ctx, session, tt.command, 10*time.Second)

			if tt.expectError {
				if err == nil {
					t.Errorf("ExecuteInSession() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("ExecuteInSession() unexpected error = %v", err)
			}

			if result.ExitCode != tt.wantExitCode {
				t.Errorf("ExecuteInSession() exitCode = %v, want %v", result.ExitCode, tt.wantExitCode)
			}

			if tt.wantStdoutSub != "" && !strings.Contains(result.Stdout, tt.wantStdoutSub) {
				t.Errorf("ExecuteInSession() stdout = %q, want to contain %q", result.Stdout, tt.wantStdoutSub)
			}

			if tt.wantStderrSub != "" && !strings.Contains(result.Stderr, tt.wantStderrSub) {
				t.Errorf("ExecuteInSession() stderr = %q, want to contain %q", result.Stderr, tt.wantStderrSub)
			}

			if result.Duration <= 0 {
				t.Errorf("ExecuteInSession() duration should be positive, got %v", result.Duration)
			}

			if result.WorkingDirectory == "" {
				t.Errorf("ExecuteInSession() working directory should not be empty")
			}
		})
	}
}

func TestShellExecutor_ExecuteInSession_Timeout(t *testing.T) {
	executor := NewShellExecutor()
	session := createTestSession()

	if session == nil {
		t.Fatal("createTestSession returned nil")
	}

	ctx := context.Background()
	_, err := executor.ExecuteInSession(ctx, session, "sleep 5", 100*time.Millisecond)

	if err == nil {
		t.Error("ExecuteInSession() expected timeout error but got none")
		return
	}

	if !strings.Contains(err.Error(), "timed out") {
		t.Errorf("ExecuteInSession() error = %v, want timeout error", err)
	}
}

func TestShellExecutor_ExecuteInSession_ContextCancellation(t *testing.T) {
	executor := NewShellExecutor()
	session := createTestSession()

	ctx, cancel := context.WithCancel(context.Background())

	// Cancel context immediately
	cancel()

	_, err := executor.ExecuteInSession(ctx, session, "echo test", 5*time.Second)

	if err == nil {
		t.Error("ExecuteInSession() expected context cancellation error but got none")
	}
}

func TestShellExecutor_HandleCdCommand(t *testing.T) {
	executor := NewShellExecutor()

	// Create a temporary directory for testing
	tempDir := t.TempDir()
	subDir := filepath.Join(tempDir, "subdir")
	err := os.Mkdir(subDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	tests := []struct {
		name        string
		command     string
		initialDir  string
		expectedDir string
		expectError bool
	}{
		{
			name:        "cd to subdirectory",
			command:     "cd " + subDir,
			initialDir:  tempDir,
			expectedDir: subDir,
			expectError: false,
		},
		{
			name:        "cd with quotes",
			command:     `cd "` + subDir + `"`,
			initialDir:  tempDir,
			expectedDir: subDir,
			expectError: false,
		},
		{
			name:        "cd with single quotes",
			command:     `cd '` + subDir + `'`,
			initialDir:  tempDir,
			expectedDir: subDir,
			expectError: false,
		},
		{
			name:        "cd to relative directory",
			command:     "cd subdir",
			initialDir:  tempDir,
			expectedDir: subDir,
			expectError: false,
		},
		{
			name:        "cd to parent directory",
			command:     "cd ..",
			initialDir:  subDir,
			expectedDir: tempDir,
			expectError: false,
		},
		{
			name:        "cd to home directory",
			command:     "cd",
			initialDir:  tempDir,
			expectedDir: "", // Will be set to home directory
			expectError: false,
		},
		{
			name:        "cd to nonexistent directory",
			command:     "cd /nonexistent/directory",
			initialDir:  tempDir,
			expectError: true,
		},
		{
			name:        "cd to file (not directory)",
			command:     "cd " + filepath.Join(tempDir, "file.txt"),
			initialDir:  tempDir,
			expectError: true,
		},
	}

	// Create a test file for the "cd to file" test
	testFile := filepath.Join(tempDir, "file.txt")
	err = os.WriteFile(testFile, []byte("test"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session := &ShellSession{
				ID:               "test",
				WorkingDirectory: tt.initialDir,
				Environment:      make(map[string]string),
				CreatedAt:        time.Now(),
				LastUsed:         time.Now(),
			}

			err := executor.handleCdCommand(session, tt.command)

			if tt.expectError {
				if err == nil {
					t.Errorf("handleCdCommand() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("handleCdCommand() unexpected error = %v", err)
				return
			}

			expectedDir := tt.expectedDir
			if expectedDir == "" && tt.command == "cd" {
				// For home directory test
				homeDir, _ := os.UserHomeDir()
				expectedDir = homeDir
			}

			if session.WorkingDirectory != expectedDir {
				t.Errorf("handleCdCommand() working directory = %q, want %q", session.WorkingDirectory, expectedDir)
			}
		})
	}
}

func TestShellExecutor_HandleExportCommand(t *testing.T) {
	executor := NewShellExecutor()

	tests := []struct {
		name        string
		command     string
		expectError bool
		expectedVar string
		expectedVal string
	}{
		{
			name:        "export with value",
			command:     "export TEST_VAR=hello",
			expectError: false,
			expectedVar: "TEST_VAR",
			expectedVal: "hello",
		},
		{
			name:        "export with quoted value",
			command:     `export TEST_VAR="hello world"`,
			expectError: false,
			expectedVar: "TEST_VAR",
			expectedVal: "hello world",
		},
		{
			name:        "export with single quoted value",
			command:     `export TEST_VAR='hello world'`,
			expectError: false,
			expectedVar: "TEST_VAR",
			expectedVal: "hello world",
		},
		{
			name:        "export with spaces around equals",
			command:     "export TEST_VAR = hello",
			expectError: false,
			expectedVar: "TEST_VAR",
			expectedVal: "hello",
		},
		{
			name:        "export existing environment variable",
			command:     "export PATH",
			expectError: false,
			expectedVar: "PATH",
			expectedVal: "", // Will be set to actual PATH value
		},
		{
			name:        "export nonexistent variable",
			command:     "export NONEXISTENT_VAR_XYZ",
			expectError: false,
			expectedVar: "",
			expectedVal: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use isolated environment for each test
			session := &ShellSession{
				ID:          "test",
				Environment: make(map[string]string),
				CreatedAt:   time.Now(),
				LastUsed:    time.Now(),
			}

			err := executor.handleExportCommand(session, tt.command)

			if tt.expectError {
				if err == nil {
					t.Errorf("handleExportCommand() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("handleExportCommand() unexpected error = %v", err)
				return
			}

			if tt.expectedVar != "" {
				value, exists := session.Environment[tt.expectedVar]
				if !exists {
					t.Errorf("handleExportCommand() variable %q not found in environment", tt.expectedVar)
					return
				}

				if tt.expectedVar == "PATH" {
					// For PATH, just check that it's not empty and contains system PATH
					if value == "" {
						t.Errorf("handleExportCommand() PATH should not be empty")
					}
					// Verify it contains some expected system paths
					if !strings.Contains(value, "/bin") && !strings.Contains(value, "/usr/bin") {
						t.Errorf("handleExportCommand() PATH should contain system paths, got: %s", value)
					}
				} else if value != tt.expectedVal {
					t.Errorf("handleExportCommand() variable %q = %q, want %q", tt.expectedVar, value, tt.expectedVal)
				}
			}
		})
	}
}

func TestShellExecutor_PersistentState(t *testing.T) {
	executor := NewShellExecutor()
	session := createTestSession()

	ctx := context.Background()

	// Test that environment variables persist
	_, err := executor.ExecuteInSession(ctx, session, "export PERSISTENT_VAR=persistent_value", 5*time.Second)
	if err != nil {
		t.Fatalf("Export command failed: %v", err)
	}

	result, err := executor.ExecuteInSession(ctx, session, "echo $PERSISTENT_VAR", 5*time.Second)
	if err != nil {
		t.Fatalf("Echo command failed: %v", err)
	}

	if !strings.Contains(result.Stdout, "persistent_value") {
		t.Errorf("Environment variable not persisted: %q", result.Stdout)
	}

	// Test that working directory persists
	tempDir := t.TempDir()
	_, err = executor.ExecuteInSession(ctx, session, "cd "+tempDir, 5*time.Second)
	if err != nil {
		t.Fatalf("CD command failed: %v", err)
	}

	result, err = executor.ExecuteInSession(ctx, session, "pwd", 5*time.Second)
	if err != nil {
		t.Fatalf("PWD command failed: %v", err)
	}

	if !strings.Contains(result.Stdout, tempDir) {
		t.Errorf("Working directory not persisted: %q, expected %q", result.Stdout, tempDir)
	}
}

func TestShellExecutor_EnvironmentIsolation(t *testing.T) {
	executor := NewShellExecutor()
	session := createTestSession()

	ctx := context.Background()

	// Set environment variable in session
	_, err := executor.ExecuteInSession(ctx, session, "export ISOLATED_VAR=session_value", 5*time.Second)
	if err != nil {
		t.Fatalf("Export command failed: %v", err)
	}

	// Verify it doesn't leak to process environment
	if value := os.Getenv("ISOLATED_VAR"); value != "" {
		t.Errorf("Session environment leaked to process: %q", value)
	}

	// Verify it exists in session
	result, err := executor.ExecuteInSession(ctx, session, "echo $ISOLATED_VAR", 5*time.Second)
	if err != nil {
		t.Fatalf("Echo command failed: %v", err)
	}

	if !strings.Contains(result.Stdout, "session_value") {
		t.Errorf("Environment variable not found in session: %q", result.Stdout)
	}
}

func TestShellExecutor_LongOutput(t *testing.T) {
	executor := NewShellExecutor()
	session := createTestSession()

	ctx := context.Background()

	// Generate long output (use printf for proper variable expansion)
	command := `for i in {1..1000}; do printf "Line %d with some additional text to make it longer\n" $i; done`
	result, err := executor.ExecuteInSession(ctx, session, command, 10*time.Second)

	if err != nil {
		t.Fatalf("Long output command failed: %v", err)
	}

	if result.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", result.ExitCode)
	}

	if len(result.Stdout) < 10000 { // Should be much longer than this
		t.Errorf("Output seems too short for long command: %d characters", len(result.Stdout))
	}

	// Verify it contains expected content
	if !strings.Contains(result.Stdout, "Line 1 with") {
		t.Error("Output should contain first line")
	}

	if !strings.Contains(result.Stdout, "Line 1000 with") {
		t.Error("Output should contain last line")
	}
}

func TestShellExecutor_BinaryOutput(t *testing.T) {
	executor := NewShellExecutor()
	session := createTestSession()

	ctx := context.Background()

	// Generate binary output (null bytes)
	command := "printf '\\x00\\x01\\x02\\x03\\xFF'"
	result, err := executor.ExecuteInSession(ctx, session, command, 5*time.Second)

	if err != nil {
		t.Fatalf("Binary output command failed: %v", err)
	}

	if result.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", result.ExitCode)
	}

	// Should handle binary data gracefully
	if len(result.Stdout) != 5 {
		t.Errorf("Expected 5 bytes of output, got %d", len(result.Stdout))
	}
}

func TestShellExecutor_SpecialCharacters(t *testing.T) {
	executor := NewShellExecutor()
	session := createTestSession()

	ctx := context.Background()

	tests := []struct {
		name     string
		command  string
		expected string
	}{
		{
			name:     "unicode characters",
			command:  "echo '🚀 Hello 世界'",
			expected: "🚀 Hello 世界",
		},
		{
			name:     "special shell characters",
			command:  "echo 'Special: $HOME | & ; ( ) [ ] { }'",
			expected: "Special: $HOME | & ; ( ) [ ] { }",
		},
		{
			name:     "newlines and tabs",
			command:  "printf 'Line1\\nLine2\\tTabbed'",
			expected: "Line1\nLine2\tTabbed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := executor.ExecuteInSession(ctx, session, tt.command, 5*time.Second)
			if err != nil {
				t.Fatalf("Command failed: %v", err)
			}

			if !strings.Contains(result.Stdout, tt.expected) {
				t.Errorf("Output %q should contain %q", result.Stdout, tt.expected)
			}
		})
	}
}

func TestShellExecutor_CommandMightChangeDirectory(t *testing.T) {
	executor := NewShellExecutor()

	tests := []struct {
		command  string
		expected bool
	}{
		{"cd /tmp", true},
		{"cd", true},
		{"pushd /tmp", true},
		{"popd", true},
		{"echo hello", false},
		{"ls -la", false},
		{"pwd", false},
		{"  cd  /tmp  ", true}, // with whitespace
		{"echo cd", false},     // cd as part of other command
		{"cd_function", false}, // command starting with cd but not cd
	}

	for _, tt := range tests {
		t.Run(tt.command, func(t *testing.T) {
			result := executor.commandMightChangeDirectory(tt.command)
			if result != tt.expected {
				t.Errorf("commandMightChangeDirectory(%q) = %v, want %v", tt.command, result, tt.expected)
			}
		})
	}
}

func TestShellExecutor_UpdateWorkingDirectoryFromPwd(t *testing.T) {
	executor := NewShellExecutor()

	tempDir := t.TempDir()
	session := &ShellSession{
		ID:               "test",
		WorkingDirectory: tempDir,
		Environment:      make(map[string]string),
		CreatedAt:        time.Now(),
		LastUsed:         time.Now(),
	}

	err := executor.updateWorkingDirectoryFromPwd(session)
	if err != nil {
		t.Fatalf("updateWorkingDirectoryFromPwd failed: %v", err)
	}

	// Working directory should be updated to the actual pwd output
	if session.WorkingDirectory == "" {
		t.Error("Working directory should not be empty after update")
	}

	// Should be an absolute path
	if !filepath.IsAbs(session.WorkingDirectory) {
		t.Errorf("Working directory should be absolute path: %q", session.WorkingDirectory)
	}
}

func TestShellExecutor_ComplexCommands(t *testing.T) {
	executor := NewShellExecutor()
	session := createTestSession()

	ctx := context.Background()

	tests := []struct {
		name    string
		command string
	}{
		{
			name:    "pipeline",
			command: "echo 'hello\nworld\nhello' | grep hello | wc -l",
		},
		{
			name:    "command substitution",
			command: "echo \"Current date: $(date)\"",
		},
		{
			name:    "conditional execution",
			command: "true && echo 'success' || echo 'failed'",
		},
		{
			name:    "variable expansion",
			command: "VAR=test; echo \"Variable: $VAR\"",
		},
		{
			name:    "background process (wait)",
			command: "sleep 0.1 & wait",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := executor.ExecuteInSession(ctx, session, tt.command, 10*time.Second)
			if err != nil {
				t.Fatalf("Complex command failed: %v", err)
			}

			if result.ExitCode != 0 {
				t.Errorf("Expected exit code 0, got %d. Stderr: %s", result.ExitCode, result.Stderr)
			}
		})
	}
}

func TestShellExecutor_SecurityInjectionAttempts(t *testing.T) {
	executor := NewShellExecutor()

	// These should be caught by ValidateCommand
	dangerousCommands := []string{
		"rm -rf /",
		":(){ :|:& };:",
		"dd if=/dev/zero of=/dev/sda",
		"mkfs.ext4 /dev/sdb1",
		"fdisk /dev/sda",
	}

	for _, cmd := range dangerousCommands {
		t.Run("dangerous_"+cmd, func(t *testing.T) {
			err := executor.ValidateCommand(cmd)
			if err == nil {
				t.Errorf("ValidateCommand should have rejected dangerous command: %q", cmd)
			}
		})
	}

	// Test command injection attempts that should be safe
	session := createTestSession()
	ctx := context.Background()

	// Commands that contain dangerous patterns in any form (even quoted) should be rejected
	// This is a conservative security approach
	potentiallyDangerousCommands := []string{
		"echo 'rm -rf /'",           // quoted dangerous command - still rejected for security
		"VAR='rm -rf /'; echo $VAR", // variable containing dangerous command - still rejected
	}

	safeCommands := []string{
		"echo hello; echo world", // command chaining without dangerous patterns
	}

	// Test that potentially dangerous commands are rejected
	for _, cmd := range potentiallyDangerousCommands {
		t.Run("dangerous_quoted_"+cmd, func(t *testing.T) {
			// These should be rejected (conservative security approach)
			err := executor.ValidateCommand(cmd)
			if err == nil {
				t.Errorf("ValidateCommand should reject potentially dangerous command: %q", cmd)
			}
		})
	}

	// Test that truly safe commands are allowed
	for _, cmd := range safeCommands {
		t.Run("safe_injection_"+cmd, func(t *testing.T) {
			// These should pass validation (they're not directly dangerous)
			err := executor.ValidateCommand(cmd)
			if err != nil {
				t.Errorf("ValidateCommand should not reject safe command: %q, error: %v", cmd, err)
			}

			// And should execute successfully
			result, err := executor.ExecuteInSession(ctx, session, cmd, 5*time.Second)
			if err != nil {
				t.Fatalf("Safe command execution failed: %v", err)
			}

			if result.ExitCode != 0 {
				t.Errorf("Safe command should succeed, got exit code %d", result.ExitCode)
			}
		})
	}
}

func TestShellExecutor_ResourceCleanup(t *testing.T) {
	executor := NewShellExecutor()
	session := createTestSession()

	ctx := context.Background()

	// Test that processes are properly cleaned up on timeout
	ctxWithTimeout, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()

	_, err := executor.ExecuteInSession(ctxWithTimeout, session, "sleep 5", 100*time.Millisecond)

	// Should timeout
	if err == nil {
		t.Error("Expected timeout error")
	}

	// Give a moment for cleanup
	time.Sleep(50 * time.Millisecond)

	// Process should be terminated - we can't easily test this without
	// external tools, but the timeout should have handled it
}

// Helper function to create a test session
func createTestSession() *ShellSession {
	cwd, _ := os.Getwd()
	return &ShellSession{
		ID:               "test",
		WorkingDirectory: cwd,
		Environment:      make(map[string]string),
		CreatedAt:        time.Now(),
		LastUsed:         time.Now(),
		AccessCount:      0,
	}
}
