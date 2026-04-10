package ollama_model

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	commontypes "owl/common_types"
	"owl/data"
	openai_base "owl/models/open-ai-base"
)

type OllamaModel struct {
	openai_base.OpenAICompatibleModel
	ModelVersion string
	ollamaURL    string
}

func NewOllamaModel(responseHandler commontypes.ResponseHandler, historyRepository data.HistoryRepository, modelName string) *OllamaModel {
	// Get model name from env or use default
	if modelName == "" {
		modelName = "qwen3" // default model
	}

	// Get Ollama URL from env or use default
	ollamaURL := os.Getenv("OLLAMA_URL")
	if ollamaURL == "" {
		ollamaURL = "http://localhost:11434" // default Ollama URL
	}

	return &OllamaModel{
		OpenAICompatibleModel: openai_base.OpenAICompatibleModel{
			ResponseHandler:   responseHandler,
			HistoryRepository: historyRepository,
		},
		ModelVersion: modelName,
		ollamaURL:    ollamaURL,
	}
}

func (model *OllamaModel) SetResponseHandler(responseHandler commontypes.ResponseHandler) {
	model.ResponseHandler = responseHandler
}

func (model *OllamaModel) CreateRequest(context *data.Context, prompt string, streaming bool, history []data.History, modifiers *commontypes.PayloadModifiers) *http.Request {
	model.Prompt = prompt
	model.AccumulatedAnswer = ""
	model.ContextId = context.Id
	model.Context = context
	model.StreamedToolCalls = make(map[int]*openai_base.StreamingToolCall)
	model.ModelName = model.ModelVersion
	model.Modifiers = modifiers

	payload := openai_base.CreatePayload(prompt, streaming, history, modifiers, model.ModelVersion, 8000, context)
	return model.createRequest(payload)
}

func (model *OllamaModel) HandleStreamedLine(line []byte) {
	model.OpenAICompatibleModel.HandleStreamedLine(line, model)
}

func (model *OllamaModel) HandleBodyBytes(bytes []byte) {
	model.OpenAICompatibleModel.HandleBodyBytes(bytes, model)
}

func (model *OllamaModel) createRequest(payload interface{}) *http.Request {
	// Ollama doesn't require an API key for local instances
	// But we'll check for one in case someone is using a remote Ollama instance
	apiKey := os.Getenv("OLLAMA_API_KEY")

	jsonpayload, err := json.Marshal(payload)
	if err != nil {
		panic("failed to marshal payload")
	}

	// Use OpenAI-compatible endpoint
	url := fmt.Sprintf("%s/v1/chat/completions", model.ollamaURL)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonpayload))
	if err != nil {
		panic(fmt.Errorf("failed to create request: %v", err))
	}

	req.Header.Set("Content-Type", "application/json")

	// Add authorization header if API key is provided
	if apiKey != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))
	}

	return req
}
