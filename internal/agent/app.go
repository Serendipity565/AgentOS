package agent

import (
	"AgentOS/config"
	"fmt"
	"strings"

	"AgentOS/internal/llm"
	"AgentOS/internal/memory"
	"AgentOS/internal/tools/mcp"
	"AgentOS/internal/tools/skill"
)

// App is the assembled runtime used by process entrypoints.
type App struct {
	Service *Service
	Memory  memory.Store
}

// Initialize builds the agent runtime from configuration.
func Initialize(cfg config.Config) (*App, error) {
	provider, err := newProvider(cfg)
	if err != nil {
		return nil, err
	}

	skills := newSkillRegistry(cfg)
	mcps := newMCPRegistry(cfg)
	store := memory.NewInMemoryStore()

	return &App{
		Service: NewService(provider, skills, mcps),
		Memory:  store,
	}, nil
}

func newSkillRegistry(_ config.Config) *skill.Registry {
	registry := skill.NewRegistry()
	registry.Register(skill.EchoSkill{})
	return registry
}

func newMCPRegistry(_ config.Config) *mcp.Registry {
	registry := mcp.NewRegistry()
	registry.Register(mcp.TimeServer{})
	return registry
}

func newProvider(cfg config.Config) (llm.Provider, error) {
	selected, err := cfg.LLM.ActiveProvider()
	if err != nil {
		return nil, err
	}

	driver := strings.ToLower(strings.TrimSpace(selected.Driver))
	if driver == "" {
		driver = "openai-compatible"
	}

	if driver == "mock" || strings.EqualFold(selected.Provider, "mock") {
		return llm.NewMockProvider(), nil
	}

	if driver != "openai-compatible" {
		return nil, fmt.Errorf("unsupported llm driver %q for provider %q", selected.Driver, selected.Name)
	}

	provider, err := llm.NewOpenAICompatibleProvider(llm.OpenAICompatibleConfig{
		BaseURL: selected.BaseURL,
		APIKey:  selected.APIKey,
		Model:   selected.Model,
	})
	if err != nil {
		return nil, fmt.Errorf("build llm provider: %w", err)
	}
	return provider, nil
}
