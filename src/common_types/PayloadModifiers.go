package commontypes

import (
	"net/http"
	"owl/data"
)

type ToolResponse struct {
	Id              string
	Response        string
	ResponseMessage interface{}
}

type TokenUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	CacheReadTokens  int `json:"cache_read_tokens"`
	CacheWriteTokens int `json:"cache_write_tokens"`
}

type PayloadModifiers struct {
	ToolResponses    []ToolResponse
	Pdf              string
	Web              bool
	Image            bool
	ToolGroupFilters []string
}

type ResponseHandler interface {
	RecievedText(text string, color *string)
	FinalText(contextId int64, prompt string, response string, responseContent string, toolResults string, modelName string, usage *TokenUsage)
	// func recievedImage(encoded string)
}

type Model interface {
	CreateRequest(context *data.Context, prompt string, streaming bool, history []data.History, modifiers *PayloadModifiers) *http.Request
	HandleStreamedLine(line []byte)
	HandleBodyBytes(bytes []byte)
	SetResponseHandler(responseHandler ResponseHandler)
}
