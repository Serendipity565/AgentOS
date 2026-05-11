package llm

import (
	"context"

	"AgentOS/pkg/schema"
)

type StreamHandler func(delta string) error

// Provider abstracts a concrete LLM backend behind a common chat interface.
type Provider interface {
	Complete(ctx context.Context, req schema.ChatRequest) (schema.ChatResponse, error)
	Stream(ctx context.Context, req schema.ChatRequest, onDelta StreamHandler) (schema.ChatResponse, error)
	Model() string
	SetModel(model string)
}
