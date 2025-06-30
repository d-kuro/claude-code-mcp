package security

import (
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
