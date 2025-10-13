package grok_model

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"owl/data"
	"owl/models"
	"owl/services"
	"strings"
)

type GrokModel struct {
	ResponseHandler   models.ResponseHandler
	prompt            string
	accumulatedAnswer string
	contextId         int64
}

func (model *GrokModel) SetResponseHandler(responseHandler models.ResponseHandler) {
	model.ResponseHandler = responseHandler

}

func (model *GrokModel) CreateRequest(context *data.Context, prompt string, streaming bool, history []data.History, modifiers *models.PayloadModifiers) *http.Request {
	payload := createGrokPayload(prompt, streaming, history, modifiers.Image)
	model.prompt = prompt
	model.accumulatedAnswer = ""
	model.contextId = context.Id
	return createRequest(payload, history)
}

func (model *GrokModel) HandleStreamedLine(line []byte) {
	responseLine := string(line)

	// fmt.Printf("\n\n%v\n", responseLine)
	if strings.HasPrefix(responseLine, "data: ") {
		var apiResponse ChatCompletionChunk
		data, _ := strings.CutPrefix(responseLine, "data: ")
		if err := json.Unmarshal([]byte(data), &apiResponse); err != nil {
			// fmt.Printf("Error unmarshalling response: %v\n %s", err, line)
		}

		if len(apiResponse.Choices) > 0 {
			choice := apiResponse.Choices[0]

			model.accumulatedAnswer = model.accumulatedAnswer + choice.Delta.Content
			model.ResponseHandler.RecievedText(choice.Delta.Content, nil)

			if choice.FinishReason != nil {
				fmt.Println(*&choice.FinishReason)
				model.ResponseHandler.FinalText(model.contextId, model.prompt, model.accumulatedAnswer)
			}
		}
	}
}

func (model *GrokModel) HandleBodyBytes(bytes []byte) {
	var apiResponse ChatCompletion
	if err := json.Unmarshal(bytes, &apiResponse); err != nil {
		// Handle error, maybe return or log
		println(fmt.Sprintf("Error unmarshalling response body: %v\n", err))
	}

	model.ResponseHandler.FinalText(model.contextId, model.prompt, apiResponse.Choices[0].Message.Content)
}

func createGrokPayload(prompt string, streamed bool, history []data.History, image bool) ChatCompletionRequest {
	messages := []RequestMessage{}
	for _, h := range history {
		questionContent := RequestContent{Type: "text", Text: h.Prompt}
		messages = append(messages, RequestMessage{Role: "user", Content: []RequestContent{questionContent}})
		answerContent := RequestContent{Type: "text", Text: h.Response}
		messages = append(messages, RequestMessage{Role: "assistant", Content: []RequestContent{answerContent}})
	}

	// messages = append(messages, RequestMessage{Role: "user", Content: []RequestContent{{Type: "text", Text: prompt}}})

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
		Model:     "grok-4",
		Stream:    streamed,
		Messages:  messages,
		MaxTokens: 2000,
	}

	return payload
}

func createRequest(payload ChatCompletionRequest, history []data.History) *http.Request {
	//use gcloud to fetch the token
	apiKey, ok := os.LookupEnv("XAI_API_KEY")
	if !ok {
		panic(fmt.Errorf("Could not fetch api key"))
	}
	// fmt.Printf("\nkey: -%s-", apiKey)

	jsonpayload, err := json.Marshal(payload)
	if err != nil {
		panic("failed to marshal payload")
	}

	url := "https://api.x.ai/v1/chat/completions"
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonpayload))
	if err != nil {
		panic(fmt.Errorf("failed to create request: %v", err))
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))

	return req
}
