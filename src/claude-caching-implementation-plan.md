# Claude Prompt Caching Implementation Plan

## Overview

This document outlines the plan to implement Anthropic's prompt caching feature in the Claude model. Caching will be applied to:

1. **The second message in a context** - To cache the system prompt and initial conversation context
2. **Tool result responses** - As these tend to be larger and benefit most from caching

## How Claude Caching Works

Anthropic's prompt caching uses a `cache_control` block with `type: "ephemeral"` added to content blocks. The cache is applied to everything **up to and including** the marked block.

```json
{
  "type": "text",
  "text": "Some content...",
  "cache_control": { "type": "ephemeral" }
}
```

---

## Implementation Steps

### Step 1: Update Data Model (`models/claude/claude-data-model.go`)

Add the `CacheControl` struct and update content types to support caching.

#### 1.1 Add CacheControl Type

**Location:** After the `ThinkingBlock` struct (around line 45)

```go
// CacheControl enables prompt caching for content blocks
type CacheControl struct {
	Type string `json:"type"` // "ephemeral"
}
```

#### 1.2 Update TextContent Struct

**Location:** Replace existing `TextContent` struct (around line 82)

```go
type TextContent struct {
	Type         string        `json:"type"`
	Text         string        `json:"text"`
	CacheControl *CacheControl `json:"cache_control,omitempty"`
}
```

#### 1.3 Update ToolResponseContent Struct

**Location:** Replace existing `ToolResponseContent` struct (around line 72)

```go
type ToolResponseContent struct {
	Type         string        `json:"type"`
	Id           string        `json:"tool_use_id"`
	Content      string        `json:"content"`
	IsError      bool          `json:"is_error"`
	CacheControl *CacheControl `json:"cache_control,omitempty"`
}
```

#### 1.4 Add System Message Type for Caching

**Location:** After the `TextContent` struct

The system prompt can also benefit from caching. Add a new type:

```go
type SystemContent struct {
	Type         string        `json:"type"`
	Text         string        `json:"text"`
	CacheControl *CacheControl `json:"cache_control,omitempty"`
}
```

#### 1.5 Update MessageBody for System Array

**Location:** Update the `MessageBody` struct to support system as array

```go
type MessageBody struct {
	Model     string          `json:"model"`
	Messages  Message         `json:"messages"`
	MaxTokens int             `json:"max_tokens"`
	System    []SystemContent `json:"system,omitempty"`
	Stream    bool            `json:"stream"`
	Thinking  *ThinkingBlock  `json:"thinking,omitempty"`
	Temp      float32         `json:"temperature"`
	Tools     []ToolModel     `json:"tools"`
}
```

---

### Step 2: Update Request Creation (`models/claude/claude-model.go`)

#### 2.1 Add Cache Control Helper Function

**Location:** Add before `createClaudePayload` function (around line 250)

```go
// getCacheControl returns a cache control block for ephemeral caching
func getCacheControl() *CacheControl {
	return &CacheControl{Type: "ephemeral"}
}
```

#### 2.2 Update System Prompt Handling in createClaudePayload

**Location:** Replace the system prompt assignment (around line 330-332)

**Current code:**
```go
if context != nil && context.SystemPrompt != "" {
	payload.System = context.SystemPrompt
}
```

**New code:**
```go
if context != nil && context.SystemPrompt != "" {
	// Cache system prompt when we have conversation history
	systemContent := SystemContent{
		Type: "text",
		Text: context.SystemPrompt,
	}
	if len(history) >= 1 {
		systemContent.CacheControl = getCacheControl()
	}
	payload.System = []SystemContent{systemContent}
}
```

#### 2.3 Apply Caching to Second Message in History

**Location:** Inside the history processing loop in `createClaudePayload` (around line 270)

Update the history loop to mark the second message (index 1) with cache control:

```go
// Process history and handle tool results
for i, h := range history {
	j, err := json.Marshal(h)
	if err != nil {
		panic("failed to marshall h")
	}
	logger.Debug.Printf("added history: %s", j)

	// Determine if this is the second message (good cache breakpoint)
	isSecondMessage := (i == 1)
	
	// Create user message with potential caching
	userContent := TextContent{
		Type: "text",
		Text: h.Prompt,
	}
	if isSecondMessage {
		userContent.CacheControl = getCacheControl()
		logger.Debug.Printf("Applying cache control to message at index %d", i)
	}
	messages = append(messages, RequestMessage{
		Role:    "user",
		Content: []Content{userContent},
	})

	// ... rest of history processing remains the same
```

#### 2.4 Apply Caching to Tool Result Responses

**Location:** Inside the tool response handling in `createClaudePayload` (around line 300-315)

**Current code:**
```go
if len(modifiers.ToolResponses) > 0 {
	for _, response := range modifiers.ToolResponses {
		content = append(content, ToolResponseContent{
			Type:    "tool_result",
			Content: response.Response,
			IsError: false,
			Id:      response.Id,
		})
	}
}
```

**New code:**
```go
if len(modifiers.ToolResponses) > 0 {
	for i, response := range modifiers.ToolResponses {
		toolContent := ToolResponseContent{
			Type:    "tool_result",
			Content: response.Response,
			IsError: false,
			Id:      response.Id,
		}
		// Apply cache to the last tool result (caches everything up to this point)
		if i == len(modifiers.ToolResponses)-1 {
			toolContent.CacheControl = getCacheControl()
			logger.Debug.Printf("Applying cache control to tool result: %s", response.Id)
		}
		content = append(content, toolContent)
	}
}
```

---

### Step 3: Update HTTP Headers

#### 3.1 Enable Prompt Caching Beta Header

**Location:** In `createClaudeRequest` function (around line 370)

Add the beta header after existing headers:

```go
req.Header.Set("x-api-key", apiKey)
req.Header.Set("Content-Type", "application/json")
req.Header.Set("anthropic-version", "2023-06-01")
req.Header.Set("anthropic-beta", "prompt-caching-2024-07-31")  // ADD THIS LINE
```

---

## Summary of Changes

| File | Location | Change |
|------|----------|--------|
| `claude-data-model.go` | Line ~45 | Add `CacheControl` struct |
| `claude-data-model.go` | Line ~72 | Update `ToolResponseContent` with `CacheControl` field |
| `claude-data-model.go` | Line ~82 | Update `TextContent` with `CacheControl` field |
| `claude-data-model.go` | Line ~85 | Add `SystemContent` struct |
| `claude-data-model.go` | Line ~10 | Update `MessageBody.System` to `[]SystemContent` |
| `claude-model.go` | Line ~250 | Add `getCacheControl()` helper function |
| `claude-model.go` | Line ~270 | Apply cache to second history message |
| `claude-model.go` | Line ~300 | Apply cache to tool result responses |
| `claude-model.go` | Line ~330 | Update system prompt handling for caching |
| `claude-model.go` | Line ~370 | Add `anthropic-beta` header |

---

## Testing Recommendations

1. **Verify cache header is sent** - Check logs for the `anthropic-beta` header
2. **Monitor token usage** - Cached requests should show `cache_creation_input_tokens` and `cache_read_input_tokens` in the response
3. **Test with multi-turn conversations** - Verify cache is applied on second message
4. **Test with tool calls** - Verify tool results are cached properly

---

## Expected Benefits

- **Cost reduction**: Cached tokens are charged at 90% discount
- **Latency improvement**: Cached content processes faster (up to 85% faster for large contexts)
- **Best impact on tool results**: Tool results are often large (file contents, etc.) and benefit most from caching

---

## Rollback Plan

If issues arise, simply:
1. Remove the `anthropic-beta` header
2. Remove `CacheControl` fields from content structs (or leave them as they'll be `omitempty`)
