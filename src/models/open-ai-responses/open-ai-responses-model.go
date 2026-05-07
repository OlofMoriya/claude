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
	"owl/openai_auth"
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
	auth, authErr := openai_auth.Resolve()
	if authErr != nil {
		panic(fmt.Errorf("could not resolve OpenAI auth: %w", authErr))
	}

	payload := createResponsePayload(context, prompt, streaming, history, modifiers, model.ModelVersion, auth.IsCodex)
	model.prompt = prompt
	model.accumulatedAnswer = ""
	model.contextId = context.Id
	model.modelName = payload.Model
	return createRequest(payload, auth)
}

func createRequest(payload RequestPayload, auth openai_auth.ResolvedAuth) *http.Request {
	jsonpayload, err := json.Marshal(payload)
	logger.Debug.Println("Will send payload")
	logger.Debug.Println(jsonpayload)
	if err != nil {
		panic("failed to marshal payload")
	}

	url := "https://api.openai.com/v1/responses"
	if auth.IsCodex {
		url = "https://chatgpt.com/backend-api/codex/responses"
	}
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonpayload))
	if err != nil {
		panic(fmt.Errorf("failed to create request: %v", err))
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", auth.Token))
	if auth.AccountID != "" {
		req.Header.Set("ChatGPT-Account-Id", auth.AccountID)
	}

	return req
}

func createResponsePayload(context *data.Context, prompt string, streaming bool, history []data.History, modifiers *commontypes.PayloadModifiers, requestedModel string, codexAuth bool) RequestPayload {
	modelVersion := "gpt-5.3-chat-latest"
	switch requestedModel {
	case "codex":
		modelVersion = "gpt-5.3-codex"
	case "gpt":
		modelVersion = "gpt-5.3-chat-latest"
	case "gpt-5.4":
		modelVersion = "gpt-5.4"
	case "gpt-5.5":
		modelVersion = "gpt-5.5"
	case "gpt-mini":
		modelVersion = "gpt-5.4-mini-2026-03-17"
	case "gpt-nano":
		modelVersion = "gpt-5.4-nano-2026-03-17"
	}
	if codexAuth && (requestedModel == "gpt" || requestedModel == "") {
		modelVersion = "gpt-5.3-codex"
	}

	input := make([]InputItem, 0, len(history)*2+1)
	for _, h := range history {
		if strings.TrimSpace(h.Prompt) != "" {
			input = append(input, InputItem{
				Role: "user",
				Content: []ContentBlock{{
					Type: "input_text",
					Text: h.Prompt,
				}},
			})
		}
		if strings.TrimSpace(h.Response) != "" {
			input = append(input, InputItem{
				Role: "assistant",
				Content: []ContentBlock{{
					Type: "output_text",
					Text: h.Response,
				}},
			})
		}
	}
	input = append(input, InputItem{
		Role: "user",
		Content: []ContentBlock{{
			Type: "input_text",
			Text: prompt,
		}},
	})

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

	request := RequestPayload{
		Model: modelVersion,
		Input: input,
		Tools: tools,
	}
	if codexAuth {
		store := false
		request.Store = &store
		instructions := ""
		if context != nil {
			instructions = strings.TrimSpace(context.SystemPrompt)
		}
		if instructions == "" {
			instructions = "You are Owl, a coding assistant that prioritizes safe, minimal, and verifiable changes while following repository conventions."
		}
		request.Instructions = instructions
	}

	logger.Debug.Println("Will send payload")
	logger.Debug.Printf("request %v", request)
	return request
}

func (model *OpenAiResponseModel) HandleStreamedLine(line []byte) {
	responseLine := string(line)

	// fmt.Printf("\n\n%v\n", responseLine)
	if strings.HasPrefix(responseLine, "data: ") {
		var apiResponse ChatCompletionChunk
		data, _ := strings.CutPrefix(responseLine, "data: ")

		logger.Debug.Printf("json")
		logger.Debug.Printf("%v", apiResponse)
		if err := json.Unmarshal([]byte(data), &apiResponse); err != nil {
			// fmt.Printf("Error unmarshalling response: %v\n %s", err, line)
		}

		if len(apiResponse.Choices) > 0 {
			choice := apiResponse.Choices[0]

			model.accumulatedAnswer = model.accumulatedAnswer + choice.Delta.Content
			model.ResponseHandler.RecievedText(choice.Delta.Content, nil)

			if choice.FinishReason != nil {
				fmt.Println(*&choice.FinishReason)
				model.ResponseHandler.FinalText(model.contextId, model.prompt, model.accumulatedAnswer, nil, model.modelName, nil)
			}
		}
	}
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
