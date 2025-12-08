package claude_model

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	data "owl/data"
	"owl/logger"
	models "owl/models"
	"owl/services"
	"owl/tools"
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
		model_version = "claude-opus-4-5-20251101"
	case "sonnet":
		model_version = "claude-sonnet-4-5-20250929"
	default:
		model_version = "claude-sonnet-4-5-20250929"
	}
	payload := createClaudePayload(prompt, streaming, history, model_version, model.UseThinking, context, modifiers)
	model.Prompt = prompt
	model.AccumulatedAnswer = ""
	model.Context = context

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
			model.ResponseHandler.FinalText(model.Context.Id, model.Prompt, model.AccumulatedAnswer, "", "")
		}
		//TODO: catch the token count response
	}
}

func (model *ClaudeModel) HandleBodyBytes(bytes []byte) {
	var apiResponse MessageResponse

	logger.Debug.Printf("bytes %s", string(bytes))
	if err := json.Unmarshal(bytes, &apiResponse); err != nil {
		println(fmt.Sprintf("Error unmarshalling response body: %v\n", err))
		logger.Debug.Println(err)
	}

	logger.Debug.Println("response")
	logger.Debug.Printf("%s", apiResponse)

	textIndex := 0
	toolResponses := []models.ToolResponse{}
	for i, content := range apiResponse.Content {
		if content.Type == "text" {
			textIndex = i
		} else if content.Type == "tool_use" {
			response, err := model.useTool(content)
			if err != nil {
				logger.Debug.Println(err)
			}
			toolResponses = append(toolResponses, response)
		}
		logger.Debug.Printf("i: %d, content: %s", i, content)
	}

	contentJson, err := json.Marshal(apiResponse.Content)
	if err != nil {
		logger.Debug.Printf("Error marshalling json content from response: %s", err)
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

	// Save the assistant response with tool results
	model.ResponseHandler.FinalText(model.Context.Id, model.Prompt, apiResponse.Content[textIndex].Text, string(contentJson), toolResultsJson)

	if len(toolResponses) > 0 {
		// Continue conversation with tool results
		services.AwaitedQuery("", model, model.HistoryRepository, 3, model.Context, &models.PayloadModifiers{
			ToolResponses: toolResponses,
		})
	}
}

func (model *ClaudeModel) useTool(content ResponseMessage) (models.ToolResponse, error) {
	var result string
	var toolInput interface{}

	switch content.Name {
	case "early_bird_track_lookup":
		toolInput = tools.TrackingNumberLookupInput{
			TrackingNumber: content.Input["TrackingNumber"],
		}
	case "image_generator":
		toolInput = tools.ImageInput{
			Prompt:  content.Input["Prompt"],
			Context: model.Context,
		}
	case "issue_list":
		toolInput = tools.IssueListLookupInput{
			Span: content.Input["Span"],
		}
	case "file_list":
		toolInput = tools.FileListInput{
			Filter: content.Input["Filter"],
		}
	case "read_files":
		toolInput = tools.ReadFileInput{
			FileNames: content.Input["FileNames"],
		}
	case "write_file":
		toolInput = tools.FileWriteInput{
			FileName: content.Input["FileName"],
			Content:  content.Input["Content"],
		}
	}

	runner := tools.ToolRunner{ResponseHandler: &model.ResponseHandler, HistoryRepository: &model.HistoryRepository}
	result = runner.RunTool(content.Name, toolInput).(string)

	if result != "" {
		return models.ToolResponse{
			Response:        result,
			Id:              content.Id,
			ResponseMessage: content,
		}, nil

	}

	return models.ToolResponse{Id: content.Id, Response: "error"}, errors.New(fmt.Sprintf("No such tool is defined: %s", content.Name))
}

func createClaudePayload(prompt string, streamed bool, history []data.History, model string, useThinking bool, context *data.Context, modifiers *models.PayloadModifiers) MessageBody {
	logger.Debug.Printf("crateClaudePayload called with responseCount: %d and history count: %d", len(modifiers.ToolResponses), len(history))

	messages := []Message{}

	// Process history and handle tool results
	for i, h := range history {
		j, err := json.Marshal(h)
		if err != nil {
			panic("failed to marshall h")
		}
		logger.Debug.Printf("added history: %s", j)

		messages = append(messages, TextMessage{Role: "user", Content: h.Prompt})
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

			if hasToolUse && i+1 < len(history) {
				toolResultContent := []Content{}
				if h.ToolResults != "" {
					var toolResults []models.ToolResponse
					err := json.Unmarshal([]byte(h.ToolResults), &toolResults)
					if err == nil {
						for _, tr := range toolResults {
							toolResultContent = append(toolResultContent, ToolResponseContent{
								Type:    "tool_result",
								Content: tr.Response,
								IsError: false,
								Id:      tr.Id,
							})
						}
						messages = append(messages, RequestMessage{Role: "user", Content: toolResultContent})
					}
				}
			}
		} else {
			messages = append(messages, TextMessage{Role: "assistant", Content: h.Response})
		}
	}

	if modifiers.Image {
		imageMessage := createImageMessage(prompt, *modifiers)
		messages = append(messages, imageMessage)
	} else if modifiers.Pdf != "" {
		imageMessage := createPdfMessage(prompt, *modifiers)
		messages = append(messages, imageMessage)
	} else {
		content := []Content{}

		if len(modifiers.ToolResponses) > 0 {
			for _, response := range modifiers.ToolResponses {
				content = append(content, ToolResponseContent{
					Type:    "tool_result",
					Content: response.Response,
					IsError: false,
					Id:      response.Id,
				})
			}
		}
		if prompt != "" {
			content = append(content, TextContent{Type: "text", Text: prompt})
		}

		messages = append(messages, RequestMessage{Role: "user", Content: content})
	}

	payload := MessageBody{
		Model:     model,
		Messages:  messages,
		MaxTokens: 20000,
		Stream:    streamed,
	}

	if modifiers.Web {
		payload.Tools = []ToolModel{{Value: getWebSearchTool()}}
	} else {
		list, err := json.Marshal(tools.GetCustomTools())
		if err != nil {
			panic("failed to marshal json from tools definitions")
		}

		var t []Tool
		err = json.Unmarshal(list, &t)
		if err != nil {
			panic("failed to unmarshal json to tools definitions")
		}

		payload.Tools = make([]ToolModel, len(t))
		for i := range t {
			payload.Tools[i] = ToolModel{Value: t[i]}
		}
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

func createImageMessage(prompt string, payloadModifiers models.PayloadModifiers) RequestMessage {
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

func createPdfMessage(prompt string, modifiers models.PayloadModifiers) RequestMessage {
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
