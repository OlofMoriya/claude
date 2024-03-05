package main

type MessageBody struct {
	Model     string  `json:"model"`
	Messages  Message `json:"messages"`
	MaxTokens int     `json:"max_tokens"`
	System    string  `json:"system"`
	Stream    bool    `json:"stream"`
	Temp      float32 `json:"temperature"`
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

type TextMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ImageMessage struct {
	Role    string       `json:"role"`
	Content ImageContent `json:"content"`
}

type ImageContent struct {
	Type   string `json:"type"`
	Source Source `json:"source"`
}

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
