package protocol

// ChatMessage represents a role-based message (OpenAI standard)
type ChatMessage struct {
	Role    string `json:"role"`    // "system", "user", "assistant"
	Content string `json:"content"` // Text prompt or answer
}

// ChatCompletionRequest is the standard OpenAI payload sent by Cursor/Claude/Python
type ChatCompletionRequest struct {
	AgentType   string        `json:"agent_type,omitempty"` // Optional: "MATH_SOLVER", "CODE_GENERATOR", etc.
	Model       string        `json:"model"`                // e.g. "deepseek-r1"
	Messages    []ChatMessage `json:"messages"`             // Array of conversation messages
	Temperature float64       `json:"temperature"`          // 0.0 to 1.0
	MaxTokens   int           `json:"max_tokens"`
	Stream      bool          `json:"stream,omitempty"`     // Server-Sent Events (SSE) streaming mode
}

// ChatChoice represents a generated completion choice
type ChatChoice struct {
	Index        int         `json:"index"`
	Message      ChatMessage `json:"message"`
	FinishReason string      `json:"finish_reason"`
}

// TokenUsage tracks token metrics
type TokenUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// SettlementDetails gives the buyer transparent micro-USD accounting
type SettlementDetails struct {
	CostMicroUSD        int64  `json:"cost_micro_usd"`
	FormattedCostUSD    string `json:"formatted_cost_usd"`
	PlatformFeeMicroUSD int64  `json:"platform_fee_micro_usd"`
	ExecutionNodeID     string `json:"execution_node_id"`
}

// ChatCompletionResponse is returned to the buyer (non-streaming)
type ChatCompletionResponse struct {
	ID         string            `json:"id"`
	Object     string            `json:"object"` // "chat.completion"
	Created    int64             `json:"created"`
	Model      string            `json:"model"`
	Choices    []ChatChoice      `json:"choices"`
	Usage      TokenUsage        `json:"usage"`
	Settlement SettlementDetails `json:"settlement"`
}

// ChatCompletionChunkChoice represents delta content in an SSE stream
type ChatCompletionChunkChoice struct {
	Index        int         `json:"index"`
	Delta        ChatMessage `json:"delta"`
	FinishReason *string     `json:"finish_reason"`
}

// ChatCompletionChunk is an SSE streaming event packet (OpenAI standard)
type ChatCompletionChunk struct {
	ID         string                      `json:"id"`
	Object     string                      `json:"object"` // "chat.completion.chunk"
	Created    int64                       `json:"created"`
	Model      string                      `json:"model"`
	Choices    []ChatCompletionChunkChoice `json:"choices"`
	Settlement *SettlementDetails          `json:"settlement,omitempty"`
}
