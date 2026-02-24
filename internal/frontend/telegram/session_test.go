package telegram

import (
	"sync"
	"testing"

	"github.com/vadimtrunov/MediaMate/internal/agent"
)

func testAgentFactory() *agent.Agent {
	return agent.New(nil, nil, nil, nil, nil)
}

func TestSessionManager_IsAllowed(t *testing.T) {
	t.Run("empty whitelist allows all", func(t *testing.T) {
		sm := newSessionManager(nil)
		if !sm.isAllowed(123) {
			t.Error("expected all users allowed with nil whitelist")
		}
		if !sm.isAllowed(456) {
			t.Error("expected all users allowed with nil whitelist")
		}
	})

	t.Run("empty slice allows all", func(t *testing.T) {
		sm := newSessionManager([]int64{})
		if !sm.isAllowed(123) {
			t.Error("expected all users allowed with empty whitelist")
		}
	})

	t.Run("whitelist restricts", func(t *testing.T) {
		sm := newSessionManager([]int64{100, 200})
		if !sm.isAllowed(100) {
			t.Error("expected user 100 allowed")
		}
		if !sm.isAllowed(200) {
			t.Error("expected user 200 allowed")
		}
		if sm.isAllowed(300) {
			t.Error("expected user 300 denied")
		}
	})
}

func TestSessionManager_GetOrCreate(t *testing.T) {
	sm := newSessionManager(nil)
	factory := testAgentFactory

	a1 := sm.getOrCreate(100, factory)
	if a1 == nil {
		t.Fatal("expected non-nil agent")
	}

	a2 := sm.getOrCreate(100, factory)
	if a1 != a2 {
		t.Error("expected same agent for same user")
	}

	a3 := sm.getOrCreate(200, factory)
	if a3 == nil {
		t.Fatal("expected non-nil agent for user 200")
	}
	if a1 == a3 {
		t.Error("expected different agents for different users")
	}
}

func TestSessionManager_Reset(t *testing.T) {
	sm := newSessionManager(nil)
	factory := testAgentFactory

	a1 := sm.getOrCreate(100, factory)
	sm.reset(100)
	a2 := sm.getOrCreate(100, factory)

	if a1 == a2 {
		t.Error("expected new agent after reset")
	}
}

func TestSessionManager_Concurrent(t *testing.T) {
	sm := newSessionManager(nil)
	factory := testAgentFactory

	var wg sync.WaitGroup
	for i := range 100 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			userID := int64(i % 10)
			a := sm.getOrCreate(userID, factory)
			if a == nil {
				t.Error("expected non-nil agent")
			}
		}()
	}
	wg.Wait()
}
