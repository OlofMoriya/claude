package grok_model

// Grok uses OpenAI-compatible API format
// All data models are imported from the base package
// This file exists for backward compatibility and future Grok-specific extensions

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
