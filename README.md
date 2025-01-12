# Owlllm / Claude
This is a cli-tool/service that prompts different llms and saves the contexts of the prompts to maintain a full conversation between a user and multiple llms.

To start server 
owl --serve

To use cli 
owl --context_name="$context" --prompt="$message" --stream --history=20

The response can be streamed or awaited. the streamed response is outputted without any formatting while the final text will be formatted with as markdown when using the cli. 

Api keys are supplied from environment for the different llm services.

## Features
- [x] Cli
- [x] Server
- [x] Switch models with --model=
- [x] Local db storage with sqlite (for cli mode)
- [x] Remote db storage with postgresql (for server mode)
- [x] Store history by user
- [x] Store history by context
- [x] Generate embeddings for string

- [ ] Supply system prompt for a context
- [ ] Implement ollama as a model

- [ ] Store texts with embeddings for RAG
- [ ] Implement vector search with vertex ai
- [ ] Prompt with embeddings search string
