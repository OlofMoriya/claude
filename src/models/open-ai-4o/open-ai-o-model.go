package openai_4o_model

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	commontypes "owl/common_types"
	"owl/data"
	"owl/logger"
	openai_base "owl/models/open-ai-base"
)

type OpenAi4oModel struct {
	openai_base.OpenAICompatibleModel
	ModelVersion string
}

func (model *OpenAi4oModel) SetResponseHandler(responseHandler commontypes.ResponseHandler) {
	model.ResponseHandler = responseHandler
}

func (model *OpenAi4oModel) CreateRequest(context *data.Context, prompt string, streaming bool, history []data.History, modifiers *commontypes.PayloadModifiers) *http.Request {
	model.Prompt = prompt
	model.AccumulatedAnswer = ""
	model.ContextId = context.Id
	model.Context = context
	model.StreamedToolCalls = make(map[int]*openai_base.StreamingToolCall)
	model.Modifiers = modifiers

	modelVersion := "gpt-4o"
	if model.ModelVersion != "" {
		modelVersion = model.ModelVersion
	}
	model.ModelName = modelVersion

	payload := openai_base.CreatePayload(prompt, streaming, history, modifiers, modelVersion, 15000, context)
	return createOpenAI4oRequest(payload)
}

func (model *OpenAi4oModel) HandleStreamedLine(line []byte) {
	model.OpenAICompatibleModel.HandleStreamedLine(line, model)
}

func (model *OpenAi4oModel) HandleBodyBytes(bytes []byte) {
	model.OpenAICompatibleModel.HandleBodyBytes(bytes, model)
}

func createOpenAI4oRequest(payload interface{}) *http.Request {
	apiKey, ok := os.LookupEnv("OPENAI_API_KEY")
	if !ok {
		panic(fmt.Errorf("Could not fetch OPENAI_API_KEY"))
	}

	jsonpayload, err := json.Marshal(payload)
	if err != nil {
		panic("failed to marshal payload")
	}

	logger.Debug.Printf("OpenAI 4o Request Payload:\n%s", string(jsonpayload))

	url := "https://api.openai.com/v1/chat/completions"
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonpayload))
	if err != nil {
		panic(fmt.Errorf("failed to create request: %v", err))
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))

	return req
}
