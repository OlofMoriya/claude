package vertex_claude_model

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	data "owl/data"
	models "owl/models"
	"strings"
)

type ClaudeModel struct {
	ResponseHandler   models.ResponseHandler
	prompt            string
	accumulatedAnswer string
	contextId         int64
}

func (model *ClaudeModel) SetResponseHandler(responseHandler models.ResponseHandler) {
	model.ResponseHandler = responseHandler
}

func (model *ClaudeModel) CreateRequest(contextId int64, prompt string, streaming bool, history []data.History, modifiers *models.PayloadModifiers) *http.Request {
	payload := createClaudePayload(prompt, streaming, history)
	model.prompt = prompt
	model.accumulatedAnswer = ""
	model.contextId = contextId
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
			model.accumulatedAnswer = model.accumulatedAnswer + apiResponse.Delta.Text
			model.ResponseHandler.RecievedText(apiResponse.Delta.Text, nil)
		} else if apiResponse.Type == message_stop {
			model.ResponseHandler.FinalText(model.contextId, model.prompt, model.accumulatedAnswer)
		}
		//TODO: catch the token count response
	} else {
		println(fmt.Sprintf("%v", responseLine))
	}
}

func (model *ClaudeModel) HandleBodyBytes(bytes []byte) {
	var apiResponse MessageResponse
	if err := json.Unmarshal(bytes, &apiResponse); err != nil {
		// Handle error, maybe return or log
		fmt.Printf("Error unmarshalling response body: %v\n", err)
	}

	model.ResponseHandler.FinalText(model.contextId, model.prompt, apiResponse.Content[0].Text)
}

func createClaudePayload(prompt string, streamed bool, history []data.History) VertexMessageBody {
	messages := []Message{}
	for _, h := range history {
		messages = append(messages, TextMessage{Role: "user", Content: h.Prompt})
		messages = append(messages, TextMessage{Role: "assistant", Content: h.Response})
	}
	messages = append(messages, TextMessage{Role: "user", Content: prompt})
	payload := VertexMessageBody{
		AnthropicVersion: "vertex-2023-10-16",
		Messages:         messages,
		MaxTokens:        2000,
		Stream:           streamed,
	}

	return payload
}

func createClaudeRequest(payload VertexMessageBody, history []data.History) *http.Request {
	//use gcloud to fetch the token
	cmd := exec.Command("gcloud", "auth", "print-access-token")
	apiKeyBytes, err := cmd.CombinedOutput()
	//apiKey, ok := os.LookupEnv("VERTEX_CLAUDE_API_KEY")
	apiKey := strings.TrimSpace(string(apiKeyBytes))
	if err != nil {
		panic(fmt.Errorf("Could not fetch api key %v", err))
	}

	jsonpayload, err := json.Marshal(payload)
	if err != nil {
		panic("failed to marshal payload")
	}

	project_id := "sandbox-416509"
	model := "claude-3-sonnet@20240229"
	location := "us-central1"

	url := fmt.Sprintf("https://%s-aiplatform.googleapis.com/v1/projects/%s/locations/%s/publishers/anthropic/models/%s:streamRawPredict", location, project_id, location, model)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonpayload))
	if err != nil {
		panic("failed to create request")
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("anthropic-version", "vertex-2023-10-16")

	return req
}
