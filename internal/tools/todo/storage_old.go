//go:build ignore

// Package todo provides in-memory storage for session-based todos.
package todo

import (
	"sync"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// sessionTodos stores todos for each session in memory.
var sessionTodos = make(map[*mcp.ServerSession][]TodoItem)

// sessionMutex protects access to sessionTodos.
var sessionMutex sync.RWMutex

// GetSessionTodos retrieves todos for the given session.
func GetSessionTodos(session *mcp.ServerSession) []TodoItem {
	sessionMutex.RLock()
	defer sessionMutex.RUnlock()

	todos, exists := sessionTodos[session]
	if !exists {
		return []TodoItem{}
	}

	// Return a copy to prevent external modification
	result := make([]TodoItem, len(todos))
	copy(result, todos)
	return result
}

// SetSessionTodos updates todos for the given session.
func SetSessionTodos(session *mcp.ServerSession, todos []TodoItem) {
	sessionMutex.Lock()
	defer sessionMutex.Unlock()

	// Store a copy to prevent external modification
	sessionTodos[session] = make([]TodoItem, len(todos))
	copy(sessionTodos[session], todos)
}

// ClearSessionTodos removes all todos for the given session.
func ClearSessionTodos(session *mcp.ServerSession) {
	sessionMutex.Lock()
	defer sessionMutex.Unlock()

	delete(sessionTodos, session)
}

// GetAllSessions returns all sessions that have todos.
func GetAllSessions() []*mcp.ServerSession {
	sessionMutex.RLock()
	defer sessionMutex.RUnlock()

	sessions := make([]*mcp.ServerSession, 0, len(sessionTodos))
	for session := range sessionTodos {
		sessions = append(sessions, session)
	}
	return sessions
}

// GetSessionCount returns the number of sessions with todos.
func GetSessionCount() int {
	sessionMutex.RLock()
	defer sessionMutex.RUnlock()

	return len(sessionTodos)
}

// GetTotalTodoCount returns the total number of todos across all sessions.
func GetTotalTodoCount() int {
	sessionMutex.RLock()
	defer sessionMutex.RUnlock()

	total := 0
	for _, todos := range sessionTodos {
		total += len(todos)
	}
	return total
}
