// Package security provides security validation and sandboxing functionality.
package security

import (
	"net/url"
	"path/filepath"
	"strings"

	"github.com/d-kuro/claude-code-mcp/internal/errors"
)

// Validator defines the security validation interface.
type Validator interface {
	ValidatePath(path string) error
	ValidateCommand(cmd string, args []string) error
	ValidateURL(urlStr string) error
	SanitizePath(path string) (string, error)
}

// DefaultValidator provides default security validation implementation.
type DefaultValidator struct {
	allowedPaths    []string
	blockedPaths    []string
	allowedCommands []string
	blockedCommands []string
}

// NewDefaultValidator creates a new default validator with secure defaults.
func NewDefaultValidator() *DefaultValidator {
	return &DefaultValidator{
		allowedPaths: []string{},
		blockedPaths: []string{
			"/etc",
			"/usr/bin",
			"/usr/sbin",
			"/sbin",
			"/bin",
			"/sys",
			"/proc",
		},
		allowedCommands: []string{},
		blockedCommands: []string{
			"sudo",
			"su",
			"chmod",
			"chown",
			"rm",
			"rmdir",
			"dd",
			"mkfs",
			"fdisk",
			"mount",
			"umount",
		},
	}
}

// WithAllowedPaths sets the allowed paths for file operations.
func (v *DefaultValidator) WithAllowedPaths(paths []string) *DefaultValidator {
	v.allowedPaths = make([]string, len(paths))
	copy(v.allowedPaths, paths)
	return v
}

// WithBlockedPaths adds blocked paths to the default list.
func (v *DefaultValidator) WithBlockedPaths(paths []string) *DefaultValidator {
	v.blockedPaths = append(v.blockedPaths, paths...)
	return v
}

// WithAllowedCommands sets the allowed commands for execution.
func (v *DefaultValidator) WithAllowedCommands(commands []string) *DefaultValidator {
	v.allowedCommands = make([]string, len(commands))
	copy(v.allowedCommands, commands)
	return v
}

// WithBlockedCommands adds blocked commands to the default list.
func (v *DefaultValidator) WithBlockedCommands(commands []string) *DefaultValidator {
	v.blockedCommands = append(v.blockedCommands, commands...)
	return v
}

// ValidatePath validates and checks if a file path is allowed.
func (v *DefaultValidator) ValidatePath(path string) error {
	if !filepath.IsAbs(path) {
		return errors.Security("path must be absolute")
	}

	cleanPath := filepath.Clean(path)
	resolvedPath, err := filepath.EvalSymlinks(cleanPath)
	if err != nil {
		resolvedPath = cleanPath
	}

	for _, blocked := range v.blockedPaths {
		if strings.HasPrefix(resolvedPath, blocked) {
			return errors.SecurityWithDetails(
				"path is blocked",
				"path accesses restricted system directory",
			)
		}
	}

	if len(v.allowedPaths) > 0 {
		allowed := false
		for _, allowedPath := range v.allowedPaths {
			if strings.HasPrefix(resolvedPath, allowedPath) {
				allowed = true
				break
			}
		}
		if !allowed {
			return errors.SecurityWithDetails(
				"path not allowed",
				"path is not in allowed directories",
			)
		}
	}

	return nil
}

// ValidateCommand validates if a command is allowed to be executed.
func (v *DefaultValidator) ValidateCommand(cmd string, args []string) error {
	if cmd == "" {
		return errors.Validation("command cannot be empty")
	}

	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return errors.Validation("invalid command format")
	}

	baseName := filepath.Base(parts[0])

	for _, blocked := range v.blockedCommands {
		if matched, _ := filepath.Match(blocked, baseName); matched {
			return errors.SecurityWithDetails(
				"command is blocked",
				"command is in the blocked list for security",
			)
		}
	}

	if len(v.allowedCommands) > 0 {
		allowed := false
		for _, allowedCmd := range v.allowedCommands {
			if matched, _ := filepath.Match(allowedCmd, baseName); matched {
				allowed = true
				break
			}
		}
		if !allowed {
			return errors.SecurityWithDetails(
				"command not allowed",
				"command is not in the allowed list",
			)
		}
	}

	return nil
}

// ValidateURL validates if a URL is safe to access.
func (v *DefaultValidator) ValidateURL(urlStr string) error {
	if urlStr == "" {
		return errors.Validation("URL cannot be empty")
	}

	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return errors.ValidationWithDetails(
			"invalid URL format",
			err.Error(),
		)
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return errors.SecurityWithDetails(
			"invalid URL scheme",
			"only HTTP and HTTPS are allowed",
		)
	}

	if parsedURL.Host == "" {
		return errors.Validation("URL must have a host")
	}

	if strings.Contains(parsedURL.Host, "localhost") ||
		strings.Contains(parsedURL.Host, "127.0.0.1") ||
		strings.Contains(parsedURL.Host, "::1") {
		return errors.SecurityWithDetails(
			"localhost access denied",
			"access to local services is not allowed",
		)
	}

	return nil
}

// SanitizePath cleans and validates a file path.
func (v *DefaultValidator) SanitizePath(path string) (string, error) {
	if err := v.ValidatePath(path); err != nil {
		return "", err
	}

	cleanPath := filepath.Clean(path)
	return cleanPath, nil
}
