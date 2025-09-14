package claude_model

type MessageBody struct {
	Model     string         `json:"model"`
	Messages  Message        `json:"messages"`
	MaxTokens int            `json:"max_tokens"`
	System    string         `json:"system"`
	Stream    bool           `json:"stream"`
	Thinking  *ThinkingBlock `json:"thinking,omitempty"`
	Temp      float32        `json:"temperature"`
}

type ThinkingBlock struct {
	Type         string `json:"type"`
	BudgetTokens int    `json:"budget_tokens"`
}

type Message interface {
}

type MessageResponse struct {
	Id         string            `json:"id"`
	Type       string            `json:"type"`
	Role       string            `json:"role"`
	Content    []ResponseMessage `json:"content"`
	Model      string            `json:"model"`
	StopReason string            `json:"stop_reason"`
	Usage      Usage             `json:"usage"`
}

type Usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

type ResponseMessage struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type Role string

const (
	Apple  Role = "user"
	Banana Role = "assistant"
)

type RequestMessage struct {
	Role    string    `json:"role"`
	Content []Content `json:"content"`
}

type Content interface {
}

type ImageContent struct {
	Type   string      `json:"type"`
	Source ImageSource `json:"source"`
}
type ImageSource struct {
	Type      string `json:"type"`
	MediaType string `json:"media_type"`
	Data      string `json:"data"`
}

type TextContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type TextMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ImageMessage struct {
	Role    string       `json:"role"`
	Content ImageContent `json:"content"`
}

// type ImageContent struct {
// 	Type   string `json:"type"`
// 	Source Source `json:"source"`
// }

type MediaType string

const (
	Jpeg   MediaType = "image/jpeg"
	Base64 MediaType = "base64"
	Png    MediaType = "image/png"
	Gif    MediaType = "image/gif"
)

type Source struct {
	Type      string    `json:"type"`
	MediaType MediaType `json:"media_type"`
	Data      string    `json:"data"`
}

type StreamEventType string

const (
	ping                StreamEventType = "ping"
	message_stop        StreamEventType = "message_stop"
	message_delta       StreamEventType = "message_delta"
	content_block_delta StreamEventType = "content_block_delta"
	content_block_stop  StreamEventType = "content_block_stop"
)

type StreamResponse struct {
	Event StreamEventType `json:"event"`
	Data  StreamData      `json:"data"`
}

type StreamData struct {
	Type  StreamEventType `json:"type"`
	Index int             `json:"index"`
	Delta StreamDelta     `json:"delta"`
}

type StreamDelta struct {
	Type     string `json:"type"`
	Text     string `json:"text"`
	Thinking string `json:"thinking"`
}
