package grok_model

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"owl/data"
	"owl/logger"
	"owl/models"
	"owl/models/open-ai-base"
)

type GrokModel struct {
	openai_base.OpenAICompatibleModel
}

func (model *GrokModel) SetResponseHandler(responseHandler models.ResponseHandler) {
	model.ResponseHandler = responseHandler
}

func (model *GrokModel) CreateRequest(context *data.Context, prompt string, streaming bool, history []data.History, modifiers *models.PayloadModifiers) *http.Request {
	// Initialize the base model fields
	model.Prompt = prompt
	model.AccumulatedAnswer = ""
	model.ContextId = context.Id
	model.Context = context
	model.StreamedToolCalls = make(map[int]*openai_base.StreamingToolCall)

	// Create payload using base implementation with Grok-specific settings
	payload := openai_base.CreatePayload(prompt, streaming, history, modifiers, "grok-3", 8000)

	return createGrokRequest(payload)
}

func (model *GrokModel) HandleStreamedLine(line []byte) {
	// Delegate to base implementation
	model.OpenAICompatibleModel.HandleStreamedLine(line, model)
}

func (model *GrokModel) HandleBodyBytes(bytes []byte) {
	// Delegate to base implementation
	model.OpenAICompatibleModel.HandleBodyBytes(bytes, model)
}

func createGrokRequest(payload openai_base.ChatCompletionRequest) *http.Request {
	apiKey, ok := os.LookupEnv("XAI_API_KEY")
	if !ok {
		panic(fmt.Errorf("Could not fetch XAI_API_KEY"))
	}

	jsonpayload, err := json.Marshal(payload)
	if err != nil {
		panic("failed to marshal payload")
	}

	logger.Debug.Printf("Grok Request Payload:\n%s", string(jsonpayload))

	url := "https://api.x.ai/v1/chat/completions"
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonpayload))
	if err != nil {
		panic(fmt.Errorf("failed to create request: %v", err))
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))

	return req
}
