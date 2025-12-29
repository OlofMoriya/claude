# Ollama Model

This package provides an Ollama integration for the Owl project using Ollama's OpenAI-compatible API.

## Overview

You're correct! Ollama supports the OpenAI API format, which makes integration straightforward. This implementation uses the same data structures as the OpenAI model but points to a local (or remote) Ollama instance.

## Prerequisites

1. **Install Ollama**: Download from [ollama.ai](https://ollama.ai)
2. **Pull a model**: 
   ```bash
   ollama pull llama2
   # or other models like:
   ollama pull mistral
   ollama pull codellama
   ollama pull llama3
   ollama pull phi3
   ```

3. **Start Ollama**: Ollama typically runs as a service on `http://localhost:11434`

## Configuration

Configure the Ollama model using environment variables:

### Required
None! The model will use sensible defaults.

### Optional

- **OLLAMA_MODEL**: The model to use (default: `llama2`)
  ```bash
  export OLLAMA_MODEL="mistral"
  export OLLAMA_MODEL="llama3"
  export OLLAMA_MODEL="codellama"
  ```

- **OLLAMA_URL**: The Ollama server URL (default: `http://localhost:11434`)
  ```bash
  export OLLAMA_URL="http://localhost:11434"
  # Or for remote instances:
  export OLLAMA_URL="http://your-server:11434"
  ```

- **OLLAMA_API_KEY**: Optional API key (only needed for remote/secured instances)
  ```bash
  export OLLAMA_API_KEY="your-api-key"
  ```

## Usage

```go
import "owl/models/ollama"

// Create the model
model := ollama_model.NewOllamaModel()

// Set response handler
model.SetResponseHandler(responseHandler)

// Create request
req := model.CreateRequest(context, prompt, streaming, history, modifiers)
```

## Features

✅ Streaming support
✅ Conversation history
✅ Image support (for vision models like llava)
✅ OpenAI-compatible API
✅ No API key required for local instances
✅ Configurable model selection

## Supported Models

Ollama has many models available:

- **General**: llama2, llama3, mistral, mixtral
- **Code**: codellama, deepseek-coder, wizardcoder
- **Vision**: llava, bakllava
- **Small/Fast**: phi3, tinyllama, gemma

Check available models:
```bash
ollama list
```

Browse all models:
```bash
ollama search
```

## API Endpoint

This implementation uses Ollama's OpenAI-compatible endpoint:
```
POST http://localhost:11434/v1/chat/completions
```

## Benefits

- **Privacy**: All processing happens locally
- **No costs**: No API fees
- **Offline capable**: Works without internet
- **Fast**: No network latency (for local instances)
- **Flexible**: Easy to switch between different models

## Troubleshooting

### Check if Ollama is running
```bash
curl http://localhost:11434/api/tags
```

### Test the OpenAI-compatible API
```bash
curl http://localhost:11434/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "llama2",
    "messages": [{"role": "user", "content": "Hello!"}]
  }'
```

### Check Ollama logs
```bash
# On macOS
tail -f ~/.ollama/logs/server.log

# On Linux
journalctl -u ollama -f
```

## Performance Tips

1. **GPU Acceleration**: Ollama automatically uses GPU if available (NVIDIA, AMD, Metal on macOS)
2. **Model Size**: Smaller models (7B) are faster, larger models (70B) are more capable
3. **Quantization**: Ollama models are pre-quantized for efficient inference
4. **Context Length**: Adjust based on your needs (longer context = more memory)

## Notes

- The implementation reuses the same data structures as the OpenAI model
- Streaming works the same way as OpenAI's streaming
- Image support requires vision-capable models (e.g., llava)
- Some advanced OpenAI features may not be available in all Ollama models
