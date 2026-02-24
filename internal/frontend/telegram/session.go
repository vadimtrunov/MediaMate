package telegram

import (
	"sync"

	"github.com/vadimtrunov/MediaMate/internal/agent"
)

// sessionManager manages per-user agent sessions and access control.
type sessionManager struct {
	mu       sync.Mutex
	sessions map[int64]*agent.Agent
	allowed  map[int64]bool // nil or empty = allow all
}

// newSessionManager creates a session manager.
// If allowedUserIDs is empty, all users are allowed.
func newSessionManager(allowedUserIDs []int64) *sessionManager {
	allowed := make(map[int64]bool, len(allowedUserIDs))
	for _, id := range allowedUserIDs {
		allowed[id] = true
	}
	return &sessionManager{
		sessions: make(map[int64]*agent.Agent),
		allowed:  allowed,
	}
}

// isAllowed checks if a user is authorized to use the bot.
func (sm *sessionManager) isAllowed(userID int64) bool {
	if len(sm.allowed) == 0 {
		return true
	}
	return sm.allowed[userID]
}

// getOrCreate returns an existing session or creates a new one using the factory.
func (sm *sessionManager) getOrCreate(userID int64, factory AgentFactory) *agent.Agent {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if a, ok := sm.sessions[userID]; ok {
		return a
	}
	a := factory()
	sm.sessions[userID] = a
	return a
}

// reset clears a user's session, forcing a new agent on next message.
func (sm *sessionManager) reset(userID int64) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	delete(sm.sessions, userID)
}
