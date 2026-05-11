package llm

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"AgentOS/pkg/schema"
)

type MockProvider struct {
	model string
	mu    sync.RWMutex
}

func NewMockProvider() *MockProvider {
	return &MockProvider{model: "mock"}
}

func (p *MockProvider) Complete(_ context.Context, req schema.ChatRequest) (schema.ChatResponse, error) {
	last := req.Messages[len(req.Messages)-1]
	return schema.ChatResponse{
		Message: schema.Message{
			Role: schema.RoleAssistant,
			Content: fmt.Sprintf(
				"[mock-provider] received: %s\nConfigure internal/config/config.yaml to use a third-party OpenAI-compatible API.",
				strings.TrimSpace(last.Content),
			),
		},
		Metadata: map[string]string{"provider": "mock"},
	}, nil
}

func (p *MockProvider) Stream(ctx context.Context, req schema.ChatRequest, onDelta StreamHandler) (schema.ChatResponse, error) {
	resp, err := p.Complete(ctx, req)
	if err != nil {
		return schema.ChatResponse{}, err
	}
	if err := onDelta(resp.Message.Content); err != nil {
		return schema.ChatResponse{}, err
	}
	return resp, nil
}

func (p *MockProvider) Model() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.model
}

func (p *MockProvider) SetModel(model string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.model = strings.TrimSpace(model)
}
