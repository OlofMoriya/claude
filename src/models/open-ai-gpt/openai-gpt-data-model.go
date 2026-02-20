package openai_gpt_model

// OpenAI GPT-5.2 uses OpenAI-compatible API format
// All data models are imported from the base package
// This file exists for backward compatibility and future GPT-specific extensions

import (
	"owl/models/open-ai-base"
)

// Re-export base types for convenience
type Message = openai_base.Message
type RequestMessage = openai_base.RequestMessage
type ChatCompletionRequest = openai_base.ChatCompletionRequest
type FunctionTool = openai_base.FunctionTool
type FunctionDefinition = openai_base.FunctionDefinition
type ToolCall = openai_base.ToolCall
type FunctionCall = openai_base.FunctionCall
type Choice = openai_base.Choice
type Usage = openai_base.Usage
type ChatCompletion = openai_base.ChatCompletion
type Delta = openai_base.Delta
type RequestContent = openai_base.RequestContent
type Image = openai_base.Image
type ChatCompletionChunkChoice = openai_base.ChatCompletionChunkChoice
type ChatCompletionChunk = openai_base.ChatCompletionChunk

// Re-export web search types from base
type WebSearchTool = openai_base.WebSearchTool
type ResponseRequest = openai_base.ResponseRequest
type ResponseItem = openai_base.ResponseItem
type ContentOutput = openai_base.ContentOutput
type Annotation = openai_base.Annotation
type ResponseAPIResponse = openai_base.ResponseAPIResponse
type WebSearchResponseContent = openai_base.WebSearchResponseContent
type InputMessage = openai_base.InputMessage
