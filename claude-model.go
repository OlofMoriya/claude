package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
)

type ClaudeModel struct {
	responseHandler   ResponseHandler
	prompt            string
	accumulatedAnswer string
	contextId         int64
}

func (model *ClaudeModel) createRequest(contextId int64, prompt string, streaming bool, history []History) *http.Request {
	payload := createCaludePayload(prompt, true, history)
	model.prompt = prompt
	model.accumulatedAnswer = ""
	model.contextId = contextId
	return createClaudeRequest(payload, history)
}

func (model *ClaudeModel) handleStreamedLine(line []byte) {
	responseLine := string(line)

	if strings.HasPrefix(responseLine, "data: ") {
		var apiResponse StreamData
		data, _ := strings.CutPrefix(responseLine, "data: ")
		if err := json.Unmarshal([]byte(data), &apiResponse); err != nil {
			fmt.Printf("Error unmarshalling response: %v\n %s", err, line)
		}

		if apiResponse.Type == content_block_delta {
			model.accumulatedAnswer = model.accumulatedAnswer + apiResponse.Delta.Text
			model.responseHandler.recievedText(apiResponse.Delta.Text)
		} else if apiResponse.Type == message_stop {
			model.responseHandler.finalText(model.contextId, model.prompt, model.accumulatedAnswer)
		}
		//TODO: catch the token count response
	}
}

func (model *ClaudeModel) handleBodyBytes(bytes []byte) {
	var apiResponse MessageResponse
	if err := json.Unmarshal(bytes, &apiResponse); err != nil {
		// Handle error, maybe return or log
		fmt.Printf("Error unmarshalling response body: %v\n", err)
	}

	model.responseHandler.finalText(model.contextId, model.prompt, apiResponse.Content[0].Text)
}

func createCaludePayload(prompt string, streamed bool, history []History) MessageBody {
	messages := []Message{}
	for _, h := range history {
		messages = append(messages, TextMessage{Role: "user", Content: h.Prompt})
		messages = append(messages, TextMessage{Role: "assistant", Content: h.Response})
	}
	messages = append(messages, TextMessage{Role: "user", Content: prompt})
	payload := MessageBody{
		Model:     "claude-3-opus-20240229",
		Messages:  messages,
		MaxTokens: 2000,
		Stream:    streamed,
	}

	return payload
}

func createClaudeRequest(payload MessageBody, history []History) *http.Request {
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

type ResponseHandler interface {
	recievedText(text string)
	finalText(contextId int64, prompt string, response string)
	// func recievedImage(encoded string)
}
