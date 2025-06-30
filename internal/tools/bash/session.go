// Package bash provides session management for persistent shell execution.
package bash

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"
	"time"
)

// SessionManager manages persistent shell sessions with TTL-based cleanup.
type SessionManager struct {
	mu             sync.RWMutex
	sessions       map[string]*ShellSession
	executor       *ShellExecutor
	sessionTimeout time.Duration
	cleanupTicker  *time.Ticker
	ctx            context.Context
	cancel         context.CancelFunc
	wg             sync.WaitGroup
}

// ShellSession represents a persistent shell session.
type ShellSession struct {
	ID               string
	WorkingDirectory string
	Environment      map[string]string
	CreatedAt        time.Time
	LastUsed         time.Time
	AccessCount      int64
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

// ShutdownGlobalSessionManager gracefully shuts down the global session manager.
// This should be called during application shutdown to prevent resource leaks.
func ShutdownGlobalSessionManager() {
	if globalSessionManager != nil {
		globalSessionManager.Shutdown()
	}
}

// NewSessionManager creates a new session manager with TTL-based cleanup.
func NewSessionManager() *SessionManager {
	return NewSessionManagerWithConfig(30*time.Minute, 5*time.Minute)
}

// NewSessionManagerWithConfig creates a session manager with custom TTL and cleanup interval.
func NewSessionManagerWithConfig(sessionTimeout, cleanupInterval time.Duration) *SessionManager {
	ctx, cancel := context.WithCancel(context.Background())

	sm := &SessionManager{
		sessions:       make(map[string]*ShellSession),
		executor:       NewShellExecutor(),
		sessionTimeout: sessionTimeout,
		cleanupTicker:  time.NewTicker(cleanupInterval),
		ctx:            ctx,
		cancel:         cancel,
	}

	// Start background cleanup goroutine
	sm.startCleanupRoutine()

	return sm
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
			AccessCount:      0,
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

	// Update last used time and access count
	session.LastUsed = time.Now()
	session.AccessCount++
	sm.mu.Unlock()

	// Execute command with session context
	return sm.executor.ExecuteInSession(ctx, session, command, timeout)
}

// GetSession returns a session by ID and updates its last used time.
func (sm *SessionManager) GetSession(sessionID string) (*ShellSession, bool) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	session, exists := sm.sessions[sessionID]
	if exists {
		session.LastUsed = time.Now()
		session.AccessCount++
	}
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

// startCleanupRoutine starts the background cleanup goroutine.
func (sm *SessionManager) startCleanupRoutine() {
	sm.wg.Add(1)
	go func() {
		defer sm.wg.Done()
		for {
			select {
			case <-sm.ctx.Done():
				return
			case <-sm.cleanupTicker.C:
				sm.cleanupExpiredSessions()
			}
		}
	}()
}

// cleanupExpiredSessions removes sessions that have exceeded the TTL.
func (sm *SessionManager) cleanupExpiredSessions() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	now := time.Now()
	expiredSessions := make([]string, 0)

	for sessionID, session := range sm.sessions {
		if now.Sub(session.LastUsed) > sm.sessionTimeout {
			expiredSessions = append(expiredSessions, sessionID)
		}
	}

	if len(expiredSessions) > 0 {
		log.Printf("Cleaning up %d expired sessions", len(expiredSessions))
		for _, sessionID := range expiredSessions {
			sm.cleanupSessionResources(sm.sessions[sessionID])
			delete(sm.sessions, sessionID)
		}
	}
}

// cleanupSessionResources performs cleanup of session-specific resources.
func (sm *SessionManager) cleanupSessionResources(session *ShellSession) {
	// Log session cleanup for monitoring
	log.Printf("Cleaning up session %s (created: %v, last used: %v, access count: %d)",
		session.ID, session.CreatedAt, session.LastUsed, session.AccessCount)

	// Additional cleanup can be added here if needed:
	// - Close file handles
	// - Clean temporary files
	// - Reset environment variables
	// For now, we just clear the environment map
	session.Environment = nil
}

// Shutdown gracefully shuts down the session manager.
func (sm *SessionManager) Shutdown() {
	sm.cancel()
	sm.cleanupTicker.Stop()
	sm.wg.Wait()

	// Clean up all remaining sessions
	sm.mu.Lock()
	defer sm.mu.Unlock()

	log.Printf("Shutting down session manager with %d active sessions", len(sm.sessions))
	for sessionID, session := range sm.sessions {
		sm.cleanupSessionResources(session)
		delete(sm.sessions, sessionID)
	}
}

// GetSessionCount returns the current number of active sessions (for monitoring).
func (sm *SessionManager) GetSessionCount() int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return len(sm.sessions)
}

// GetSessionStats returns detailed statistics about sessions.
func (sm *SessionManager) GetSessionStats() map[string]interface{} {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	stats := map[string]interface{}{
		"total_sessions":     len(sm.sessions),
		"session_timeout":    sm.sessionTimeout.String(),
		"oldest_session":     time.Time{},
		"newest_session":     time.Time{},
		"total_access_count": int64(0),
	}

	if len(sm.sessions) > 0 {
		oldest := time.Now()
		newest := time.Time{}
		totalAccess := int64(0)

		for _, session := range sm.sessions {
			if session.CreatedAt.Before(oldest) {
				oldest = session.CreatedAt
			}
			if session.CreatedAt.After(newest) {
				newest = session.CreatedAt
			}
			totalAccess += session.AccessCount
		}

		stats["oldest_session"] = oldest
		stats["newest_session"] = newest
		stats["total_access_count"] = totalAccess
	}

	return stats
}
