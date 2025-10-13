package openai_vision_model

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"owl/data"
	"owl/models"
	"strings"
)

type OpenAiModel struct {
	ResponseHandler   models.ResponseHandler
	prompt            string
	accumulatedAnswer string
	contextId         int64
}

func (model *OpenAiModel) SetResponseHandler(responseHandler models.ResponseHandler) {
	model.ResponseHandler = responseHandler

}

func (model *OpenAiModel) CreateRequest(context data.Context, prompt string, streaming bool, history []data.History, modifiers *models.PayloadModifiers) *http.Request {
	payload := createOpenaiPayload(prompt, streaming, history)
	model.prompt = prompt
	model.accumulatedAnswer = ""
	model.contextId = context.Id
	return createRequest(payload, history)
}

func (model *OpenAiModel) HandleStreamedLine(line []byte) {
	responseLine := string(line)

	// fmt.Printf("\n\n%v\n", responseLine)
	if strings.HasPrefix(responseLine, "data: ") {
		var apiResponse StreamData
		data, _ := strings.CutPrefix(responseLine, "data: ")
		if err := json.Unmarshal([]byte(data), &apiResponse); err != nil {
			// fmt.Printf("Error unmarshalling response: %v\n %s", err, line)
		}

		if len(apiResponse.Choices) > 0 {
			choice := apiResponse.Choices[0]

			model.accumulatedAnswer = model.accumulatedAnswer + choice.Delta.Content
			model.ResponseHandler.RecievedText(choice.Delta.Content, nil)

			if choice.FinishReason != nil {
				fmt.Println(*choice.FinishReason)
				model.ResponseHandler.FinalText(model.contextId, model.prompt, model.accumulatedAnswer)
			}
		}
	}
}

func (model *OpenAiModel) HandleBodyBytes(bytes []byte) {
	var apiResponse ApiResponse
	if err := json.Unmarshal(bytes, &apiResponse); err != nil {
		// Handle error, maybe return or log
		println(fmt.Sprintf("Error unmarshalling response body: %v\n", err))
	}

	model.ResponseHandler.FinalText(model.contextId, model.prompt, apiResponse.Choices[0].Message.Content)
}

func createOpenaiPayload(prompt string, streamed bool, history []data.History) Payload {
	messages := []Message{}
	for _, h := range history {
		questionContent := Content{Type: "text", Text: h.Prompt}
		messages = append(messages, Message{Role: "user", Content: []Content{questionContent}})
		answerContent := Content{Type: "text", Text: h.Response}
		messages = append(messages, Message{Role: "assistant", Content: []Content{answerContent}})
	}

	messages = append(messages, Message{Role: "user", Content: []Content{{Type: "text", Text: prompt}}})
	payload := Payload{
		Model:     "gpt-4-vision-preview",
		Stream:    streamed,
		Messages:  messages,
		MaxTokens: 2000,
	}

	return payload
}

func createRequest(payload Payload, history []data.History) *http.Request {
	//use gcloud to fetch the token
	apiKey, ok := os.LookupEnv("OPENAI_API_KEY")
	if !ok {
		panic(fmt.Errorf("Could not fetch api key"))
	}
	// fmt.Printf("\nkey: -%s-", apiKey)

	jsonpayload, err := json.Marshal(payload)
	if err != nil {
		panic("failed to marshal payload")
	}

	url := "https://api.openai.com/v1/chat/completions"
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonpayload))
	if err != nil {
		panic(fmt.Errorf("failed to create request: %v", err))
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))

	return req
}
