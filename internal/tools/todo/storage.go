// Package todo provides in-memory storage for session-based todos.
package todo

import (
	"github.com/d-kuro/claude-code-mcp/pkg/collections"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// SessionStorage manages todo items for sessions using a generic SyncMap.
type SessionStorage struct {
	todos *collections.SyncMap[*mcp.ServerSession, []TodoItem]
}

// NewSessionStorage creates a new session storage.
func NewSessionStorage() *SessionStorage {
	return &SessionStorage{
		todos: collections.NewSyncMap[*mcp.ServerSession, []TodoItem](),
	}
}

// GetSessionTodos retrieves todos for the given session.
func (s *SessionStorage) GetSessionTodos(session *mcp.ServerSession) []TodoItem {
	todos, exists := s.todos.Get(session)
	if !exists {
		return []TodoItem{}
	}

	// Return a copy to prevent external modification
	result := make([]TodoItem, len(todos))
	copy(result, todos)
	return result
}

// SetSessionTodos updates todos for the given session.
func (s *SessionStorage) SetSessionTodos(session *mcp.ServerSession, todos []TodoItem) {
	// Store a copy to prevent external modification
	todosCopy := make([]TodoItem, len(todos))
	copy(todosCopy, todos)
	s.todos.Set(session, todosCopy)
}

// ClearSessionTodos removes all todos for the given session.
func (s *SessionStorage) ClearSessionTodos(session *mcp.ServerSession) {
	s.todos.Delete(session)
}

// GetAllSessions returns all sessions that have todos.
func (s *SessionStorage) GetAllSessions() []*mcp.ServerSession {
	var sessions []*mcp.ServerSession
	s.todos.Range(func(session *mcp.ServerSession, _ []TodoItem) bool {
		sessions = append(sessions, session)
		return true
	})
	return sessions
}

// GetSessionCount returns the number of sessions with todos.
func (s *SessionStorage) GetSessionCount() int {
	return s.todos.Len()
}

// GetTotalTodoCount returns the total number of todos across all sessions.
func (s *SessionStorage) GetTotalTodoCount() int {
	total := 0
	s.todos.Range(func(_ *mcp.ServerSession, todos []TodoItem) bool {
		total += len(todos)
		return true
	})
	return total
}

// ClearAll removes all todos from all sessions.
func (s *SessionStorage) ClearAll() {
	s.todos.Clear()
}

// Global storage instance for backward compatibility
var globalStorage = NewSessionStorage()

// Legacy functions for backward compatibility
func GetSessionTodos(session *mcp.ServerSession) []TodoItem {
	return globalStorage.GetSessionTodos(session)
}

func SetSessionTodos(session *mcp.ServerSession, todos []TodoItem) {
	globalStorage.SetSessionTodos(session, todos)
}

func ClearSessionTodos(session *mcp.ServerSession) {
	globalStorage.ClearSessionTodos(session)
}

func GetAllSessions() []*mcp.ServerSession {
	return globalStorage.GetAllSessions()
}

func GetSessionCount() int {
	return globalStorage.GetSessionCount()
}

func GetTotalTodoCount() int {
	return globalStorage.GetTotalTodoCount()
}
