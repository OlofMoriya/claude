package gemeni_model

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

type GemeniModel struct {
	openai_base.OpenAICompatibleModel
}

func (model *GemeniModel) SetResponseHandler(responseHandler commontypes.ResponseHandler) {
	model.ResponseHandler = responseHandler
}

func (model *GemeniModel) CreateRequest(context *data.Context, prompt string, streaming bool, history []data.History, modifiers *commontypes.PayloadModifiers) *http.Request {
	// Initialize the base model fields
	model.Prompt = prompt
	model.AccumulatedAnswer = ""
	model.ContextId = context.Id
	model.Context = context
	model.StreamedToolCalls = make(map[int]*openai_base.StreamingToolCall)
	model.ModelName = "gemeni"
	model.Modifiers = modifiers

	// Standard chat completions request via Gemini OpenAI-compatible endpoint
	payload := openai_base.CreatePayload(prompt, streaming, history, modifiers, "gemini-3-flash-preview", 16000, context)
	return createGemeniRequest(payload)
}

func (model *GemeniModel) HandleStreamedLine(line []byte) {
	// Delegate to base implementation
	model.OpenAICompatibleModel.HandleStreamedLine(line, model)
}

func (model *GemeniModel) HandleBodyBytes(bytes []byte) {
	// Delegate to standard base implementation
	model.OpenAICompatibleModel.HandleBodyBytes(bytes, model)
}

func createGemeniRequest(payload interface{}) *http.Request {
	apiKey, ok := os.LookupEnv("GEMINI_API_KEY")
	if !ok {
		panic(fmt.Errorf("Could not fetch GEMINI_API_KEY"))
	}

	jsonpayload, err := json.Marshal(payload)
	if err != nil {
		panic("failed to marshal payload")
	}

	logger.Debug.Printf("Gemini Request Payload:\n%s", string(jsonpayload))

	url := "https://generativelanguage.googleapis.com/v1beta/openai/chat/completions"

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonpayload))
	if err != nil {
		panic(fmt.Errorf("failed to create request: %v", err))
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))

	return req
}
