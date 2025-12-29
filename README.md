# Owl

This is a cli-tool/service that prompts different llms and saves the contexts of the prompts to maintain a full conversation between a user and multiple llms.

To use cli 
owl --context_name="$context" --prompt="$message" --stream --history=20

To start server 
owl --serve

The response can be streamed or awaited. the streamed response is outputted without any formatting while the final text will be formatted with as markdown when using the cli. 

## Run
./owl -tui

## Database
OWL_LOCAL_DATABASE=owl

## For server mode (PostgreSQL)
DB_CONNECTIONSTRING=postgres://user:pass@localhost/owl

## API Keys
ANTHROPIC_API_KEY=your_claude_key
OPENAI_API_KEY=your_openai_key
XAI_API_KEY=your_grok_key


## Structure
owl/
├── main.go                 # Entry point
├── cli-response-handler.go # CLI output formatting
├── get-context.go          # Context management
├── data/                   # Database layer
│   ├── history-*.go        # Conversation history
│   ├── postgres-db.go      # PostgreSQL adapter
│   └── sqllite-db.go       # SQLite adapter
├── models/                 # LLM integrations
│   ├── claude/
│   ├── grok/
│   ├── open-ai-4o/
│   └── ...
├── tools/                  # AI tools & capabilities
│   ├── tool_runner.go      # Tool execution engine
│   ├── git_tool.go
│   ├── http_request_tool.go
│   └── ...
├── services/               # Business logic
│   ├── query.go            # Query orchestration
│   ├── clipboard.go
│   └── pdf.go
├── tui/                    # Terminal UI
│   ├── chat_view.go
│   └── list_view.go
└── http/                   # HTTP server
    └── server.go


## Usage
owl -tui

owl -prompt "Explain quantum computing"

owl -model claude -prompt "Write a Go function"
owl -model 4o -prompt "Analyze this code"
owl -model grok -prompt "Summarize the news"

owl -context_name myproject -history 5 -prompt "Continue our discussion"

owl -stream -prompt "Tell me a long story"

owl -image -prompt "What's in this image?"

owl -pdf ./document.pdf -prompt "Summarize this document"

owl -thinking -stream_thinking -output_thinking -prompt "Solve this complex problem"

owl -view -context_name myproject -history 20

owl -serve -port 3000 -stream
owl -serve -port 443 -secure  # HTTPS

### Start a project context
owl -context_name refactoring -prompt "I want to refactor my auth system"

### Continue with history
owl -context_name refactoring -history 3 -prompt "Show me the code changes"

### Review the conversation
owl -view -context_name refactoring

### Owl can read files, analyze code, and suggest improvements
owl -prompt "Read main.go and suggest performance improvements"

owl -prompt "Generate an image of a futuristic cityscape at sunset"

### Models can make HTTP requests autonomously
owl -prompt "Fetch the latest issues from the GitHub API for golang/go"

## Features
- [x] Cli
- [x] Server
- [x] Switch models with --model=
- [x] Local db storage with sqlite (for cli mode)
- [x] Remote db storage with postgresql (for server mode)
- [x] Store history by user
- [x] Store history by context
- [x] Generate embeddings for string
- [x] Supply system prompt for a context
- [x] Grok
- [x] Claude
- [x] Open AI
- [x] Vision (send in image)
- [x] Pdf (send in pdf, claude only)
- [x] Tool use
    - [ ] Git status and logs
    - [ ] list files
    - [ ] Write file
    - [ ] Read file
- [ ] Tool use for streamed queries
- [ ] Cache files and history in cluade
- [ ] Split history in branches
- [ ] File support for web server
- [ ] Implement ollama as a model

## Maybe
- [ ] MCP  
- [ ] Store texts with embeddings for RAG
- [ ] Implement vector search with vertex ai
- [ ] Prompt with embeddings search string
