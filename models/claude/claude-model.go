package claude_model

import (
	"bytes"
	data "claude/data"
	models "claude/models"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
)

type ClaudeModel struct {
	ResponseHandler   models.ResponseHandler
	Prompt            string
	AccumulatedAnswer string
	ContextId         int64
	ModelVersion      string
}

func (model *ClaudeModel) CreateRequest(contextId int64, prompt string, streaming bool, history []data.History) *http.Request {
	var model_version string
	switch model.ModelVersion {
	case "3":
		model_version = "claude-3-opus-20240229"
	case "3.5-sonnet":
		model_version = "claude-3-5-sonnet-20240620"
	default:
		model_version = "claude-3-5-sonnet-20240620"
	}
	payload := createCaludePayload(prompt, streaming, history, model_version)
	model.Prompt = prompt
	model.AccumulatedAnswer = ""
	model.ContextId = contextId
	return createClaudeRequest(payload, history)
}

func (model *ClaudeModel) HandleStreamedLine(line []byte) {
	responseLine := string(line)

	if strings.HasPrefix(responseLine, "data: ") {
		var apiResponse StreamData
		data, _ := strings.CutPrefix(responseLine, "data: ")
		if err := json.Unmarshal([]byte(data), &apiResponse); err != nil {
			println(fmt.Sprintf("Error unmarshalling response: %v\n %s", err, line))
		}

		if apiResponse.Type == content_block_delta {
			model.AccumulatedAnswer = model.AccumulatedAnswer + apiResponse.Delta.Text
			model.ResponseHandler.RecievedText(apiResponse.Delta.Text)
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
	}

	model.ResponseHandler.FinalText(model.ContextId, model.Prompt, apiResponse.Content[0].Text)
}

func createCaludePayload(prompt string, streamed bool, history []data.History, model string) MessageBody {
	messages := []Message{}
	for _, h := range history {
		messages = append(messages, TextMessage{Role: "user", Content: h.Prompt})
		messages = append(messages, TextMessage{Role: "assistant", Content: h.Response})
	}
	messages = append(messages, TextMessage{Role: "user", Content: prompt})
	payload := MessageBody{
		Model:     model,
		Messages:  messages,
		MaxTokens: 2000,
		Stream:    streamed,
	}

	return payload
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
