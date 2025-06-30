// Package bash provides session management for persistent shell execution.
package bash

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"
)

// SessionManager manages persistent shell sessions.
type SessionManager struct {
	mu       sync.RWMutex
	sessions map[string]*ShellSession
	executor *ShellExecutor
}

// ShellSession represents a persistent shell session.
type ShellSession struct {
	ID               string
	WorkingDirectory string
	Environment      map[string]string
	CreatedAt        time.Time
	LastUsed         time.Time
}

// CommandResult represents the result of a command execution.
type CommandResult struct {
	Stdout           string
	Stderr           string
	ExitCode         int
	Duration         time.Duration
	WorkingDirectory string
}

var (
	globalSessionManager *SessionManager
	sessionManagerOnce   sync.Once
)

// GetSessionManager returns the global session manager instance.
func GetSessionManager() *SessionManager {
	sessionManagerOnce.Do(func() {
		globalSessionManager = NewSessionManager()
	})
	return globalSessionManager
}

// NewSessionManager creates a new session manager.
func NewSessionManager() *SessionManager {
	return &SessionManager{
		sessions: make(map[string]*ShellSession),
		executor: NewShellExecutor(),
	}
}

// ExecuteCommand executes a command in the default persistent session.
func (sm *SessionManager) ExecuteCommand(ctx context.Context, command string, timeout time.Duration) (*CommandResult, error) {
	sessionID := "default"

	sm.mu.Lock()
	session, exists := sm.sessions[sessionID]
	if !exists {
		// Create new session
		cwd, err := os.Getwd()
		if err != nil {
			sm.mu.Unlock()
			return nil, fmt.Errorf("failed to get current working directory: %w", err)
		}

		session = &ShellSession{
			ID:               sessionID,
			WorkingDirectory: cwd,
			Environment:      make(map[string]string),
			CreatedAt:        time.Now(),
			LastUsed:         time.Now(),
		}

		// Copy current environment
		for _, env := range os.Environ() {
			if len(env) > 0 {
				// Parse key=value format
				for i := 0; i < len(env); i++ {
					if env[i] == '=' && i > 0 {
						key := env[:i]
						value := env[i+1:]
						session.Environment[key] = value
						break
					}
				}
			}
		}

		sm.sessions[sessionID] = session
	}

	// Update last used time
	session.LastUsed = time.Now()
	sm.mu.Unlock()

	// Execute command with session context
	return sm.executor.ExecuteInSession(ctx, session, command, timeout)
}

// GetSession returns a session by ID.
func (sm *SessionManager) GetSession(sessionID string) (*ShellSession, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	session, exists := sm.sessions[sessionID]
	return session, exists
}

// DeleteSession removes a session.
func (sm *SessionManager) DeleteSession(sessionID string) bool {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	_, exists := sm.sessions[sessionID]
	if exists {
		delete(sm.sessions, sessionID)
	}

	return exists
}
