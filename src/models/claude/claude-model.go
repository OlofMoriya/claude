package claude_model

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	commontypes "owl/common_types"
	data "owl/data"
	"owl/logger"
	"owl/mode"
	models "owl/models"
	"owl/services"
	"owl/tools"
	"sort"
	"strings"
)

type ClaudeModel struct {
	HistoryRepository data.HistoryRepository
	ResponseHandler   models.ResponseHandler
	Prompt            string
	AccumulatedAnswer string
	Context           *data.Context
	ModelVersion      string
	OutputThought     bool
	StreamThought     bool
	UseThinking       bool
	UseStreaming      bool

	//Track streamed content
	CurrentEvent     string
	CurrentToolUse   *StreamedToolUse
	StreamedToolUses []StreamedToolUse
}

type StreamedToolUse struct {
	Id    string
	Name  string
	Input string
}

func (model *ClaudeModel) SetResponseHandler(responseHandler models.ResponseHandler) {
	model.ResponseHandler = responseHandler

}

func (model *ClaudeModel) CreateRequest(context *data.Context, prompt string, streaming bool, history []data.History, modifiers *commontypes.PayloadModifiers) *http.Request {

	logger.Debug.Printf("\nMODEL USE: creating claude payload: %s", model.ModelVersion)

	var model_version string
	switch model.ModelVersion {
	case "opus":
		// model_version = "claude-opus-4-5-20251101"
		model_version = "claude-opus-4-6"
	case "sonnet":
		model_version = "claude-sonnet-4-5-20250929"
	case "haiku":
		model_version = "claude-haiku-4-5-20251001"
	default:
		model_version = "claude-sonnet-4-5-20250929"
	}
	payload := createClaudePayload(prompt, streaming, history, model_version, model.UseThinking, context, modifiers)
	model.Prompt = prompt
	model.AccumulatedAnswer = ""
	model.Context = context

	request := createClaudeRequest(payload)

	return request
}

func (model *ClaudeModel) HandleStreamedLine(line []byte) {
	responseLine := string(line)

	// Capture event type
	if strings.HasPrefix(responseLine, "event: ") {
		event, _ := strings.CutPrefix(responseLine, "event: ")
		model.CurrentEvent = strings.TrimSpace(event)
		return
	}

	// Parse data based on current event
	if strings.HasPrefix(responseLine, "data: ") {
		data, _ := strings.CutPrefix(responseLine, "data: ")

		switch model.CurrentEvent {
		case "content_block_start":
			model.handleContentBlockStart(data)
		case "content_block_delta":
			model.handleContentBlockDelta(data)
		case "content_block_stop":
			model.ResponseHandler.RecievedText("\n", nil)
		case "message_delta":
			model.handleMessageDelta(data)
		case "message_stop":

			// model.StreamedToolUses
			fakeResponse := MessageResponse{
				Content: []ResponseMessage{},
			}

			logger.Debug.Printf("Message stop with toolUsesSaved: %v", model.StreamedToolUses)

			//create a faked messageresponse?
			for _, toolUse := range model.StreamedToolUses {
				var input_map map[string]string
				err := json.Unmarshal([]byte(toolUse.Input), &input_map)
				if err != nil {
					panic(fmt.Sprintf("could not marshall %v to json for streamed tool use", toolUse.Input))
				}
				content := ResponseMessage{
					Id:    toolUse.Id,
					Name:  toolUse.Name,
					Input: input_map,
					Type:  "tool_use",
				}
				fakeResponse.Content = append(fakeResponse.Content, content)
			}

			logger.Debug.Printf("Message stop with fakedResponse.Content: %v", fakeResponse.Content)
			tool_responses, tool_result_json := model.handleToolCalls(fakeResponse)
			logger.Debug.Printf("Results from tool calls %v", tool_responses)

			savedContent := ""
			if len(model.StreamedToolUses) > 0 {
				contentJson, err := json.Marshal(fakeResponse.Content)
				if err != nil {
					logger.Debug.Printf("Error marshalling json content from response: %s", err)
				}
				savedContent = string(contentJson)
			}

			model.ResponseHandler.FinalText(model.Context.Id, model.Prompt, model.AccumulatedAnswer, savedContent, tool_result_json, model.ModelVersion)

			if len(tool_responses) > 0 {
				// Continue conversation with tool results
				services.AwaitedQuery("Responding with result", model, model.HistoryRepository, 20, model.Context, &commontypes.PayloadModifiers{
					ToolResponses: tool_responses,
				}, model.ModelVersion)
			}
		}
	}
}

func (model *ClaudeModel) handleContentBlockStart(data string) {
	var response struct {
		Type         string `json:"type"`
		Index        int    `json:"index"`
		ContentBlock struct {
			Type string `json:"type"`
			Id   string `json:"id,omitempty"`
			Name string `json:"name,omitempty"`
		} `json:"content_block"`
	}

	if err := json.Unmarshal([]byte(data), &response); err != nil {
		println(fmt.Sprintf("Error unmarshalling content_block_start: %v", err))
		return
	}

	if response.ContentBlock.Type == "tool_use" {
		model.CurrentToolUse = &StreamedToolUse{}
		model.CurrentToolUse.Id = response.ContentBlock.Id
		model.CurrentToolUse.Name = response.ContentBlock.Name
		model.CurrentToolUse.Input = ""
		logger.Debug.Printf("starting streaming tool use block")
	}
}

func (model *ClaudeModel) handleContentBlockDelta(data string) {
	var response struct {
		Type  string `json:"type"`
		Index int    `json:"index"`
		Delta struct {
			Type        string `json:"type"`
			Text        string `json:"text,omitempty"`
			Thinking    string `json:"thinking,omitempty"`
			PartialJson string `json:"partial_json,omitempty"`
		} `json:"delta"`
	}

	if err := json.Unmarshal([]byte(data), &response); err != nil {
		println(fmt.Sprintf("Error unmarshalling content_block_delta: %v", err))
		return
	}

	if response.Delta.Type == "input_json_delta" {
		// Accumulate tool input
		model.CurrentToolUse.Input += response.Delta.PartialJson
	} else if response.Delta.Type == "text_delta" {
		model.AccumulatedAnswer = model.AccumulatedAnswer + response.Delta.Text
		model.ResponseHandler.RecievedText(response.Delta.Text, nil)
	} else if response.Delta.Type == "thinking_delta" {
		model.AccumulatedAnswer = model.AccumulatedAnswer + response.Delta.Thinking
		if model.StreamThought {
			color := "grey"
			model.ResponseHandler.RecievedText(response.Delta.Thinking, &color)
		}
	} else {
		logger.Debug.Printf("unhandled content block delta arrived of type: %v", response.Delta.Type)
	}
}

func (model *ClaudeModel) handleMessageDelta(data string) {
	var response struct {
		Type  string `json:"type"`
		Delta struct {
			StopReason string `json:"stop_reason"`
		} `json:"delta"`
	}

	if err := json.Unmarshal([]byte(data), &response); err != nil {
		println(fmt.Sprintf("Error unmarshalling message_delta: %v", err))
		return
	}

	if response.Delta.StopReason == "tool_use" {
		// Tool use detected - execute the tool
		model.StreamedToolUses = append(model.StreamedToolUses, *model.CurrentToolUse)

		logger.Debug.Printf("new streaming tool use completed and appended %s", model.CurrentToolUse)
		model.CurrentToolUse = nil
	}

	logger.Debug.Printf("message delta arrived with data: %v", data)
}

func (model *ClaudeModel) HandleBodyBytes(bytes []byte) {
	var apiResponse MessageResponse

	if err := json.Unmarshal(bytes, &apiResponse); err != nil {
		println(fmt.Sprintf("Error unmarshalling response body: %v\n", err))
		logger.Debug.Println(err)
	}

	logger.Debug.Println("response:")
	logger.Debug.Printf("%v", string(bytes))
	// logger.Debug.Printf("%v", apiResponse)

	responseText := ""
	for _, content := range apiResponse.Content {
		if content.Type == "text" {
			responseText += fmt.Sprintf("\n%s", content.Text)
		}
	}

	toolResponses, toolResultsJson := model.handleToolCalls(apiResponse)

	// Save the assistant response with tool results
	contentJson, err := json.Marshal(apiResponse.Content)
	if err != nil {
		logger.Debug.Printf("Error marshalling json content from response: %s", err)
	}

	model.ResponseHandler.FinalText(model.Context.Id, model.Prompt, responseText, string(contentJson), toolResultsJson, model.ModelVersion)

	if len(toolResponses) > 0 {
		// Continue conversation with tool results
		services.AwaitedQuery("Responding with result", model, model.HistoryRepository, 20, model.Context, &commontypes.PayloadModifiers{
			ToolResponses: toolResponses,
		}, model.ModelVersion)
	}
}

func (model *ClaudeModel) handleToolCalls(apiResponse MessageResponse) ([]commontypes.ToolResponse, string) {
	toolResponses := []commontypes.ToolResponse{}
	for i, content := range apiResponse.Content {
		//possible handle web search results too... But I don't know what we should do with it here
		if content.Type == "tool_use" {
			response, err := model.useTool(content)
			if err != nil {
				logger.Debug.Println(err)
			}
			toolResponses = append(toolResponses, response)
		}
		json, _ := json.Marshal(content)
		logger.Debug.Printf("i: %d, content: %v", i, string(json))
	}

	// Marshal tool results
	toolResultsJson := ""
	if len(toolResponses) > 0 {
		toolResultsBytes, err := json.Marshal(toolResponses)
		if err != nil {
			logger.Debug.Printf("Error marshalling tool results: %s", err)
		} else {
			toolResultsJson = string(toolResultsBytes)
		}
	}

	return toolResponses, toolResultsJson
}

func (model *ClaudeModel) useTool(content ResponseMessage) (commontypes.ToolResponse, error) {

	runner := tools.ToolRunner{ResponseHandler: &model.ResponseHandler, HistoryRepository: &model.HistoryRepository, Context: model.Context}
	result, err := runner.ExecuteTool(*model.Context, content.Name, content.Input)

	if result != "" && err == nil {
		return commontypes.ToolResponse{
			Response:        result,
			Id:              content.Id,
			ResponseMessage: content,
		}, nil
	}

	return commontypes.ToolResponse{Id: content.Id, Response: "error"}, fmt.Errorf("No response or err from tool: %s, err: %s", content.Name, err.Error())
}

func getCacheControl() *CacheControl {
	return &CacheControl{Type: "ephemeral"}
}

func createClaudePayload(prompt string, streamed bool, history []data.History, model string, useThinking bool, context *data.Context, modifiers *commontypes.PayloadModifiers) MessageBody {
	logger.Debug.Printf("crateClaudePayload called with responseCount: %d and history count: %d", len(modifiers.ToolResponses), len(history))

	messages := []Message{}

	lastToolResponseIndex := -1
	nextLastToolResponseIndex := -1
	for i := range history {
		if history[i].ToolResults != "" {
			nextLastToolResponseIndex = lastToolResponseIndex
			lastToolResponseIndex = i
		}
	}

	// Process history and handle tool results
	for i, h := range history {
		j, err := json.Marshal(h)
		if err != nil {
			panic("failed to marshall h")
		}
		logger.Debug.Printf("added history: %s", j)

		// Create user message with potential caching

		textContent := TextContent{
			Type: "text",
			Text: h.Prompt,
		}

		if i == 3 {
			textContent.CacheControl = getCacheControl()
		}

		messages = append(messages, RequestMessage{
			Role:    "user",
			Content: []Content{textContent},
		})

		if h.ResponseContent != "" {
			var content []ResponseMessage
			err := json.Unmarshal([]byte(h.ResponseContent), &content)
			if err != nil {
				fmt.Printf(h.ResponseContent)
				panic(err)
			}

			messages = append(messages, HistoricMessage{Role: "assistant", Content: content})

			// Check if this response contains tool_use blocks
			hasToolUse := false
			for _, c := range content {
				if c.Type == "tool_use" {
					hasToolUse = true
					break
				}
			}

			if hasToolUse && h.ToolResults != "" {
				toolResultContent := []Content{}
				var toolResults []commontypes.ToolResponse
				err := json.Unmarshal([]byte(h.ToolResults), &toolResults)
				if err == nil {
					for tr_i, tr := range toolResults {
						content := ToolResponseContent{
							Type:    "tool_result",
							Content: tr.Response,
							IsError: false,
							Id:      tr.Id,
						}
						if (i == lastToolResponseIndex || i == nextLastToolResponseIndex) && tr_i == len(toolResults)-1 {
							content.CacheControl = getCacheControl()
						}

						toolResultContent = append(toolResultContent, content)
					}
					messages = append(messages, RequestMessage{Role: "user", Content: toolResultContent})
				}

			}
		} else {
			content := TextContent{
				Type: "text",
				Text: h.Response,
			}
			if i == 0 {
				content.CacheControl = getCacheControl()
			}
			messages = append(messages, RequestMessage{Role: "assistant", Content: []Content{content}})
		}
	}

	if modifiers.Image {
		imageMessage := createImageMessage(prompt)
		messages = append(messages, imageMessage)
	} else if modifiers.Pdf != "" {
		imageMessage := createPdfMessage(prompt, *modifiers)
		messages = append(messages, imageMessage)
	} else {
		if prompt != "" {
			userContent := TextContent{
				Type: "text",
				Text: prompt,
			}
			messages = append(messages, RequestMessage{
				Role:    "user",
				Content: []Content{userContent},
			})
		}
	}

	logger.Debug.Printf("Messages length: %d", len(messages))

	payload := MessageBody{
		Model:     model,
		Messages:  messages,
		MaxTokens: 20000,
		Stream:    streamed,
	}

	toolsList := tools.GetCustomTools(mode.Mode)
	sort.Slice(toolsList, func(i, j int) bool {
		return toolsList[i].Name < toolsList[j].Name
	})

	payload.Tools = make([]ToolModel, len(toolsList))
	for i := range toolsList {
		payload.Tools[i] = ToolModel{Value: toolsList[i]}
	}

	if modifiers.Web {
		payload.Tools = append(payload.Tools, ToolModel{Value: getWebSearchTool()})
	}

	// Handle system prompt with caching - ALWAYS cache it
	if context != nil && context.SystemPrompt != "" {
		systemContent := SystemContent{
			Type:         "text",
			Text:         context.SystemPrompt,
			CacheControl: getCacheControl(), // Always cache system prompt
		}
		payload.System = []SystemContent{systemContent}
		logger.Debug.Printf("Applying cache control to system prompt")
	}

	if useThinking {
		payload.Thinking = &ThinkingBlock{
			Type:         "enabled",
			BudgetTokens: 2000,
		}
		payload.Temp = 1
	}

	// logger.Debug.Println("FULL PAYLOAD:")
	// logger.Debug.Printf("\n----\n\n%v\n\n------\n", payload)
	return payload
}

func createImageMessage(prompt string) RequestMessage {
	image, err := services.GetImageFromClipboard()
	if err != nil {
		panic(fmt.Sprintf("could not get image from clipboard, %v", err))
	}
	base64, err := services.ImageToBase64(image)
	if err != nil {
		panic(fmt.Sprintf("could not get base64 from image, %v", err))
	}

	imageMessage := RequestMessage{Role: "user", Content: []Content{
		TextContent{Type: "text", Text: prompt},
		SourceContent{Type: "image", Source: Source{
			Type:      string(Base64),
			MediaType: "image/png",
			Data:      base64,
		}},
	}}
	return imageMessage
}

func createPdfMessage(prompt string, modifiers commontypes.PayloadModifiers) RequestMessage {
	base64, err := services.ReadPDFAsBase64(modifiers.Pdf)
	if err != nil {
		panic(fmt.Sprintf("could not get base64 from pdf, %v", err))
	}
	imageMessage := RequestMessage{Role: "user", Content: []Content{
		TextContent{Type: "text", Text: prompt},
		SourceContent{Type: "document", Source: Source{
			Type:      string(Base64),
			MediaType: "application/pdf",
			Data:      base64,
		}},
	}}
	return imageMessage
}

func getWebSearchTool() BasicTool {
	return BasicTool{
		Type:    "web_search_20250305",
		Name:    "web_search",
		MaxUses: 2,
	}
}

func createClaudeRequest(payload MessageBody) *http.Request {
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
	req.Header.Set("anthropic-beta", "prompt-caching-2024-07-31")

	return req
}
