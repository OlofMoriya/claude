package ollama_model

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"owl/data"
	"owl/logger"
	"owl/models"
	services "owl/services"
	"strings"
)

type OllamaModel struct {
	ResponseHandler   models.ResponseHandler
	prompt            string
	accumulatedAnswer string
	contextId         int64
	modelName         string
	ollamaURL         string
}

func NewOllamaModel(responseHandler models.ResponseHandler, modelName string) *OllamaModel {
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
		ResponseHandler: responseHandler,
		modelName:       modelName,
		ollamaURL:       ollamaURL,
	}
}

func (model *OllamaModel) SetResponseHandler(responseHandler models.ResponseHandler) {
	model.ResponseHandler = responseHandler
}

func (model *OllamaModel) CreateRequest(context *data.Context, prompt string, streaming bool, history []data.History, modifiers *models.PayloadModifiers) *http.Request {
	payload := model.createOllamaPayload(prompt, streaming, history, modifiers.Image)
	logger.Debug.Printf("created ollama payload: %s", payload)
	model.prompt = prompt
	model.accumulatedAnswer = ""
	model.contextId = context.Id
	return model.createRequest(payload)
}

func (model *OllamaModel) HandleStreamedLine(line []byte) {
	responseLine := string(line)

	if strings.HasPrefix(responseLine, "data: ") {
		var apiResponse ChatCompletionChunk
		data, _ := strings.CutPrefix(responseLine, "data: ")

		// Skip [DONE] message
		if strings.TrimSpace(data) == "[DONE]" {
			return
		}

		logger.Debug.Printf("json")
		logger.Debug.Printf("%s", apiResponse)
		if err := json.Unmarshal([]byte(data), &apiResponse); err != nil {
			logger.Debug.Printf("Error unmarshalling response: %v\n %s", err, line)
			return
		}

		logger.Debug.Printf("%s", apiResponse)

		if len(apiResponse.Choices) > 0 {
			choice := apiResponse.Choices[0]

			model.accumulatedAnswer = model.accumulatedAnswer + choice.Delta.Content
			model.ResponseHandler.RecievedText(choice.Delta.Content, nil)

			if choice.FinishReason != nil {
				fmt.Println(*choice.FinishReason)
				model.ResponseHandler.FinalText(model.contextId, model.prompt, model.accumulatedAnswer, "", "")
			}
		}
	}
}

func (model *OllamaModel) HandleBodyBytes(bytes []byte) {
	var apiResponse ChatCompletion
	if err := json.Unmarshal(bytes, &apiResponse); err != nil {
		println(fmt.Sprintf("Error unmarshalling response body: %v\n", err))
		return
	}

	model.ResponseHandler.FinalText(model.contextId, model.prompt, apiResponse.Choices[0].Message.Content, "", "")
}

func (model *OllamaModel) createOllamaPayload(prompt string, streamed bool, history []data.History, image bool) ChatCompletionRequest {
	messages := []RequestMessage{}

	// Add conversation history
	for _, h := range history {
		questionContent := RequestContent{Type: "text", Text: h.Prompt}
		messages = append(messages, RequestMessage{Role: "user", Content: []RequestContent{questionContent}})
		answerContent := RequestContent{Type: "text", Text: h.Response}
		messages = append(messages, RequestMessage{Role: "assistant", Content: []RequestContent{answerContent}})
	}

	// Add current prompt with optional image
	if image {
		image, err := services.GetImageFromClipboard()
		if err != nil {
			panic(fmt.Sprintf("could not get image from clipboard, %s", err))
		}
		base64, err := services.ImageToBase64(image)
		if err != nil {
			panic(fmt.Sprintf("could not get base64 from image, %s", err))
		}

		messages = append(messages, RequestMessage{Role: "user", Content: []RequestContent{
			{Type: "text", Text: prompt},
			{Type: "image_url", ImageURL: Image{
				URL: fmt.Sprintf("data:image/png;base64,%s", base64),
			}},
		}})
	} else {
		messages = append(messages, RequestMessage{Role: "user", Content: []RequestContent{{Type: "text", Text: prompt}}})
	}

	payload := ChatCompletionRequest{
		Model:    model.modelName,
		Stream:   streamed,
		Messages: messages,
		Tools:    []Tool{},
		// MaxTokens is optional for Ollama
	}

	return payload
}

func (model *OllamaModel) createRequest(payload ChatCompletionRequest) *http.Request {
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
