package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"AgentOS/pkg/schema"
)

type OpenAICompatibleConfig struct {
	BaseURL    string
	APIKey     string
	Model      string
	HTTPClient *http.Client
}

type OpenAICompatibleProvider struct {
	baseURL    string
	apiKey     string
	model      string
	httpClient *http.Client
	mu         sync.RWMutex
}

func NewOpenAICompatibleProvider(cfg OpenAICompatibleConfig) (*OpenAICompatibleProvider, error) {
	if cfg.BaseURL == "" || cfg.APIKey == "" || cfg.Model == "" {
		return nil, fmt.Errorf("baseURL, apiKey and model are required")
	}
	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 60 * time.Second}
	}
	return &OpenAICompatibleProvider{
		baseURL:    strings.TrimRight(cfg.BaseURL, "/"),
		apiKey:     cfg.APIKey,
		model:      cfg.Model,
		httpClient: client,
	}, nil
}

type chatCompletionRequest struct {
	Model    string                  `json:"model"`
	Messages []chatCompletionMessage `json:"messages"`
	Stream   bool                    `json:"stream,omitempty"`
}

type chatCompletionMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatCompletionResponse struct {
	Choices []struct {
		Message chatCompletionMessage `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

type chatCompletionStreamResponse struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func (p *OpenAICompatibleProvider) Complete(ctx context.Context, req schema.ChatRequest) (schema.ChatResponse, error) {
	httpResp, err := p.sendRequest(ctx, req, false)
	if err != nil {
		return schema.ChatResponse{}, err
	}
	defer httpResp.Body.Close()

	raw, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return schema.ChatResponse{}, err
	}

	var parsed chatCompletionResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return schema.ChatResponse{}, fmt.Errorf("decode provider response: %w", err)
	}

	if httpResp.StatusCode >= 400 {
		msg := strings.TrimSpace(string(raw))
		if parsed.Error != nil && parsed.Error.Message != "" {
			msg = parsed.Error.Message
		}
		return schema.ChatResponse{}, fmt.Errorf("provider error: %s", msg)
	}

	if len(parsed.Choices) == 0 {
		return schema.ChatResponse{}, fmt.Errorf("provider returned no choices")
	}

	return schema.ChatResponse{
		Message: schema.Message{
			Role:    schema.RoleAssistant,
			Content: parsed.Choices[0].Message.Content,
		},
		Metadata: map[string]string{"provider": "openai-compatible"},
	}, nil
}

func (p *OpenAICompatibleProvider) Model() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.model
}

func (p *OpenAICompatibleProvider) SetModel(model string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.model = strings.TrimSpace(model)
}

func (p *OpenAICompatibleProvider) Stream(ctx context.Context, req schema.ChatRequest, onDelta StreamHandler) (schema.ChatResponse, error) {
	httpResp, err := p.sendRequest(ctx, req, true)
	if err != nil {
		return schema.ChatResponse{}, err
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode >= 400 {
		raw, readErr := io.ReadAll(httpResp.Body)
		if readErr != nil {
			return schema.ChatResponse{}, readErr
		}
		var parsed chatCompletionStreamResponse
		if err := json.Unmarshal(raw, &parsed); err == nil && parsed.Error != nil && parsed.Error.Message != "" {
			return schema.ChatResponse{}, fmt.Errorf("provider error: %s", parsed.Error.Message)
		}
		return schema.ChatResponse{}, fmt.Errorf("provider error: %s", strings.TrimSpace(string(raw)))
	}

	reader := bufio.NewScanner(httpResp.Body)
	reader.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var builder strings.Builder
	for reader.Scan() {
		line := strings.TrimRight(reader.Text(), "\r")
		if line == "" || strings.HasPrefix(line, ":") {
			continue
		}
		if !strings.HasPrefix(line, "data:") {
			continue
		}

		data := strings.TrimPrefix(line, "data:")
		if strings.HasPrefix(data, " ") {
			data = data[1:]
		}
		if data == "[DONE]" {
			break
		}

		var parsed chatCompletionStreamResponse
		if err := json.Unmarshal([]byte(data), &parsed); err != nil {
			return schema.ChatResponse{}, fmt.Errorf("decode stream response: %w", err)
		}
		if parsed.Error != nil && parsed.Error.Message != "" {
			return schema.ChatResponse{}, fmt.Errorf("provider error: %s", parsed.Error.Message)
		}
		if len(parsed.Choices) == 0 {
			continue
		}

		delta := parsed.Choices[0].Delta.Content
		if delta == "" {
			continue
		}
		builder.WriteString(delta)
		if err := onDelta(delta); err != nil {
			return schema.ChatResponse{}, err
		}
	}

	if err := reader.Err(); err != nil {
		return schema.ChatResponse{}, err
	}

	return schema.ChatResponse{
		Message: schema.Message{
			Role:    schema.RoleAssistant,
			Content: builder.String(),
		},
		Metadata: map[string]string{"provider": "openai-compatible"},
	}, nil
}

func (p *OpenAICompatibleProvider) sendRequest(ctx context.Context, req schema.ChatRequest, stream bool) (*http.Response, error) {
	payload := chatCompletionRequest{
		Model:    p.Model(),
		Messages: make([]chatCompletionMessage, 0, len(req.Messages)),
		Stream:   stream,
	}
	for _, msg := range req.Messages {
		payload.Messages = append(payload.Messages, chatCompletionMessage{
			Role:    string(msg.Role),
			Content: msg.Content,
		})
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")
	if stream {
		httpReq.Header.Set("Accept", "text/event-stream")
	}

	return p.httpClient.Do(httpReq)
}
