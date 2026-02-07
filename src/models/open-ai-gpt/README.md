# OpenAI GPT-5.2 Model

This model provides access to OpenAI's GPT-5.2 model with full support for both standard chat completions and web search capabilities.

## Features

- ✅ Standard chat completions via `/v1/chat/completions`
- ✅ Web search via `/v1/responses` endpoint
- ✅ Full tool/function calling support
- ✅ Streaming support (for standard completions)
- ✅ Image support (via clipboard)
- ✅ Conversation history management
- ✅ System prompt support
- ✅ URL citations from web search results

## Configuration

Set the `OPENAI_API_KEY` environment variable:

```bash
export OPENAI_API_KEY="your-api-key-here"
```

## Web Search

When the `modifiers.Web` flag is enabled, the model automatically switches to using OpenAI's `/v1/responses` endpoint with web search capabilities.

### How It Works

1. **Different Endpoint**: Web search uses `/v1/responses` instead of `/v1/chat/completions`
2. **Automatic Detection**: The model detects web search responses and handles them appropriately
3. **Citation Display**: URLs and titles from web search results are displayed as footnotes
4. **History Storage**: Web search responses are stored with a special structure to avoid history issues (learned from Claude implementation)

### Storage Format

Web search responses are stored in history as:

```json
{
  "type": "web_search_response",
  "output_text": "The actual response text...",
  "web_search_id": "ws_...",
  "message_id": "msg_...",
  "annotations": [
    {
      "type": "url_citation",
      "start_index": 100,
      "end_index": 200,
      "url": "https://example.com",
      "title": "Example Title"
    }
  ]
}
```

This structured format ensures that:
- Web search responses can be properly reconstructed from history
- Citations are preserved across conversation turns
- The system can differentiate between regular and web search responses

## Implementation Notes

### Shared with Grok

The web search functionality is implemented in the `open-ai-base` package and shared between:
- OpenAI GPT-5.2 (this model)
- Grok models

Both use the same `/v1/responses` endpoint format with minor differences in the model identifiers.

### Avoiding Claude's History Issues

The Claude model had problems with web search history because the response format was different from regular completions. This implementation:

1. Stores web search responses with a distinct `type: "web_search_response"`
2. Extracts just the `output_text` when reconstructing history
3. Preserves annotations separately for reference
4. Never tries to replay web search calls (they're handled by the API)

## Model Identifiers

- Standard completions: `gpt-5.2` (16K max tokens)
- Web search: `gpt-5` (as per OpenAI docs)

## Example Usage

```go
// Standard chat completion
request := model.CreateRequest(context, prompt, true, history, &models.PayloadModifiers{
    Web: false,
})

// With web search
request := model.CreateRequest(context, prompt, false, history, &models.PayloadModifiers{
    Web: true,
})
```

## API Endpoints

- **Standard**: `https://api.openai.com/v1/chat/completions`
- **Web Search**: `https://api.openai.com/v1/responses`
