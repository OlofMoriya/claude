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
	MaxTokens int            `json:"max_completion_tokens"`
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

// Web search specific structures (shared by OpenAI and Grok)
type WebSearchTool struct {
	Type string `json:"type"` // "web_search"
}

// Request structure for web search API (/v1/responses endpoint)
type ResponseRequest struct {
	Model  string        `json:"model"`
	Input  interface{}   `json:"input"` // Can be string or []Message
	Tools  []interface{} `json:"tools,omitempty"`
	Stream bool          `json:"stream,omitempty"`
}

// Response structures for web search responses
type ResponseItem struct {
	Type    string          `json:"type"` // "web_search_call" or "message"
	ID      string          `json:"id"`
	Status  string          `json:"status,omitempty"`  // For web_search_call
	Action  interface{}     `json:"action,omitempty"`  // For web_search_call action details
	Role    string          `json:"role,omitempty"`    // For message
	Content []ContentOutput `json:"content,omitempty"` // For message
}

type ContentOutput struct {
	Type        string       `json:"type"` // "output_text"
	Text        string       `json:"text,omitempty"`
	Annotations []Annotation `json:"annotations,omitempty"`
}

type Annotation struct {
	Type       string `json:"type"` // "url_citation"
	StartIndex int    `json:"start_index"`
	EndIndex   int    `json:"end_index"`
	URL        string `json:"url"`
	Title      string `json:"title"`
}

// Response structure from /v1/responses API
type ResponseAPIResponse struct {
	ID         string        `json:"id"`
	Object     string        `json:"object"`
	CreatedAt  int64         `json:"created_at"`
	Model      string        `json:"model"`
	OutputText string        `json:"output_text,omitempty"` // Some APIs may use this
	Output     []interface{} `json:"output,omitempty"`      // Grok uses this - array of items
	Status     string        `json:"status,omitempty"`
}

// Stored response content for history (to avoid issues like in Claude)
type WebSearchResponseContent struct {
	Type        string       `json:"type"` // "web_search_response"
	OutputText  string       `json:"output_text"`
	WebSearchId string       `json:"web_search_id,omitempty"` // ID of the web_search_call
	MessageId   string       `json:"message_id,omitempty"`    // ID of the message
	Annotations []Annotation `json:"annotations,omitempty"`
}

type InputMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}
