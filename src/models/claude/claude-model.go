package claude_model

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	data "owl/data"
	"owl/logger"
	models "owl/models"
	"owl/services"
	"strings"
)

type ClaudeModel struct {
	ResponseHandler   models.ResponseHandler
	Prompt            string
	AccumulatedAnswer string
	ContextId         int64
	ModelVersion      string
	OutputThought     bool
	StreamThought     bool
	UseThinking       bool
}

func (model *ClaudeModel) SetResponseHandler(responseHandler models.ResponseHandler) {
	model.ResponseHandler = responseHandler

}

func (model *ClaudeModel) CreateRequest(context *data.Context, prompt string, streaming bool, history []data.History, modifiers *models.PayloadModifiers) *http.Request {
	var model_version string
	switch model.ModelVersion {
	case "3.5-sonnet":
		model_version = "claude-3-5-sonnet-20240620"
	case "3.7-sonnet":
		model_version = "claude-3-7-sonnet-20250219 "
	case "4-sonnet":
		model_version = "claude-sonnet-4-20250514"
	case "opus":
		model_version = "claude-opus-4-20250514"
	case "sonnet":
		model_version = "claude-sonnet-4-5-20250929"
	default:
		model_version = "claude-sonnet-4-5-20250929"
	}
	payload := createClaudePayload(prompt, streaming, history, model_version, model.UseThinking, context, modifiers)
	model.Prompt = prompt
	model.AccumulatedAnswer = ""
	model.ContextId = context.Id

	request := createClaudeRequest(payload, history)
	// fmt.Printf("\nmodel: \n %v", request)

	return request
}

func (model *ClaudeModel) HandleStreamedLine(line []byte) {
	responseLine := string(line)

	logger.Debug.Println(responseLine)

	if strings.HasPrefix(responseLine, "data: ") {
		var apiResponse StreamData
		data, _ := strings.CutPrefix(responseLine, "data: ")
		if err := json.Unmarshal([]byte(data), &apiResponse); err != nil {
			println(fmt.Sprintf("Error unmarshalling response: %v\n %s", err, line))
		}

		// println(data)

		if apiResponse.Type == content_block_delta {
			model.AccumulatedAnswer = model.AccumulatedAnswer + apiResponse.Delta.Text
			if model.OutputThought {
				model.AccumulatedAnswer = model.AccumulatedAnswer + apiResponse.Delta.Thinking
			}
			model.ResponseHandler.RecievedText(apiResponse.Delta.Text, nil)
			if model.StreamThought {
				color := "grey"
				model.ResponseHandler.RecievedText(apiResponse.Delta.Thinking, &color)
			}
		} else if apiResponse.Type == content_block_stop {
			model.ResponseHandler.RecievedText("\n", nil)
		} else if apiResponse.Type == message_stop {
			model.ResponseHandler.FinalText(model.ContextId, model.Prompt, model.AccumulatedAnswer)
		}
		//TODO: catch the token count response
	}
}

func (model *ClaudeModel) HandleBodyBytes(bytes []byte) {
	var apiResponse MessageResponse
	if err := json.Unmarshal(bytes, &apiResponse); err != nil {
		// Handle error, maybe return or log
		println(fmt.Sprintf("Error unmarshalling response body: %v\n", err))
		logger.Debug.Println(err)
	}
	// log.Fatalf("full resposne: %v", apiResponse)

	logger.Debug.Println("response")
	textIndex := 0
	for i, content := range apiResponse.Content {
		if content.Type == "text" {
			textIndex = i
		}
		logger.Debug.Printf("i: %i, content: %s", i, content)
	}

	model.ResponseHandler.FinalText(model.ContextId, model.Prompt, apiResponse.Content[textIndex].Text)
}

func createClaudePayload(prompt string, streamed bool, history []data.History, model string, useThinking bool, context *data.Context, modifiers *models.PayloadModifiers) MessageBody {
	messages := []Message{}
	for _, h := range history {
		messages = append(messages, TextMessage{Role: "user", Content: h.Prompt})
		messages = append(messages, TextMessage{Role: "assistant", Content: h.Response})
	}

	if modifiers.Image {

		image, err := services.GetImageFromClipboard()
		if err != nil {
			panic(fmt.Sprintf("could not get image from clipboard, %v", err))
		}
		base64, err := services.ImageToBase64(image)
		if err != nil {
			panic(fmt.Sprintf("could not get base64 from image, %v", err))
		}

		messages = append(messages, RequestMessage{Role: "user", Content: []Content{
			TextContent{Type: "text", Text: prompt},
			SourceContent{Type: "image", Source: Source{
				Type:      string(Base64),
				MediaType: "image/png",
				Data:      base64,
			}},
		}})
	} else if modifiers.Pdf != "" {
		base64, err := services.ReadPDFAsBase64(modifiers.Pdf)
		if err != nil {
			panic(fmt.Sprintf("could not get base64 from pdf, %v", err))
		}

		messages = append(messages, RequestMessage{Role: "user", Content: []Content{
			TextContent{Type: "text", Text: prompt},
			SourceContent{Type: "document", Source: Source{
				Type:      string(Base64),
				MediaType: "application/pdf",
				Data:      base64,
			}},
		}})
	} else {
		messages = append(messages, RequestMessage{Role: "user", Content: []Content{TextContent{Type: "text", Text: prompt}}})
	}
	// messages = append(messages, TextMessage{Role: "user", Content: prompt})
	payload := MessageBody{
		Model:     model,
		Messages:  messages,
		MaxTokens: 20000,
		Stream:    streamed,
	}

	if modifiers.Web {
		payload.Tools = []Tool{getWebSearchTool()}
	} else {
		payload.Tools = []Tool{}
	}

	if context != nil && context.SystemPrompt != "" {
		payload.System = context.SystemPrompt
	}

	if useThinking {
		payload.Thinking = &ThinkingBlock{
			Type:         "enabled",
			BudgetTokens: 2000,
		}
		payload.Temp = 1
	}

	logger.Debug.Println("FULL PAYLOAD:")
	logger.Debug.Printf("\n%s", payload)
	return payload
}

func getWebSearchTool() Tool {
	return Tool{
		Type:    "web_search_20250305",
		Name:    "web_search",
		MaxUses: 2,
	}
}

func createClaudeRequest(payload MessageBody, history []data.History) *http.Request {
	apiKey, ok := os.LookupEnv("CLAUDE_API_KEY")
	if !ok {
		panic(fmt.Errorf("Could not fetch api key"))
	}

	jsonpayload, err := json.Marshal(payload)
	if err != nil {
		panic("failed to marshal payload")
	}
	logger.Debug.Println("FULL JSON PAYLOAD:")
	logger.Debug.Printf("\n%s", jsonpayload)

	url := "https://api.anthropic.com/v1/messages"

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonpayload))
	if err != nil {
		panic("failed to create request")
	}

	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("anthropic-version", "2023-06-01")

	return req
}
