package schema

type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
)

type Message struct {
	Role    Role   `json:"role"`
	Content string `json:"content"`
}

type ChatRequest struct {
	SessionID string    `json:"session_id"`
	Messages  []Message `json:"messages"`
}

type ChatResponse struct {
	Message  Message           `json:"message"`
	Metadata map[string]string `json:"metadata,omitempty"`
}
