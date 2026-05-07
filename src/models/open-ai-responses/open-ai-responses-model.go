package open_ai_responses

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image/png"
	"net/http"
	"os"
	commontypes "owl/common_types"
	"owl/data"
	"owl/logger"
	"strings"
	"time"

	"github.com/skratchdot/open-golang/open"
)

var MODELNAME = "open_ai_responses"

type OpenAiResponseModel struct {
	ResponseHandler   commontypes.ResponseHandler
	prompt            string
	accumulatedAnswer string
	contextId         int64
	modelName         string
	ModelVersion      string
}

func (model *OpenAiResponseModel) CreateRequest(context *data.Context, prompt string, streaming bool, history []data.History, modifiers *commontypes.PayloadModifiers) *http.Request {
	payload := createResponsePayload(prompt, streaming, history, modifiers, model.ModelVersion)
	model.prompt = prompt
	model.accumulatedAnswer = ""
	model.contextId = context.Id
	model.modelName = payload.Model
	return createRequest(payload)
}

func createRequest(payload RequestPayload) *http.Request {
	apiKey, ok := os.LookupEnv("OPENAI_API_KEY")
	if !ok {
		panic(fmt.Errorf("Could not fetch api key"))
	}

	jsonpayload, err := json.Marshal(payload)
	logger.Debug.Println("Will send payload")
	logger.Debug.Println(jsonpayload)
	if err != nil {
		panic("failed to marshal payload")
	}

	url := "https://api.openai.com/v1/responses"
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonpayload))
	if err != nil {
		panic(fmt.Errorf("failed to create request: %v", err))
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))

	return req
}

func createResponsePayload(prompt string, streaming bool, history []data.History, modifiers *commontypes.PayloadModifiers, requestedModel string) RequestPayload {
	modelVersion := "gpt-5.3-chat-latest"
	if requestedModel == "gpt-5.5" {
		modelVersion = "gpt-5.5"
	} else if requestedModel == "gpt-5.4" {
		modelVersion = "gpt-5.4"
	}

	tools := []Tool{}
	if modifiers != nil {
		if modifiers.Image {
			tools = append(tools, Tool{Type: "image_generation"})
		}
		if modifiers.Web {
			tools = append(tools, Tool{Type: "web_search"})
			tools = append(tools, Tool{Type: "web_fetch"})
		}
	}
	if len(tools) == 0 {
		tools = append(tools, Tool{Type: "image_generation"})
	}

	request := RequestPayload{
		Model: modelVersion,
		Input: prompt,
		Tools: tools,
	}
	if streaming {
		stream := true
		request.Stream = &stream
	}

	logger.Debug.Println("Will send payload")
	logger.Debug.Printf("request %v", request)
	return request
}

func (model *OpenAiResponseModel) HandleStreamedLine(line []byte) {
	responseLine := string(line)

	if strings.HasPrefix(responseLine, "data: ") {
		data, _ := strings.CutPrefix(responseLine, "data: ")
		data = strings.TrimSpace(data)
		if data == "" || data == "[DONE]" {
			if model.accumulatedAnswer != "" {
				model.ResponseHandler.FinalText(model.contextId, model.prompt, model.accumulatedAnswer, nil, model.modelName, nil)
			}
			return
		}

		var event map[string]interface{}
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			return
		}

		eventType, _ := event["type"].(string)
		switch eventType {
		case "response.output_text.delta", "response.refusal.delta", "response.reasoning_summary_text.delta", "response.content_part.added":
			text := extractEventText(event)
			if text != "" {
				model.accumulatedAnswer += text
				model.ResponseHandler.RecievedText(text, nil)
			}

		case "response.output_text.done", "response.completed":
			if text := extractEventText(event); text != "" && !strings.Contains(model.accumulatedAnswer, text) {
				model.accumulatedAnswer += text
			}
			model.ResponseHandler.FinalText(model.contextId, model.prompt, model.accumulatedAnswer, nil, model.modelName, nil)

		case "response.error":
			if msg, ok := event["message"].(string); ok && strings.TrimSpace(msg) != "" {
				model.ResponseHandler.RecievedText("\nError: "+msg+"\n", nil)
			}
			model.ResponseHandler.FinalText(model.contextId, model.prompt, model.accumulatedAnswer, nil, model.modelName, nil)

		default:
			// Ignore unknown event types to remain resilient.
		}
	}
}

func extractEventText(event map[string]interface{}) string {
	if v, ok := event["delta"].(string); ok {
		return v
	}
	if v, ok := event["text"].(string); ok {
		return v
	}
	if item, ok := event["item"].(map[string]interface{}); ok {
		if content, ok := item["content"].([]interface{}); ok {
			for _, part := range content {
				if m, ok := part.(map[string]interface{}); ok {
					if txt, ok := m["text"].(string); ok {
						return txt
					}
					if delta, ok := m["delta"].(string); ok {
						return delta
					}
				}
			}
		}
	}
	return ""
}

func (model *OpenAiResponseModel) HandleBodyBytes(byte_list []byte) {
	var apiResponse Response
	if err := json.Unmarshal(byte_list, &apiResponse); err != nil {
		// Handle error, maybe return or log
		println(fmt.Sprintf("Error unmarshalling response body: %v\n", err))
	}

	text := ""
	toolUses := []data.ToolUse{}
	toolUseByID := map[string]int{}

	for _, output := range apiResponse.Output {
		logger.Debug.Printf("%s", output)
		switch v := output.(type) {
		case ImageGenerationCall:
			unbased, err := base64.StdEncoding.DecodeString(v.Result)
			if err != nil {
				panic("Cannot decode b64")
			}

			r := bytes.NewReader(unbased)
			im, err := png.Decode(r)
			if err != nil {
				panic("Bad png")
			}

			filename := fmt.Sprintf("/Users/olofmoriya/.owl/img/%d-%s.png", model.contextId, time.Now().Format("2006-01-02:15:04:05"))
			f, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE, 0777)
			if err != nil {
				panic("Cannot open file")
			}

			png.Encode(f, im)

			err = open.Start(filename)
			if err != nil {
				fmt.Printf("Failed to open image: %v\n", err)
			}

			text += fmt.Sprintf("\n![Image](%s)\n", filename)

		case Message:
			for i, content := range v.Content {
				text += content.Text
				if i < (len(v.Content) - 1) {
					text += "\n\n"
				}
			}

		case WebSearchCall:
			actionBytes, _ := json.Marshal(v.Action)
			action := strings.TrimSpace(string(actionBytes))
			if action == "" || action == "null" {
				action = "{}"
			}

			toolUse := data.ToolUse{
				Id:         v.ID,
				Name:       "web_search",
				Input:      action,
				CallerType: "assistant_server",
				Result: data.ToolResult{
					ToolUseId: v.ID,
					Success:   true,
				},
			}
			toolUseByID[v.ID] = len(toolUses)
			toolUses = append(toolUses, toolUse)

		case WebFetchCall:
			actionBytes, _ := json.Marshal(v.Action)
			action := strings.TrimSpace(string(actionBytes))
			if action == "" || action == "null" {
				action = "{}"
			}

			toolUse := data.ToolUse{
				Id:         v.ID,
				Name:       "web_fetch",
				Input:      action,
				CallerType: "assistant_server",
				Result: data.ToolResult{
					ToolUseId: v.ID,
					Success:   true,
				},
			}
			toolUseByID[v.ID] = len(toolUses)
			toolUses = append(toolUses, toolUse)
		}
	}

	if len(toolUses) > 0 {
		resultPayload, _ := json.Marshal(map[string]string{
			"type":        "responses_tool_result",
			"output_text": text,
		})
		for toolUseID, idx := range toolUseByID {
			toolUses[idx].Result = data.ToolResult{
				ToolUseId: toolUseID,
				Content:   string(resultPayload),
				Success:   true,
			}
		}
	}

	logger.Debug.Printf("Final text from responses: %s", text)
	model.ResponseHandler.FinalText(model.contextId, model.prompt, text, toolUses, model.modelName, nil)
}

func (model *OpenAiResponseModel) SetResponseHandler(responseHandler commontypes.ResponseHandler) {
	model.ResponseHandler = responseHandler
}
