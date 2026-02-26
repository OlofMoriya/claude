package grok_model

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	commontypes "owl/common_types"
	"owl/data"
	"owl/logger"
	"owl/models/open-ai-base"
)

type GrokModel struct {
	openai_base.OpenAICompatibleModel
}

func (model *GrokModel) SetResponseHandler(responseHandler commontypes.ResponseHandler) {
	model.ResponseHandler = responseHandler

}

func (model *GrokModel) CreateRequest(context *data.Context, prompt string, streaming bool, history []data.History, modifiers *commontypes.PayloadModifiers) *http.Request {
	// Initialize the base model fields
	model.Prompt = prompt
	model.AccumulatedAnswer = ""
	model.ContextId = context.Id
	model.Context = context
	model.StreamedToolCalls = make(map[int]*openai_base.StreamingToolCall)
	model.ModelName = "grok"

	// Check if web search is enabled - use different API endpoint
	if modifiers.Web {
		logger.Debug.Println("Web search enabled for Grok, using /v1/responses endpoint")
		payload := openai_base.CreateWebSearchPayload(prompt, history, "grok-4-1-fast-reasoning", context)
		return createGrokRequest(payload, true)
	}

	// Standard chat completions request
	payload := openai_base.CreatePayload(prompt, streaming, history, modifiers, "grok-4-1-fast-reasoning", 8000, context)
	return createGrokRequest(payload, false)
}

func (model *GrokModel) HandleStreamedLine(line []byte) {
	// Delegate to base implementation
	model.OpenAICompatibleModel.HandleStreamedLine(line, model)
}

func (model *GrokModel) HandleBodyBytes(bytes []byte) {
	// Try to detect if this is a web search response
	var webSearchResponse openai_base.ResponseAPIResponse
	if err := json.Unmarshal(bytes, &webSearchResponse); err == nil && len(webSearchResponse.Output) > 0 {
		logger.Debug.Println("Detected Grok web search response format (output array present)")
		model.OpenAICompatibleModel.HandleWebSearchResponse(bytes, model)
		return
	}

	// Otherwise delegate to standard base implementation
	model.OpenAICompatibleModel.HandleBodyBytes(bytes, model)
}

func createGrokRequest(payload interface{}, isWebSearch bool) *http.Request {
	apiKey, ok := os.LookupEnv("XAI_API_KEY")
	if !ok {
		panic(fmt.Errorf("Could not fetch XAI_API_KEY"))
	}

	jsonpayload, err := json.Marshal(payload)
	if err != nil {
		panic("failed to marshal payload")
	}

	logger.Debug.Printf("Grok Request Payload:\n%s", string(jsonpayload))

	// Use different endpoint for web search
	url := "https://api.x.ai/v1/chat/completions"
	if isWebSearch {
		url = "https://api.x.ai/v1/responses"
		logger.Debug.Println("Using Grok web search endpoint: /v1/responses")
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonpayload))
	if err != nil {
		panic(fmt.Errorf("failed to create request: %v", err))
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))

	return req
}
