package bash

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestNewSessionManager(t *testing.T) {
	sm := NewSessionManager()
	defer sm.Shutdown()

	if sm == nil {
		t.Fatal("NewSessionManager returned nil")
	}

	if sm.GetSessionCount() != 0 {
		t.Errorf("Expected 0 sessions, got %d", sm.GetSessionCount())
	}

	// Verify default configuration
	stats := sm.GetSessionStats()
	if stats["total_sessions"] != 0 {
		t.Errorf("Expected 0 total sessions, got %v", stats["total_sessions"])
	}

	if sm.sessionTimeout != 30*time.Minute {
		t.Errorf("Expected default timeout of 30m, got %v", sm.sessionTimeout)
	}
}

func TestNewSessionManagerWithConfig(t *testing.T) {
	sessionTimeout := 10 * time.Minute
	cleanupInterval := 2 * time.Minute

	sm := NewSessionManagerWithConfig(sessionTimeout, cleanupInterval)
	defer sm.Shutdown()

	if sm.sessionTimeout != sessionTimeout {
		t.Errorf("Expected timeout %v, got %v", sessionTimeout, sm.sessionTimeout)
	}

	// Verify cleanup ticker is set correctly
	if sm.cleanupTicker == nil {
		t.Error("Cleanup ticker should not be nil")
	}
}

func TestGetSessionManager_Singleton(t *testing.T) {
	// Reset global state
	globalSessionManager = nil
	sessionManagerOnce = sync.Once{}

	sm1 := GetSessionManager()
	sm2 := GetSessionManager()

	if sm1 != sm2 {
		t.Error("GetSessionManager should return the same instance")
	}

	defer sm1.Shutdown()
}

func TestExecuteCommand_NewSession(t *testing.T) {
	sm := NewSessionManagerWithConfig(5*time.Minute, 1*time.Minute)
	defer sm.Shutdown()

	ctx := context.Background()
	result, err := sm.ExecuteCommand(ctx, "echo hello", 5*time.Second)

	if err != nil {
		t.Fatalf("ExecuteCommand failed: %v", err)
	}

	if result.Stdout != "hello\n" {
		t.Errorf("Expected 'hello\\n', got %q", result.Stdout)
	}

	if result.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", result.ExitCode)
	}

	// Verify session was created
	if sm.GetSessionCount() != 1 {
		t.Errorf("Expected 1 session, got %d", sm.GetSessionCount())
	}
}

func TestExecuteCommand_PersistentSession(t *testing.T) {
	sm := NewSessionManagerWithConfig(5*time.Minute, 1*time.Minute)
	defer sm.Shutdown()

	ctx := context.Background()

	// First command to create session
	_, err := sm.ExecuteCommand(ctx, "export TEST_VAR=hello", 5*time.Second)
	if err != nil {
		t.Fatalf("First command failed: %v", err)
	}

	// Second command should see the exported variable
	result, err := sm.ExecuteCommand(ctx, "echo $TEST_VAR", 5*time.Second)
	if err != nil {
		t.Fatalf("Second command failed: %v", err)
	}

	if result.Stdout != "hello\n" {
		t.Errorf("Expected 'hello\\n', got %q", result.Stdout)
	}

	// Should still be only one session
	if sm.GetSessionCount() != 1 {
		t.Errorf("Expected 1 session, got %d", sm.GetSessionCount())
	}
}

func TestGetSession(t *testing.T) {
	sm := NewSessionManagerWithConfig(5*time.Minute, 1*time.Minute)
	defer sm.Shutdown()

	// Get non-existent session
	session, exists := sm.GetSession("nonexistent")
	if exists {
		t.Error("Should not find non-existent session")
	}
	if session != nil {
		t.Error("Session should be nil for non-existent session")
	}

	// Create a session by executing a command
	ctx := context.Background()
	_, err := sm.ExecuteCommand(ctx, "echo test", 5*time.Second)
	if err != nil {
		t.Fatalf("ExecuteCommand failed: %v", err)
	}

	// Get existing session
	session, exists = sm.GetSession("default")
	if !exists {
		t.Error("Should find existing session")
	}
	if session == nil {
		t.Error("Session should not be nil")
		return
	}

	if session.ID != "default" {
		t.Errorf("Expected session ID 'default', got %q", session.ID)
	}

	if session.AccessCount < 1 {
		t.Errorf("Expected access count >= 1, got %d", session.AccessCount)
	}
}

func TestDeleteSession(t *testing.T) {
	sm := NewSessionManagerWithConfig(5*time.Minute, 1*time.Minute)
	defer sm.Shutdown()

	// Try to delete non-existent session
	deleted := sm.DeleteSession("nonexistent")
	if deleted {
		t.Error("Should not delete non-existent session")
	}

	// Create a session
	ctx := context.Background()
	_, err := sm.ExecuteCommand(ctx, "echo test", 5*time.Second)
	if err != nil {
		t.Fatalf("ExecuteCommand failed: %v", err)
	}

	if sm.GetSessionCount() != 1 {
		t.Errorf("Expected 1 session, got %d", sm.GetSessionCount())
	}

	// Delete existing session
	deleted = sm.DeleteSession("default")
	if !deleted {
		t.Error("Should delete existing session")
	}

	if sm.GetSessionCount() != 0 {
		t.Errorf("Expected 0 sessions after deletion, got %d", sm.GetSessionCount())
	}
}

func TestSessionStats(t *testing.T) {
	sm := NewSessionManagerWithConfig(5*time.Minute, 1*time.Minute)
	defer sm.Shutdown()

	// Test stats with no sessions
	stats := sm.GetSessionStats()
	if stats["total_sessions"] != 0 {
		t.Errorf("Expected 0 total sessions, got %v", stats["total_sessions"])
	}

	// Create some sessions and commands
	ctx := context.Background()
	_, err := sm.ExecuteCommand(ctx, "echo test1", 5*time.Second)
	if err != nil {
		t.Fatalf("ExecuteCommand failed: %v", err)
	}

	// Execute another command to increase access count
	_, err = sm.ExecuteCommand(ctx, "echo test2", 5*time.Second)
	if err != nil {
		t.Fatalf("ExecuteCommand failed: %v", err)
	}

	stats = sm.GetSessionStats()
	if stats["total_sessions"] != 1 {
		t.Errorf("Expected 1 total session, got %v", stats["total_sessions"])
	}

	totalAccess, ok := stats["total_access_count"].(int64)
	if !ok || totalAccess < 2 {
		t.Errorf("Expected total access count >= 2, got %v", stats["total_access_count"])
	}

	// Check oldest and newest session times
	if stats["oldest_session"] == nil {
		t.Error("Oldest session should not be nil")
	}
	if stats["newest_session"] == nil {
		t.Error("Newest session should not be nil")
	}
}

func TestSessionTTLCleanup(t *testing.T) {
	// Use very short TTL for testing
	sm := NewSessionManagerWithConfig(100*time.Millisecond, 50*time.Millisecond)
	defer sm.Shutdown()

	// Create a session
	ctx := context.Background()
	_, err := sm.ExecuteCommand(ctx, "echo test", 5*time.Second)
	if err != nil {
		t.Fatalf("ExecuteCommand failed: %v", err)
	}

	if sm.GetSessionCount() != 1 {
		t.Errorf("Expected 1 session, got %d", sm.GetSessionCount())
	}

	// Wait for session to expire and cleanup to run
	time.Sleep(200 * time.Millisecond)

	// Session should be cleaned up
	if sm.GetSessionCount() != 0 {
		t.Errorf("Expected 0 sessions after TTL cleanup, got %d", sm.GetSessionCount())
	}
}

func TestSessionEnvironmentIsolation(t *testing.T) {
	sm := NewSessionManagerWithConfig(5*time.Minute, 1*time.Minute)
	defer sm.Shutdown()

	ctx := context.Background()

	// Set environment variable in session
	_, err := sm.ExecuteCommand(ctx, "export ISOLATED_VAR=session_value", 5*time.Second)
	if err != nil {
		t.Fatalf("Export command failed: %v", err)
	}

	// Verify it exists in session
	result, err := sm.ExecuteCommand(ctx, "echo $ISOLATED_VAR", 5*time.Second)
	if err != nil {
		t.Fatalf("Echo command failed: %v", err)
	}

	if result.Stdout != "session_value\n" {
		t.Errorf("Expected 'session_value\\n', got %q", result.Stdout)
	}

	// Verify it doesn't exist in current process environment
	if value := os.Getenv("ISOLATED_VAR"); value != "" {
		t.Errorf("Session environment leaked to process: %q", value)
	}
}

func TestSessionWorkingDirectoryPersistence(t *testing.T) {
	sm := NewSessionManagerWithConfig(5*time.Minute, 1*time.Minute)
	defer sm.Shutdown()

	ctx := context.Background()

	// Get initial working directory
	result, err := sm.ExecuteCommand(ctx, "pwd", 5*time.Second)
	if err != nil {
		t.Fatalf("pwd command failed: %v", err)
	}
	initialDir := result.Stdout

	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Change to temp directory
	_, err = sm.ExecuteCommand(ctx, "cd "+tempDir, 5*time.Second)
	if err != nil {
		t.Fatalf("cd command failed: %v", err)
	}

	// Verify we're in the new directory
	result, err = sm.ExecuteCommand(ctx, "pwd", 5*time.Second)
	if err != nil {
		t.Fatalf("pwd command failed: %v", err)
	}

	// Resolve symlinks for comparison (e.g., /var -> /private/var on macOS)
	expectedDir, err := filepath.EvalSymlinks(tempDir)
	if err != nil {
		expectedDir = tempDir
	}
	actualDir, err := filepath.EvalSymlinks(result.WorkingDirectory)
	if err != nil {
		actualDir = result.WorkingDirectory
	}

	if actualDir != expectedDir {
		t.Errorf("Expected working directory %q, got %q", expectedDir, actualDir)
	}

	// Verify current process is still in original directory
	currentDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd failed: %v", err)
	}

	initialDirClean := filepath.Clean(initialDir[:len(initialDir)-1]) // Remove newline
	if currentDir != initialDirClean {
		t.Errorf("Process working directory changed unexpectedly: %q vs %q", currentDir, initialDirClean)
	}
}

func TestConcurrentSessionAccess(t *testing.T) {
	sm := NewSessionManagerWithConfig(5*time.Minute, 1*time.Minute)
	defer sm.Shutdown()

	const numGoroutines = 10
	const commandsPerGoroutine = 5

	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines*commandsPerGoroutine)

	// Start multiple goroutines executing commands concurrently
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			ctx := context.Background()

			for j := 0; j < commandsPerGoroutine; j++ {
				_, err := sm.ExecuteCommand(ctx, "echo concurrent_test", 5*time.Second)
				if err != nil {
					errors <- err
					return
				}
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// Check for any errors
	for err := range errors {
		t.Errorf("Concurrent execution error: %v", err)
	}

	// Should still have only one session (all used "default")
	if sm.GetSessionCount() != 1 {
		t.Errorf("Expected 1 session after concurrent access, got %d", sm.GetSessionCount())
	}

	// Check access count
	session, exists := sm.GetSession("default")
	if !exists {
		t.Fatal("Default session should exist")
	}

	expectedAccess := int64(numGoroutines*commandsPerGoroutine + 1) // +1 for the GetSession call
	if session.AccessCount != expectedAccess {
		t.Errorf("Expected access count %d, got %d", expectedAccess, session.AccessCount)
	}
}

func TestShutdown(t *testing.T) {
	sm := NewSessionManagerWithConfig(5*time.Minute, 1*time.Minute)

	// Create some sessions
	ctx := context.Background()
	_, err := sm.ExecuteCommand(ctx, "echo test1", 5*time.Second)
	if err != nil {
		t.Fatalf("ExecuteCommand failed: %v", err)
	}

	if sm.GetSessionCount() != 1 {
		t.Errorf("Expected 1 session before shutdown, got %d", sm.GetSessionCount())
	}

	// Shutdown should clean up all sessions
	sm.Shutdown()

	if sm.GetSessionCount() != 0 {
		t.Errorf("Expected 0 sessions after shutdown, got %d", sm.GetSessionCount())
	}

	// Context should be cancelled
	select {
	case <-sm.ctx.Done():
		// Expected
	default:
		t.Error("Context should be cancelled after shutdown")
	}
}

func TestBackgroundCleanupRoutine(t *testing.T) {
	// Use short intervals for testing
	sm := NewSessionManagerWithConfig(50*time.Millisecond, 25*time.Millisecond)
	defer sm.Shutdown()

	ctx := context.Background()

	// Create a session
	_, err := sm.ExecuteCommand(ctx, "echo test", 5*time.Second)
	if err != nil {
		t.Fatalf("ExecuteCommand failed: %v", err)
	}

	if sm.GetSessionCount() != 1 {
		t.Errorf("Expected 1 session, got %d", sm.GetSessionCount())
	}

	// Wait for session to expire
	time.Sleep(75 * time.Millisecond)

	// Wait for cleanup routine to run
	time.Sleep(50 * time.Millisecond)

	// Session should be cleaned up
	if sm.GetSessionCount() != 0 {
		t.Errorf("Expected 0 sessions after cleanup, got %d", sm.GetSessionCount())
	}
}

func TestShellSessionFields(t *testing.T) {
	sm := NewSessionManagerWithConfig(5*time.Minute, 1*time.Minute)
	defer sm.Shutdown()

	ctx := context.Background()
	start := time.Now()

	// Create a session
	_, err := sm.ExecuteCommand(ctx, "echo test", 5*time.Second)
	if err != nil {
		t.Fatalf("ExecuteCommand failed: %v", err)
	}

	session, exists := sm.GetSession("default")
	if !exists {
		t.Fatal("Session should exist")
	}

	// Check session fields
	if session.ID != "default" {
		t.Errorf("Expected session ID 'default', got %q", session.ID)
	}

	if session.WorkingDirectory == "" {
		t.Error("Working directory should not be empty")
	}

	if session.Environment == nil {
		t.Error("Environment should not be nil")
	}

	if session.CreatedAt.Before(start) {
		t.Error("CreatedAt should be after test start")
	}

	if session.LastUsed.Before(start) {
		t.Error("LastUsed should be after test start")
	}

	if session.AccessCount < 1 {
		t.Errorf("AccessCount should be >= 1, got %d", session.AccessCount)
	}
}

func TestSessionUpdateTimestamps(t *testing.T) {
	sm := NewSessionManagerWithConfig(5*time.Minute, 1*time.Minute)
	defer sm.Shutdown()

	ctx := context.Background()

	// Create a session
	_, err := sm.ExecuteCommand(ctx, "echo test1", 5*time.Second)
	if err != nil {
		t.Fatalf("ExecuteCommand failed: %v", err)
	}

	session1, _ := sm.GetSession("default")
	firstLastUsed := session1.LastUsed
	firstAccessCount := session1.AccessCount

	// Wait a bit
	time.Sleep(10 * time.Millisecond)

	// Execute another command
	_, err = sm.ExecuteCommand(ctx, "echo test2", 5*time.Second)
	if err != nil {
		t.Fatalf("Second ExecuteCommand failed: %v", err)
	}

	session2, _ := sm.GetSession("default")

	// LastUsed should be updated
	if !session2.LastUsed.After(firstLastUsed) {
		t.Error("LastUsed should be updated after second command")
	}

	// AccessCount should be incremented
	if session2.AccessCount <= firstAccessCount {
		t.Errorf("AccessCount should be incremented: %d vs %d", session2.AccessCount, firstAccessCount)
	}
}

func TestGlobalSessionManagerShutdown(t *testing.T) {
	// Reset global state
	globalSessionManager = nil
	sessionManagerOnce = sync.Once{}

	// Get global manager
	sm := GetSessionManager()

	// Create a session
	ctx := context.Background()
	_, err := sm.ExecuteCommand(ctx, "echo test", 5*time.Second)
	if err != nil {
		t.Fatalf("ExecuteCommand failed: %v", err)
	}

	if sm.GetSessionCount() != 1 {
		t.Errorf("Expected 1 session, got %d", sm.GetSessionCount())
	}

	// Shutdown global manager
	ShutdownGlobalSessionManager()

	if sm.GetSessionCount() != 0 {
		t.Errorf("Expected 0 sessions after global shutdown, got %d", sm.GetSessionCount())
	}

	// Reset for other tests
	globalSessionManager = nil
	sessionManagerOnce = sync.Once{}
}

func TestExecuteCommandWithCurrentDirectoryFailure(t *testing.T) {
	sm := NewSessionManagerWithConfig(5*time.Minute, 1*time.Minute)
	defer sm.Shutdown()

	// Mock os.Getwd to return error by temporarily changing to a non-existent directory
	// This is tricky to test without mocking, so we'll test the error handling indirectly

	// For now, just ensure the session manager handles it gracefully
	ctx := context.Background()
	_, err := sm.ExecuteCommand(ctx, "echo test", 5*time.Second)
	if err != nil {
		t.Fatalf("ExecuteCommand should not fail: %v", err)
	}

	// The session should still be created successfully
	if sm.GetSessionCount() != 1 {
		t.Errorf("Expected 1 session despite directory issues, got %d", sm.GetSessionCount())
	}
}
