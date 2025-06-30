package file

import (
	"context"
	"testing"
	"time"
)

func TestCommandExecutor(t *testing.T) {
	executor := NewCommandExecutor(5 * time.Second)

	tests := []struct {
		name        string
		command     string
		args        []string
		expectError bool
	}{
		{
			name:        "valid echo command",
			command:     "echo",
			args:        []string{"hello", "world"},
			expectError: false,
		},
		{
			name:        "invalid command",
			command:     "nonexistent-command-xyz",
			args:        []string{},
			expectError: true,
		},
		{
			name:        "ls command",
			command:     "ls",
			args:        []string{"-1", "/"},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := executor.Execute(context.Background(), tt.command, tt.args...)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if result == nil {
				t.Errorf("expected result but got nil")
				return
			}

			if result.Duration == 0 {
				t.Errorf("expected non-zero duration")
			}
		})
	}
}

func TestCommandValidation(t *testing.T) {
	executor := NewCommandExecutor(5 * time.Second)

	tests := []struct {
		name        string
		command     string
		args        []string
		expectError bool
	}{
		{
			name:        "valid command",
			command:     "ls",
			args:        []string{"-l"},
			expectError: false,
		},
		{
			name:        "empty command",
			command:     "",
			args:        []string{},
			expectError: true,
		},
		{
			name:        "command with dangerous characters",
			command:     "ls;rm",
			args:        []string{},
			expectError: true,
		},
		{
			name:        "args with dangerous characters",
			command:     "ls",
			args:        []string{"-l", "file;rm -rf /"},
			expectError: true,
		},
		{
			name:        "valid glob pattern in args",
			command:     "find",
			args:        []string{"-name", "*.go"},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := executor.ValidateCommand(tt.command, tt.args)

			if tt.expectError && err == nil {
				t.Errorf("expected error but got none")
			}

			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestFindBinary(t *testing.T) {
	tests := []struct {
		name        string
		binary      string
		expectError bool
	}{
		{
			name:        "ls binary",
			binary:      "ls",
			expectError: false,
		},
		{
			name:        "echo binary",
			binary:      "echo",
			expectError: false,
		},
		{
			name:        "nonexistent binary",
			binary:      "nonexistent-binary-xyz",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, err := FindBinary(tt.binary)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if path == "" {
				t.Errorf("expected non-empty path")
			}
		})
	}
}
