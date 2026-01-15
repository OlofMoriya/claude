package openai_base

type Message struct {
	Role       string     `json:"role"`
	Content    string     `json:"content,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallId string     `json:"tool_call_id,omitempty"`
}

type RequestMessage struct {
	Role       string      `json:"role"`
	Content    interface{} `json:"content,omitempty"`
	ToolCalls  []ToolCall  `json:"tool_calls,omitempty"`
	ToolCallId string      `json:"tool_call_id,omitempty"`
}

type ChatCompletionRequest struct {
	Model     string         `json:"model"`
	Messages  []interface{}  `json:"messages"`
	Tools     []FunctionTool `json:"tools,omitempty"`
	Stream    bool           `json:"stream"`
	MaxTokens int            `json:"max_tokens"`
}

// Tool calling structures (OpenAI format)
type FunctionTool struct {
	Type     string             `json:"type"`
	Function FunctionDefinition `json:"function"`
}

type FunctionDefinition struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

type ToolCall struct {
	Index    int          `json:"index,omitempty"`
	Id       string       `json:"id"`
	Type     string       `json:"type"`
	Function FunctionCall `json:"function"`
}

type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type Choice struct {
	Index        int         `json:"index"`
	Message      Message     `json:"message"`
	Logprobs     interface{} `json:"logprobs"`
	FinishReason string      `json:"finish_reason"`
}

type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type ChatCompletion struct {
	ID                string   `json:"id"`
	Object            string   `json:"object"`
	Created           int64    `json:"created"`
	Model             string   `json:"model"`
	SystemFingerprint string   `json:"system_fingerprint"`
	Choices           []Choice `json:"choices"`
	Usage             Usage    `json:"usage"`
}

type Delta struct {
	Role      string     `json:"role,omitempty"`
	Content   string     `json:"content,omitempty"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
}

type RequestContent struct {
	Type     string `json:"type"`
	Text     string `json:"text,omitempty"`
	ImageURL Image  `json:"image_url,omitempty"`
}

type Image struct {
	URL string `json:"url"`
}

type ChatCompletionChunkChoice struct {
	Index        int         `json:"index"`
	Delta        Delta       `json:"delta"`
	Logprobs     interface{} `json:"logprobs"`
	FinishReason *string     `json:"finish_reason,omitempty"`
}

type ChatCompletionChunk struct {
	ID                string                      `json:"id"`
	Object            string                      `json:"object"`
	Created           int64                       `json:"created"`
	Model             string                      `json:"model"`
	SystemFingerprint string                      `json:"system_fingerprint"`
	Choices           []ChatCompletionChunkChoice `json:"choices"`
}
