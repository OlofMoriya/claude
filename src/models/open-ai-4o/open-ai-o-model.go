package openai_4o_model

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"owl/data"
	"owl/logger"
	"owl/models"
	"owl/services"
	"owl/tools"
	"strings"
)

type OpenAi4oModel struct {
	ResponseHandler   models.ResponseHandler
	prompt            string
	accumulatedAnswer string
	contextId         int64
	context           *data.Context
	HistoryRepository data.HistoryRepository

	// Streaming tool call tracking
	streamedToolCalls map[int]*StreamingToolCall
}

type StreamingToolCall struct {
	Id              string
	Type            string
	FunctionName    string
	ArgumentsBuffer string
}

func (model *OpenAi4oModel) SetResponseHandler(responseHandler models.ResponseHandler) {
	model.ResponseHandler = responseHandler
}

func (model *OpenAi4oModel) CreateRequest(context *data.Context, prompt string, streaming bool, history []data.History, modifiers *models.PayloadModifiers) *http.Request {
	payload := createOpenaiPayload(prompt, streaming, history, modifiers)
	model.prompt = prompt
	model.accumulatedAnswer = ""
	model.contextId = context.Id
	model.context = context
	model.streamedToolCalls = make(map[int]*StreamingToolCall)
	return createRequest(payload, history)
}

func (model *OpenAi4oModel) HandleStreamedLine(line []byte) {
	responseLine := string(line)

	if strings.HasPrefix(responseLine, "data: ") {
		data, _ := strings.CutPrefix(responseLine, "data: ")

		// Skip [DONE] message
		if strings.TrimSpace(data) == "[DONE]" {
			logger.Debug.Println("streamed line incoming with data : DONE")
			model.finishStreaming()
			return
		}

		var apiResponse ChatCompletionChunk
		if err := json.Unmarshal([]byte(data), &apiResponse); err != nil {
			logger.Debug.Printf("Error unmarshalling chunk: %v\n", err)
			return
		}

		logger.Debug.Printf("Chunk: %+v", apiResponse)

		if len(apiResponse.Choices) > 0 {
			choice := apiResponse.Choices[0]

			// Handle tool calls in delta
			if len(choice.Delta.ToolCalls) > 0 {
				model.handleStreamedToolCalls(choice.Delta.ToolCalls)
			}

			// Handle regular content
			if choice.Delta.Content != "" {
				model.accumulatedAnswer += choice.Delta.Content
				model.ResponseHandler.RecievedText(choice.Delta.Content, nil)
			}
		}
	}
}

func (model *OpenAi4oModel) handleStreamedToolCalls(deltaToolCalls []ToolCall) {
	logger.Debug.Printf("Handling streamed tool calls: %+v", deltaToolCalls)

	for _, tc := range deltaToolCalls {
		// Initialize or update tool call
		if _, exists := model.streamedToolCalls[tc.Index]; !exists {
			model.streamedToolCalls[tc.Index] = &StreamingToolCall{
				Id:              tc.Id,
				Type:            tc.Type,
				FunctionName:    tc.Function.Name,
				ArgumentsBuffer: "",
			}
			logger.Debug.Printf("Started new tool call at index %d: %s", tc.Index, tc.Function.Name)
		}

		// Accumulate arguments
		if tc.Function.Arguments != "" {
			model.streamedToolCalls[tc.Index].ArgumentsBuffer += tc.Function.Arguments
			logger.Debug.Printf("Accumulated args for index %d: %s", tc.Index, model.streamedToolCalls[tc.Index].ArgumentsBuffer)
		}
	}
}

func (model *OpenAi4oModel) finishStreaming() {
	logger.Debug.Printf("Finishing streaming with %d tool calls", len(model.streamedToolCalls))

	// Check if we have tool calls
	if len(model.streamedToolCalls) > 0 {
		// Convert to Message format
		message := Message{
			Role:      "assistant",
			Content:   model.accumulatedAnswer,
			ToolCalls: []ToolCall{},
		}

		for idx := 0; idx < len(model.streamedToolCalls); idx++ {
			stc := model.streamedToolCalls[idx]
			message.ToolCalls = append(message.ToolCalls, ToolCall{
				Id:   stc.Id,
				Type: stc.Type,
				Function: FunctionCall{
					Name:      stc.FunctionName,
					Arguments: stc.ArgumentsBuffer,
				},
			})
		}

		logger.Debug.Printf("Executing %d tools", len(message.ToolCalls))

		// Execute tools
		toolResponses, toolResultsJson := model.handleToolCalls(message)

		// Save
		messageJson, err := json.Marshal(message)
		if err != nil {
			logger.Debug.Printf("Error marshalling message: %s", err)
		}

		logger.Debug.Printf("Calling Final Text with anwer: %v, \nand tool result: %v", model.accumulatedAnswer, toolResultsJson)
		model.ResponseHandler.FinalText(model.contextId, model.prompt, model.accumulatedAnswer, string(messageJson), toolResultsJson)

		// Continue with results
		if len(toolResponses) > 0 {
			services.AwaitedQuery("Responding with result", model, model.HistoryRepository, 20, model.context, &models.PayloadModifiers{
				ToolResponses: toolResponses,
			})
		}

		// Reset
		model.streamedToolCalls = make(map[int]*StreamingToolCall)
	} else {
		// Regular finish
		logger.Debug.Printf("Calling Final Text with anwer: %v", model.accumulatedAnswer)
		model.ResponseHandler.FinalText(model.contextId, model.prompt, model.accumulatedAnswer, "", "")
	}
}

func (model *OpenAi4oModel) HandleBodyBytes(bytes []byte) {
	var apiResponse ChatCompletion
	if err := json.Unmarshal(bytes, &apiResponse); err != nil {
		println(fmt.Sprintf("Error unmarshalling response body: %v\n", err))
		return
	}

	logger.Debug.Printf("Response: %+v", apiResponse)

	if len(apiResponse.Choices) == 0 {
		logger.Debug.Println("No choices in response")
		return
	}

	choice := apiResponse.Choices[0]
	message := choice.Message

	// Check for tool calls
	if len(message.ToolCalls) > 0 {
		logger.Debug.Printf("Found %d tool calls", len(message.ToolCalls))
		toolResponses, toolResultsJson := model.handleToolCalls(message)

		// Save assistant message with tool calls
		messageJson, err := json.Marshal(message)
		if err != nil {
			logger.Debug.Printf("Error marshalling message: %s", err)
		}

		model.ResponseHandler.FinalText(model.contextId, model.prompt, message.Content, string(messageJson), toolResultsJson)

		// Continue conversation with tool results
		if len(toolResponses) > 0 {
			services.AwaitedQuery("Responding with result", model, model.HistoryRepository, 20, model.context, &models.PayloadModifiers{
				ToolResponses: toolResponses,
			})
		}
	} else {
		// Regular text response
		model.ResponseHandler.FinalText(model.contextId, model.prompt, message.Content, "", "")
	}
}

func (model *OpenAi4oModel) handleToolCalls(message Message) ([]models.ToolResponse, string) {
	toolResponses := []models.ToolResponse{}

	for _, toolCall := range message.ToolCalls {
		logger.Debug.Printf("Executing tool: %s with args: %s", toolCall.Function.Name, toolCall.Function.Arguments)

		// Parse arguments
		var args map[string]string
		err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args)
		if err != nil {
			logger.Debug.Printf("Error parsing tool arguments: %s", err)
			// Try to handle partial JSON or error gracefully
			toolResponses = append(toolResponses, models.ToolResponse{
				Id:       toolCall.Id,
				Response: fmt.Sprintf("Error parsing arguments: %s", err),
			})
			continue
		}

		// Execute tool
		runner := tools.ToolRunner{
			ResponseHandler:   &model.ResponseHandler,
			HistoryRepository: &model.HistoryRepository,
			Context:           model.context,
		}
		result, err := runner.ExecuteTool(*model.context, toolCall.Function.Name, args)

		if err != nil {
			logger.Debug.Printf("Error executing tool: %s", err)
			result = fmt.Sprintf("Error: %s", err)
		}

		logger.Debug.Printf("Tool result: %s", result)

		toolResponses = append(toolResponses, models.ToolResponse{
			Id:       toolCall.Id,
			Response: result,
		})
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

func createOpenaiPayload(prompt string, streamed bool, history []data.History, modifiers *models.PayloadModifiers) ChatCompletionRequest {
	messages := []interface{}{}

	// Process history (including tool results)
	for i, h := range history {
		questionContent := RequestContent{Type: "text", Text: h.Prompt}
		messages = append(messages, RequestMessage{Role: "user", Content: []RequestContent{questionContent}})

		// Check if this history item has tool calls
		if h.ResponseContent != "" {
			// Try to parse as tool calls
			var assistantMsg Message
			err := json.Unmarshal([]byte(h.ResponseContent), &assistantMsg)
			if err == nil && len(assistantMsg.ToolCalls) > 0 {
				// Add assistant message with tool calls
				messages = append(messages, assistantMsg)

				// Add tool results if available
				if h.ToolResults != "" && i+1 < len(history) {
					var toolResults []models.ToolResponse
					err := json.Unmarshal([]byte(h.ToolResults), &toolResults)
					if err == nil {
						for _, tr := range toolResults {
							messages = append(messages, Message{
								Role:       "tool",
								Content:    tr.Response,
								ToolCallId: tr.Id,
							})
						}
					}
				}
			} else {
				// Regular text response
				answerContent := RequestContent{Type: "text", Text: h.Response}
				messages = append(messages, RequestMessage{Role: "assistant", Content: []RequestContent{answerContent}})
			}
		} else {
			// Regular text response
			answerContent := RequestContent{Type: "text", Text: h.Response}
			messages = append(messages, RequestMessage{Role: "assistant", Content: []RequestContent{answerContent}})
		}
	}

	// Add tool responses if this is a continuation
	if len(modifiers.ToolResponses) > 0 {
		logger.Debug.Printf("Adding %d tool responses to payload", len(modifiers.ToolResponses))
		for _, tr := range modifiers.ToolResponses {
			messages = append(messages, Message{
				Role:       "tool",
				Content:    tr.Response,
				ToolCallId: tr.Id,
			})
		}
	}

	// Add current prompt
	if modifiers.Image {
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
		if prompt != "" {
			messages = append(messages, RequestMessage{Role: "user", Content: []RequestContent{{Type: "text", Text: prompt}}})
		}
	}

	payload := ChatCompletionRequest{
		Model:     "gpt-4o",
		Stream:    streamed,
		Messages:  messages,
		MaxTokens: 15000,
	}

	// Add tools
	customTools := tools.GetCustomTools()
	if len(customTools) > 0 {
		payload.Tools = convertToolsToOpenAIFormat(customTools)
		logger.Debug.Printf("Added %d tools to payload", len(payload.Tools))
	}

	return payload
}

func convertToolsToOpenAIFormat(toolsList []tools.Tool) []FunctionTool {
	openaiTools := make([]FunctionTool, len(toolsList))

	for i, tool := range toolsList {
		// Convert InputSchema to OpenAI parameters format
		parameters := map[string]interface{}{
			"type":       tool.InputSchema.Type,
			"properties": convertProperties(tool.InputSchema.Properties),
		}

		if len(tool.InputSchema.Required) > 0 {
			parameters["required"] = tool.InputSchema.Required
		}

		openaiTools[i] = FunctionTool{
			Type: "function",
			Function: FunctionDefinition{
				Name:        tool.Name,
				Description: tool.Description,
				Parameters:  parameters,
			},
		}

		logger.Debug.Printf("Converted tool: %s", tool.Name)
	}

	return openaiTools
}

func convertProperties(props map[string]tools.Property) map[string]interface{} {
	result := make(map[string]interface{})

	for key, prop := range props {
		propMap := map[string]interface{}{
			"type": prop.Type,
		}
		if prop.Description != "" {
			propMap["description"] = prop.Description
		}
		result[key] = propMap
	}

	return result
}

func createRequest(payload ChatCompletionRequest, history []data.History) *http.Request {
	apiKey, ok := os.LookupEnv("OPENAI_API_KEY")
	if !ok {
		panic(fmt.Errorf("Could not fetch api key"))
	}

	jsonpayload, err := json.Marshal(payload)
	if err != nil {
		panic("failed to marshal payload")
	}

	logger.Debug.Printf("OpenAI Request Payload:\n%s", string(jsonpayload))

	url := "https://api.openai.com/v1/chat/completions"
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonpayload))
	if err != nil {
		panic(fmt.Errorf("failed to create request: %v", err))
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))

	return req
}
