package memory

import (
	"context"
	"sync"

	"AgentOS/pkg/schema"
)

// Store defines the memory abstraction used to persist conversation history.
type Store interface {
	Load(ctx context.Context, sessionID string) ([]schema.Message, error)
	Save(ctx context.Context, sessionID string, messages []schema.Message) error
}

// InMemoryStore is a minimal memory implementation useful for local
// development before a persistent backend is introduced.
type InMemoryStore struct {
	mu       sync.RWMutex
	sessions map[string][]schema.Message
}

func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{sessions: make(map[string][]schema.Message)}
}

func (s *InMemoryStore) Load(_ context.Context, sessionID string) ([]schema.Message, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	history := append([]schema.Message(nil), s.sessions[sessionID]...)
	return history, nil
}

func (s *InMemoryStore) Save(_ context.Context, sessionID string, messages []schema.Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.sessions[sessionID] = append([]schema.Message(nil), messages...)
	return nil
}
