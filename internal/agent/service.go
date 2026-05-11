package agent

import (
	"context"
	"fmt"
	"strings"

	"AgentOS/internal/llm"
	"AgentOS/internal/tools/mcp"
	"AgentOS/internal/tools/skill"
	"AgentOS/pkg/schema"
)

// Service is the orchestration boundary between interface adapters and the
// agent runtime. It decides whether an input should be handled as a built-in
// command or forwarded to the configured LLM provider.
type Service struct {
	provider llm.Provider
	skills   *skill.Registry
	mcps     *mcp.Registry
}

type commandInput struct {
	name string
	args []string
	rest string
}

type commandDefinition struct {
	name    string
	usage   string
	handler func(context.Context, commandInput) (schema.ChatResponse, error)
}

// NewService builds the core request handler used by every interface adapter.
func NewService(provider llm.Provider, skills *skill.Registry, mcps *mcp.Registry) *Service {
	return &Service{
		provider: provider,
		skills:   skills,
		mcps:     mcps,
	}
}

// Handle validates the incoming conversation, routes slash-commands locally,
// and otherwise enriches the request with runtime context before calling the
// LLM provider.
func (s *Service) Handle(ctx context.Context, req schema.ChatRequest) (schema.ChatResponse, error) {
	enriched, localResp, local, err := s.prepare(ctx, req)
	if local || err != nil {
		return localResp, err
	}

	resp, err := s.provider.Complete(ctx, enriched)
	if err != nil {
		return schema.ChatResponse{}, err
	}

	if resp.Metadata == nil {
		resp.Metadata = map[string]string{}
	}
	resp.Metadata["path"] = "llm"
	return resp, nil
}

// CommandNames returns the currently registered local slash commands.
func (s *Service) CommandNames() []string {
	definitions := s.commandDefinitions()
	names := make([]string, 0, len(definitions))
	for _, definition := range definitions {
		names = append(names, definition.name)
	}
	return names
}

// IsLocalCommand reports whether the input maps to a registered slash command.
func (s *Service) IsLocalCommand(input string) bool {
	parsed, ok := parseCommand(input)
	if !ok {
		return false
	}

	_, ok = s.commandHandlers()[parsed.name]
	return ok
}

// HandleStream runs the same agent flow as Handle but forwards LLM deltas to
// the caller as they arrive.
func (s *Service) HandleStream(ctx context.Context, req schema.ChatRequest, onDelta func(string) error) (schema.ChatResponse, error) {
	enriched, localResp, local, err := s.prepare(ctx, req)
	if local || err != nil {
		if err != nil {
			return localResp, err
		}
		if err := onDelta(localResp.Message.Content); err != nil {
			return schema.ChatResponse{}, err
		}
		return localResp, nil
	}

	resp, err := s.provider.Stream(ctx, enriched, onDelta)
	if err != nil {
		return schema.ChatResponse{}, err
	}

	if resp.Metadata == nil {
		resp.Metadata = map[string]string{}
	}
	resp.Metadata["path"] = "llm"
	return resp, nil
}

func (s *Service) prepare(ctx context.Context, req schema.ChatRequest) (schema.ChatRequest, schema.ChatResponse, bool, error) {
	if len(req.Messages) == 0 {
		return schema.ChatRequest{}, schema.ChatResponse{}, false, fmt.Errorf("request must contain at least one message")
	}

	last := req.Messages[len(req.Messages)-1]
	if last.Role != schema.RoleUser {
		return schema.ChatRequest{}, schema.ChatResponse{}, false, fmt.Errorf("last message must be a user message")
	}

	if handled, resp, err := s.handleCommand(ctx, last.Content); handled {
		return schema.ChatRequest{}, resp, true, err
	}

	enriched := schema.ChatRequest{
		SessionID: req.SessionID,
		Messages:  make([]schema.Message, 0, len(req.Messages)+1),
	}
	enriched.Messages = append(enriched.Messages, schema.Message{
		Role:    schema.RoleSystem,
		Content: s.runtimePrompt(),
	})
	enriched.Messages = append(enriched.Messages, req.Messages...)
	return enriched, schema.ChatResponse{}, false, nil
}

// handleCommand intercepts locally supported commands so they can execute
// without involving the LLM.
func (s *Service) handleCommand(ctx context.Context, input string) (bool, schema.ChatResponse, error) {
	parsed, ok := parseCommand(input)
	if !ok {
		return false, schema.ChatResponse{}, nil
	}

	handler, ok := s.commandHandlers()[parsed.name]
	if !ok {
		return false, schema.ChatResponse{}, nil
	}

	resp, err := handler(ctx, parsed)
	return true, resp, err
}

// runtimePrompt summarizes the current runtime capabilities and is prepended as
// a system message before delegating to the LLM.
func (s *Service) runtimePrompt() string {
	return buildRuntimePrompt(s.skills.Descriptions(), s.mcps.Descriptions())
}

func buildRuntimePrompt(skills []string, mcps []string) string {
	return fmt.Sprintf(
		"You are an web runtime. Available skills: %s. Available MCP servers: %s. When the user asks to use them, explain the best route and keep answers concise.",
		strings.Join(skills, "; "),
		strings.Join(mcps, "; "),
	)
}

func (s *Service) commandDefinitions() []commandDefinition {
	return []commandDefinition{
		{name: "/help", usage: "/help", handler: s.handleHelpCommand},
		{name: "/list", usage: "/list", handler: nil},
		{name: "/model", usage: "/model [name]", handler: s.handleModelCommand},
		{name: "/skill", usage: "/skill <name> [input]", handler: s.handleSkillCommand},
		{name: "/mcp", usage: "/mcp <name> [input]", handler: s.handleMCPCommand},
	}
}

func (s *Service) commandHandlers() map[string]func(context.Context, commandInput) (schema.ChatResponse, error) {
	handlers := make(map[string]func(context.Context, commandInput) (schema.ChatResponse, error))
	for _, definition := range s.commandDefinitions() {
		handlers[definition.name] = definition.handler
	}
	return handlers
}

func (s *Service) handleHelpCommand(_ context.Context, _ commandInput) (schema.ChatResponse, error) {
	usages := make([]string, 0, len(s.commandDefinitions()))
	for _, definition := range s.commandDefinitions() {
		usages = append(usages, definition.usage)
	}

	return schema.ChatResponse{
		Message: schema.Message{
			Role: schema.RoleAssistant,
			Content: fmt.Sprintf(
				"Commands:\n%s\n\nSkills: %s\nMCP: %s",
				strings.Join(usages, "\n"),
				strings.Join(s.skills.Names(), ", "),
				strings.Join(s.mcps.Names(), ", "),
			),
		},
		Metadata: map[string]string{"path": "builtin"},
	}, nil
}

func (s *Service) handleModelCommand(_ context.Context, input commandInput) (schema.ChatResponse, error) {
	if input.rest == "" {
		return schema.ChatResponse{
			Message:  schema.Message{Role: schema.RoleAssistant, Content: fmt.Sprintf("Current model: %s", fallback(s.provider.Model(), "not configured"))},
			Metadata: map[string]string{"path": "builtin"},
		}, nil
	}

	previous := fallback(s.provider.Model(), "not configured")
	s.provider.SetModel(input.rest)
	return schema.ChatResponse{
		Message: schema.Message{
			Role:    schema.RoleAssistant,
			Content: fmt.Sprintf("Model changed: %s -> %s", previous, s.provider.Model()),
		},
		Metadata: map[string]string{"path": "builtin"},
	}, nil
}

func (s *Service) handleSkillCommand(ctx context.Context, input commandInput) (schema.ChatResponse, error) {
	if len(input.args) == 0 {
		return schema.ChatResponse{}, fmt.Errorf("usage: /skill <name> [input]")
	}

	name := input.args[0]
	payload := strings.TrimSpace(strings.TrimPrefix(input.rest, name))
	result, err := s.skills.Execute(ctx, name, skill.Request{
		Input: payload,
	})
	if err != nil {
		return schema.ChatResponse{}, err
	}

	return schema.ChatResponse{
		Message: schema.Message{Role: schema.RoleAssistant, Content: result.Output},
		Metadata: map[string]string{
			"path": "skill",
			"name": name,
		},
	}, nil
}

func (s *Service) handleMCPCommand(ctx context.Context, input commandInput) (schema.ChatResponse, error) {
	if len(input.args) == 0 {
		return schema.ChatResponse{}, fmt.Errorf("usage: /mcp <server> [input]")
	}

	name := input.args[0]
	payload := strings.TrimSpace(strings.TrimPrefix(input.rest, name))
	result, err := s.mcps.Call(ctx, name, mcp.Request{
		Input: payload,
	})
	if err != nil {
		return schema.ChatResponse{}, err
	}

	return schema.ChatResponse{
		Message: schema.Message{Role: schema.RoleAssistant, Content: result.Output},
		Metadata: map[string]string{
			"path": "mcp",
			"name": name,
		},
	}, nil
}

func parseCommand(input string) (commandInput, bool) {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" || !strings.HasPrefix(trimmed, "/") {
		return commandInput{}, false
	}

	parts := strings.Fields(trimmed)
	if len(parts) == 0 {
		return commandInput{}, false
	}

	return commandInput{
		name: parts[0],
		args: parts[1:],
		rest: strings.TrimSpace(strings.TrimPrefix(trimmed, parts[0])),
	}, true
}

func fallback(value, defaultValue string) string {
	if strings.TrimSpace(value) == "" {
		return defaultValue
	}
	return value
}
