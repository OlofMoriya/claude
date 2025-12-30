# OpenAI Base Model Implementation

## Overview

This document describes the shared base implementation for OpenAI-compatible models (OpenAI 4o, Grok, Ollama).

## Architecture

### Base Package (`models/open-ai-base/`)

The base package provides shared functionality for all OpenAI-compatible API models:

```
models/open-ai-base/
├── openai-base-data-model.go   # Shared data structures
└── openai-base-model.go        # Shared business logic
```

### Key Components

#### 1. **Data Models** (`openai-base-data-model.go`)
- `Message` - Chat message with tool call support
- `RequestMessage` - Request message format
- `ChatCompletionRequest` - API request structure
- `ChatCompletion` - API response structure
- `ToolCall`, `FunctionTool`, `FunctionCall` - Tool calling structures
- `ChatCompletionChunk` - Streaming response chunk

#### 2. **Business Logic** (`openai-base-model.go`)

**OpenAICompatibleModel** - Base model with shared functionality:

```go
type OpenAICompatibleModel struct {
    ResponseHandler   models.ResponseHandler
    HistoryRepository data.HistoryRepository
    Prompt            string
    AccumulatedAnswer string
    ContextId         int64
    Context           *data.Context
    StreamedToolCalls map[int]*StreamingToolCall
}
```

**Key Methods:**

- `HandleStreamedLine(line []byte)` - Process streaming responses
- `HandleBodyBytes(bytes []byte)` - Process non-streaming responses
- `HandleStreamedToolCalls(deltaToolCalls []ToolCall)` - Accumulate tool calls
- `FinishStreaming()` - Complete streaming and execute tools
- `HandleToolCalls(message Message)` - Execute tools and return results

**Shared Functions:**

- `CreatePayload()` - Build API request with history and tools
- `ConvertToolsToOpenAIFormat()` - Convert tools to OpenAI format
- `ConvertProperties()` - Convert tool property definitions

## Model Implementations

### Grok Model

The Grok model now uses the base implementation by embedding `OpenAICompatibleModel`:

```go
type GrokModel struct {
    openai_base.OpenAICompatibleModel
}
```

**Grok-specific customizations:**
- API URL: `https://api.x.ai/v1/chat/completions`
- API Key: `XAI_API_KEY` environment variable
- Model name: `grok-beta`
- Max tokens: 8000

### File Structure

```
models/grok/
├── grok-model.go        # Grok-specific implementation (minimal)
└── grok-data-model.go   # Re-exports base types for convenience
```

### Usage Example

```go
// Create Grok model
model := &grok_model.GrokModel{
    OpenAICompatibleModel: openai_base.OpenAICompatibleModel{
        ResponseHandler:   responseHandler,
        HistoryRepository: historyRepo,
    },
}

// Use like any other model
request := model.CreateRequest(context, prompt, streaming, history, modifiers)
```

## Benefits of Base Implementation

### 1. **Code Reuse**
- ~500 lines of code shared between models
- Single source of truth for tool calling logic
- Consistent behavior across models

### 2. **Maintainability**
- Bug fixes apply to all models
- New features added once
- Easier to understand and modify

### 3. **Consistency**
- Same tool calling behavior
- Same error handling
- Same logging patterns

### 4. **Easy Extension**
- Add new OpenAI-compatible models quickly
- Override specific methods if needed
- Customize API endpoints and parameters

## How to Add a New Model

To add a new OpenAI-compatible model (e.g., Ollama):

### 1. Create model file:

```go
package new_model

import "owl/models/open-ai-base"

type NewModel struct {
    openai_base.OpenAICompatibleModel
}

func (model *NewModel) CreateRequest(context *data.Context, prompt string, streaming bool, history []data.History, modifiers *models.PayloadModifiers) *http.Request {
    // Initialize base fields
    model.Prompt = prompt
    model.AccumulatedAnswer = ""
    model.ContextId = context.Id
    model.Context = context
    model.StreamedToolCalls = make(map[int]*openai_base.StreamingToolCall)

    // Create payload with your model-specific settings
    payload := openai_base.CreatePayload(
        prompt, 
        streaming, 
        history, 
        modifiers, 
        "your-model-name",  // Model name
        4000,                // Max tokens
    )
    
    return createRequest(payload)  // Your custom request function
}

func (model *NewModel) HandleStreamedLine(line []byte) {
    model.OpenAICompatibleModel.HandleStreamedLine(line)
}

func (model *NewModel) HandleBodyBytes(bytes []byte) {
    model.OpenAICompatibleModel.HandleBodyBytes(bytes)
}

func createRequest(payload openai_base.ChatCompletionRequest) *http.Request {
    // Your API-specific request creation
}
```

### 2. Create data model file:

```go
package new_model

import "owl/models/open-ai-base"

// Re-export base types
type Message = openai_base.Message
type ChatCompletionRequest = openai_base.ChatCompletionRequest
// ... etc
```

### 3. Register in main.go:

```go
case "new-model":
    model = &new_model.NewModel{
        OpenAICompatibleModel: openai_base.OpenAICompatibleModel{
            ResponseHandler:   cliResponseHandler,
            HistoryRepository: user,
        },
    }
```

## Tool Calling Flow

### Non-Streaming Mode:
```
User Prompt
    ↓
CreateRequest() → Create payload with tools
    ↓
API Call
    ↓
HandleBodyBytes() → Parse response
    ↓
Found tool calls? → HandleToolCalls()
    ↓
Execute tools via ToolRunner
    ↓
AwaitedQuery() with tool results
    ↓
Model continues with results
    ↓
Final response
```

### Streaming Mode:
```
User Prompt
    ↓
CreateRequest() → Create payload with tools
    ↓
API Call (streaming)
    ↓
HandleStreamedLine() → Process chunks
    ↓
Tool call chunks? → HandleStreamedToolCalls()
    ↓
Accumulate tool call data
    ↓
[DONE] → FinishStreaming()
    ↓
Execute accumulated tools
    ↓
AwaitedQuery() with tool results
    ↓
Final response
```

## Testing

### Test Grok with Tools

```bash
# List files
./owl -model grok -prompt "List all files in the current directory"

# Read file
./owl -model grok -prompt "Read the README.md file"

# Git status
./owl -model grok -prompt "Show me git status"

# Streaming with tools
./owl -model grok -prompt "List files and read the first one" -stream
```

### Debug Logging

Check `~/.owl/debug.log` for detailed logging:
```bash
tail -f ~/.owl/debug.log
```

Look for:
- `Added X tools to payload` - Tools included in request
- `Handling streamed tool calls` - Tool calls being accumulated
- `Executing X tools` - Tools being executed
- `Tool result: ...` - Tool execution results

## Future Improvements

### Potential Enhancements:

1. **Rate Limiting**
   - Add rate limiting per model
   - Handle 429 responses gracefully

2. **Retry Logic**
   - Automatic retry on transient failures
   - Exponential backoff

3. **Token Counting**
   - Track token usage per model
   - Warn when approaching limits

4. **Tool Call Caching**
   - Cache tool results for identical calls
   - Reduce redundant API calls

5. **Parallel Tool Execution**
   - Execute independent tools in parallel
   - Reduce total response time

6. **Tool Call Limits**
   - Prevent infinite loops
   - Max depth for recursive calls

## Comparison: Base vs Individual Implementation

| Aspect | Before (Individual) | After (Base) |
|--------|---------------------|--------------|
| Lines of code per model | ~500 | ~50 |
| Code duplication | High | Minimal |
| Bug fixes needed | 3x (per model) | 1x (in base) |
| New feature cost | 3x implementation | 1x implementation |
| Consistency | Variable | Guaranteed |
| Testing effort | 3x | 1x + integration |

## Migration Status

| Model | Status | Notes |
|-------|--------|-------|
| OpenAI 4o | ✅ Working | Original implementation |
| Grok | ✅ Migrated | Uses base, tested |
| Ollama | ⏳ Pending | Next to migrate |

## API Compatibility

The base implementation supports OpenAI Chat Completions API format:

- ✅ Messages with text content
- ✅ Messages with image content
- ✅ Tool/function calling
- ✅ Streaming responses
- ✅ Non-streaming responses
- ✅ Conversation history
- ✅ System prompts (via context)
- ✅ Tool results in messages
- ✅ Multiple tool calls per response

## Troubleshooting

### Issue: Tools not being called

**Check:**
1. Is `tools.GetCustomTools()` returning tools?
2. Is the model name correct?
3. Are tool definitions valid?
4. Check debug logs for payload

### Issue: Tool execution fails

**Check:**
1. Are tool arguments being parsed correctly?
2. Does the tool exist in ToolRunner?
3. Check tool error messages in logs
4. Verify tool has required permissions

### Issue: Streaming stops early

**Check:**
1. Is `[DONE]` message being received?
2. Are there network timeouts?
3. Check for JSON parsing errors
4. Verify API key is valid

## Contact & Support

For issues or questions about the base implementation:
- Check debug logs: `~/.owl/debug.log`
- Review this documentation
- Test with non-streaming first
- Compare with working OpenAI 4o model

## Changelog

### Version 1.0 (Current)
- ✅ Created base package
- ✅ Migrated Grok to use base
- ✅ Full tool calling support
- ✅ Streaming and non-streaming modes
- ✅ Comprehensive logging

### Planned
- ⏳ Migrate Ollama
- ⏳ Add retry logic
- ⏳ Add token counting
- ⏳ Add tool result caching
