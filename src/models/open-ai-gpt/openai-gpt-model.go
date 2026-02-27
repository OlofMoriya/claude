package openai_gpt_model

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

type OpenAIGPTModel struct {
	openai_base.OpenAICompatibleModel
	ModelVersion string
}

func (model *OpenAIGPTModel) SetResponseHandler(responseHandler commontypes.ResponseHandler) {
	model.ResponseHandler = responseHandler
}

func (model *OpenAIGPTModel) CreateRequest(context *data.Context, prompt string, streaming bool, history []data.History, modifiers *commontypes.PayloadModifiers) *http.Request {
	// Initialize the base model fields
	model.Prompt = prompt
	model.AccumulatedAnswer = ""
	model.ContextId = context.Id
	model.Context = context
	model.StreamedToolCalls = make(map[int]*openai_base.StreamingToolCall)
	model.Modifiers = modifiers

	var model_version string
	switch model.ModelVersion {
	case "codex":
		model_version = "gpt-5.3-codex"
	case "gpt":
		model_version = "gpt-5.3"
	default:
		model_version = "gpt-5.3"
	}
	model.ModelName = model_version

	// Check if web search is enabled - use different API endpoint
	if modifiers.Web {
		logger.Debug.Println("Web search enabled, using /v1/responses endpoint")
		payload := openai_base.CreateWebSearchPayload(prompt, history, "gpt-5", context)
		return createOpenAIGPTRequest(payload, true)
	}

	// Standard chat completions request
	payload := openai_base.CreatePayload(prompt, streaming, history, modifiers, "gpt-5.2", 16000, context)
	return createOpenAIGPTRequest(payload, false)
}

func (model *OpenAIGPTModel) HandleStreamedLine(line []byte) {
	// Delegate to base implementation
	model.OpenAICompatibleModel.HandleStreamedLine(line, model)
}

func (model *OpenAIGPTModel) HandleBodyBytes(bytes []byte) {
	// Try to detect if this is a web search response
	var webSearchResponse openai_base.ResponseAPIResponse
	if err := json.Unmarshal(bytes, &webSearchResponse); err == nil && len(webSearchResponse.Output) > 0 {
		logger.Debug.Println("Detected OpenAI web search response format (output array present)")
		model.OpenAICompatibleModel.HandleWebSearchResponse(bytes, model)
		return
	}

	// Otherwise delegate to standard base implementation
	model.OpenAICompatibleModel.HandleBodyBytes(bytes, model)
}

func createOpenAIGPTRequest(payload interface{}, isWebSearch bool) *http.Request {
	apiKey, ok := os.LookupEnv("OPENAI_API_KEY")
	if !ok {
		panic(fmt.Errorf("Could not fetch OPENAI_API_KEY"))
	}

	jsonpayload, err := json.Marshal(payload)
	if err != nil {
		panic("failed to marshal payload")
	}

	logger.Debug.Printf("OpenAI GPT-5.2 Request Payload:\n%s", string(jsonpayload))

	// Use different endpoint for web search
	url := "https://api.openai.com/v1/chat/completions"
	if isWebSearch {
		url = "https://api.openai.com/v1/responses"
		logger.Debug.Println("Using OpenAI web search endpoint: /v1/responses")
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonpayload))
	if err != nil {
		panic(fmt.Errorf("failed to create request: %v", err))
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))

	return req
}
