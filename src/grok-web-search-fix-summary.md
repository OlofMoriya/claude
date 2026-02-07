# Grok Web Search Response Handling - Fix Summary

## Problem
The Grok API's web search response from the `/v1/responses` endpoint was not being parsed correctly. The response has a complex nested structure that was not matching the expected data model.

## Root Cause
The actual Grok response structure uses:
- An `output` array (not a simple `output_text` string field)
- Multiple items in the array with different types (`web_search_call` and `message`)
- The actual text content nested inside: `output[].content[].text`
- Annotations for citations nested inside: `output[].content[].annotations[]`

## Changes Made

### 1. Data Model Updates (`models/open-ai-base/openai-base-data-model.go`)

```go
type ResponseAPIResponse struct {
    ID         string         `json:"id"`
    Object     string         `json:"object"`
    CreatedAt  int64          `json:"created_at"`      // Changed from Created
    Model      string         `json:"model"`
    OutputText string         `json:"output_text,omitempty"` 
    Output     []interface{}  `json:"output,omitempty"`      // Changed from string to []interface{}
    Status     string         `json:"status,omitempty"`      // Added
}

type ResponseItem struct {
    Type    string          `json:"type"`
    ID      string          `json:"id"`
    Status  string          `json:"status,omitempty"`
    Action  interface{}     `json:"action,omitempty"`  // Added
    Role    string          `json:"role,omitempty"`
    Content []ContentOutput `json:"content,omitempty"`
}
```

### 2. Response Handler (`models/open-ai-base/openai-base-model.go`)

Completely rewrote `HandleWebSearchResponse()` to:
1. Iterate through the `output` array
2. Marshal/unmarshal each item to parse it as a `ResponseItem`
3. Identify `web_search_call` items (for tracking search IDs)
4. Identify `message` items with `role: "assistant"`
5. Extract text from `content[].text` where `type == "output_text"`
6. Extract annotations for citations
7. Display the response text and citations to the user
8. Store the structured response for history

### 3. Detection Logic (`models/grok/grok-model.go`)

```go
// Changed from:
if err := json.Unmarshal(bytes, &webSearchResponse); err == nil && webSearchResponse.OutputText != "" {

// To:
if err := json.Unmarshal(bytes, &webSearchResponse); err == nil && len(webSearchResponse.Output) > 0 {
```

## Grok Response Structure

```json
{
  "id": "85491799-fd93-8bd1-d4f6-52d0d38a66e1",
  "object": "response",
  "created_at": 1769983884,
  "model": "grok-4-1-fast-reasoning",
  "status": "completed",
  "output": [
    {
      "id": "ws_..._call_83313814",
      "type": "web_search_call",
      "status": "completed",
      "action": { "type": "search", "query": "..." }
    },
    {
      "id": "msg_85491799...",
      "type": "message",
      "role": "assistant",
      "status": "completed",
      "content": [
        {
          "type": "output_text",
          "text": "The actual response text here...",
          "annotations": [
            {
              "type": "url_citation",
              "url": "https://example.com",
              "title": "Example",
              "start_index": 100,
              "end_index": 150
            }
          ]
        }
      ]
    }
  ]
}
```

## Testing
Test with a web search query using the Grok model with `--web` flag to verify:
- Response text is properly extracted and displayed
- Citations are shown with sources
- Response is properly stored in history for context continuity

## Related Files
- `models/open-ai-base/openai-base-data-model.go` - Data structures
- `models/open-ai-base/openai-base-model.go` - Response handling logic
- `models/grok/grok-model.go` - Grok-specific detection
