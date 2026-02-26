package openai_base

import (
	"encoding/json"
	"fmt"
	commontypes "owl/common_types"
	"owl/data"
	"owl/logger"
	"owl/mode"
	"owl/services"
	"owl/tools"
	"strings"
)

// StreamingToolCall tracks tool calls being accumulated during streaming
type StreamingToolCall struct {
	Id              string
	Type            string
	FunctionName    string
	ArgumentsBuffer string
}

// OpenAICompatibleModel provides shared functionality for OpenAI-compatible APIs
type OpenAICompatibleModel struct {
	ResponseHandler   commontypes.ResponseHandler
	HistoryRepository data.HistoryRepository
	Prompt            string
	AccumulatedAnswer string
	ContextId         int64
	Context           *data.Context
	StreamedToolCalls map[int]*StreamingToolCall
	ModelName         string
}

// HandleStreamedLine processes a single line from a streaming response
func (model *OpenAICompatibleModel) HandleStreamedLine(line []byte, callback_model commontypes.Model) {
	responseLine := string(line)

	if strings.HasPrefix(responseLine, "data: ") {
		data, _ := strings.CutPrefix(responseLine, "data: ")

		// Skip [DONE] message
		if strings.TrimSpace(data) == "[DONE]" {
			logger.Debug.Println("Streamed line incoming with data: DONE")
			model.FinishStreaming(callback_model)
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
				model.HandleStreamedToolCalls(choice.Delta.ToolCalls)
			}

			// Handle regular content
			if choice.Delta.Content != "" {
				model.AccumulatedAnswer += choice.Delta.Content
				model.ResponseHandler.RecievedText(choice.Delta.Content, nil)
			}
		}
	} else {
		logger.Debug.Printf("streamed response without data: prefix:  %s\n", responseLine)
	}
}

// HandleStreamedToolCalls accumulates tool call data during streaming
func (model *OpenAICompatibleModel) HandleStreamedToolCalls(deltaToolCalls []ToolCall) {
	logger.Debug.Printf("Handling streamed tool calls: %+v", deltaToolCalls)

	for _, tc := range deltaToolCalls {
		// Initialize or update tool call
		if _, exists := model.StreamedToolCalls[tc.Index]; !exists {
			model.StreamedToolCalls[tc.Index] = &StreamingToolCall{
				Id:              tc.Id,
				Type:            tc.Type,
				FunctionName:    tc.Function.Name,
				ArgumentsBuffer: "",
			}
			logger.Debug.Printf("Started new tool call at index %d: %s", tc.Index, tc.Function.Name)
		}

		// Accumulate arguments
		if tc.Function.Arguments != "" {
			model.StreamedToolCalls[tc.Index].ArgumentsBuffer += tc.Function.Arguments
			logger.Debug.Printf("Accumulated args for index %d: %s", tc.Index, model.StreamedToolCalls[tc.Index].ArgumentsBuffer)
		}
	}
}

// FinishStreaming completes the streaming response and executes any tool calls
func (model *OpenAICompatibleModel) FinishStreaming(callback_model commontypes.Model) {
	logger.Debug.Printf("Finishing streaming with %d tool calls", len(model.StreamedToolCalls))

	// Check if we have tool calls
	if len(model.StreamedToolCalls) > 0 {
		// Convert to Message format
		message := Message{
			Role:      "assistant",
			Content:   model.AccumulatedAnswer,
			ToolCalls: []ToolCall{},
		}

		for idx := 0; idx < len(model.StreamedToolCalls); idx++ {
			stc := model.StreamedToolCalls[idx]
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
		toolResponses, toolResultsJson := model.HandleToolCalls(message)

		// Save
		messageJson, err := json.Marshal(message)
		if err != nil {
			logger.Debug.Printf("Error marshalling message: %s", err)
		}

		logger.Debug.Printf("Calling Final Text with answer: %v, \nand tool result: %v", model.AccumulatedAnswer, toolResultsJson)
		model.ResponseHandler.FinalText(model.ContextId, model.Prompt, model.AccumulatedAnswer, string(messageJson), toolResultsJson, model.ModelName)

		// Continue with results
		if len(toolResponses) > 0 {
			services.AwaitedQuery("Responding with result", callback_model, model.HistoryRepository, 20, model.Context, &commontypes.PayloadModifiers{
				ToolResponses: toolResponses,
			}, model.ModelName)
		}

		// Reset
		model.StreamedToolCalls = make(map[int]*StreamingToolCall)
	} else {
		// Regular finish
		logger.Debug.Printf("Calling Final Text with answer: %v", model.AccumulatedAnswer)
		model.ResponseHandler.FinalText(model.ContextId, model.Prompt, model.AccumulatedAnswer, "", "", model.ModelName)
	}
}

// HandleBodyBytes processes a complete (non-streaming) response
func (model *OpenAICompatibleModel) HandleBodyBytes(bytes []byte, callback_model commontypes.Model) {
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
		toolResponses, toolResultsJson := model.HandleToolCalls(message)

		// Save assistant message with tool calls
		messageJson, err := json.Marshal(message)
		if err != nil {
			logger.Debug.Printf("Error marshalling message: %s", err)
		}

		model.ResponseHandler.FinalText(model.ContextId, model.Prompt, message.Content, string(messageJson), toolResultsJson, model.ModelName)

		// Continue conversation with tool results
		if len(toolResponses) > 0 {
			services.AwaitedQuery("Responding with result", callback_model, model.HistoryRepository, 20, model.Context, &commontypes.PayloadModifiers{
				ToolResponses: toolResponses,
			}, model.ModelName)
		}
	} else {
		// Regular text response
		model.ResponseHandler.FinalText(model.ContextId, model.Prompt, message.Content, "", "", model.ModelName)
	}
}

// HandleToolCalls executes the requested tools and returns their results
func (model *OpenAICompatibleModel) HandleToolCalls(message Message) ([]commontypes.ToolResponse, string) {
	toolResponses := []commontypes.ToolResponse{}

	for _, toolCall := range message.ToolCalls {
		logger.Debug.Printf("Executing tool: %s with args: %s", toolCall.Function.Name, toolCall.Function.Arguments)

		// Parse arguments
		var args map[string]string
		err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args)
		if err != nil {
			logger.Debug.Printf("Error parsing tool arguments: %s", err)
			// Try to handle partial JSON or error gracefully
			toolResponses = append(toolResponses, commontypes.ToolResponse{
				Id:       toolCall.Id,
				Response: fmt.Sprintf("Error parsing arguments: %s", err),
			})
			continue
		}

		// Execute tool
		runner := tools.ToolRunner{
			ResponseHandler:   &model.ResponseHandler,
			HistoryRepository: &model.HistoryRepository,
			Context:           model.Context,
		}
		result, err := runner.ExecuteTool(*model.Context, toolCall.Function.Name, args)

		if err != nil {
			logger.Debug.Printf("Error executing tool: %s", err)
			result = fmt.Sprintf("Error: %s", err)
		}

		logger.Debug.Printf("Tool result: %s", result)

		toolResponses = append(toolResponses, commontypes.ToolResponse{
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

// CreatePayload builds the request payload with history and tool definitions
func CreatePayload(prompt string, streamed bool, history []data.History, modifiers *commontypes.PayloadModifiers, model string, maxTokens int, context *data.Context) ChatCompletionRequest {
	logger.Debug.Printf("\nMODEL USE: creating grok payload: %s", "PLACEHOLDER FROM GROK")

	messages := []interface{}{}

	if context.SystemPrompt != "" {
		messages = append(messages, RequestMessage{
			Role:    "developer",
			Content: context.SystemPrompt,
		})
	}

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
					var toolResults []commontypes.ToolResponse
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
		Model:     model,
		Stream:    streamed,
		Messages:  messages,
		MaxTokens: maxTokens,
	}

	// Add tools
	customTools := tools.GetCustomTools(mode.Mode)
	if len(customTools) > 0 {
		payload.Tools = ConvertToolsToOpenAIFormat(customTools)
		logger.Debug.Printf("Added %d tools to payload", len(payload.Tools))
	}

	return payload
}

// ConvertToolsToOpenAIFormat converts tool definitions to OpenAI function calling format
func ConvertToolsToOpenAIFormat(toolsList []tools.Tool) []FunctionTool {
	openaiTools := make([]FunctionTool, len(toolsList))

	for i, tool := range toolsList {
		// Convert InputSchema to OpenAI parameters format
		parameters := map[string]interface{}{
			"type":       tool.InputSchema.Type,
			"properties": ConvertProperties(tool.InputSchema.Properties),
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

// ConvertProperties converts tool property definitions
func ConvertProperties(props map[string]tools.Property) map[string]interface{} {
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

// HandleWebSearchResponse processes responses from the /v1/responses endpoint
func (model *OpenAICompatibleModel) HandleWebSearchResponse(bytes []byte, callback_model commontypes.Model) {
	var webSearchResponse ResponseAPIResponse
	if err := json.Unmarshal(bytes, &webSearchResponse); err != nil {
		logger.Debug.Printf("Error unmarshalling web search response: %v\n", err)
		// Fall back to regular response handling
		model.HandleBodyBytes(bytes, callback_model)
		return
	}

	logger.Debug.Printf("Web search response status: %s, output items: %d", webSearchResponse.Status, len(webSearchResponse.Output))

	// Parse the output array to find the message
	var responseText string
	allAnnotations := []Annotation{}
	webSearchId := ""
	messageId := ""

	// Process the output array (Grok uses 'output', not 'items')
	for _, outputItem := range webSearchResponse.Output {
		// Each item needs to be parsed as a ResponseItem
		itemBytes, err := json.Marshal(outputItem)
		if err != nil {
			logger.Debug.Printf("Error marshalling output item: %v", err)
			continue
		}

		var item ResponseItem
		if err := json.Unmarshal(itemBytes, &item); err != nil {
			logger.Debug.Printf("Error unmarshalling output item: %v", err)
			continue
		}

		logger.Debug.Printf("Processing output item - Type: %s, ID: %s", item.Type, item.ID)

		switch item.Type {
		case "web_search_call":
			if webSearchId == "" {
				webSearchId = item.ID
			}
			logger.Debug.Printf("Web search call ID: %s, Status: %s", item.ID, item.Status)

		case "message":
			if item.Role == "assistant" {
				messageId = item.ID
				logger.Debug.Printf("Found assistant message with %d content items", len(item.Content))

				for _, content := range item.Content {
					if content.Type == "output_text" {
						responseText = content.Text
						logger.Debug.Printf("Extracted response text: %d chars", len(responseText))

						if len(content.Annotations) > 0 {
							allAnnotations = append(allAnnotations, content.Annotations...)
							logger.Debug.Printf("Found %d annotations", len(content.Annotations))
						}
					}
				}
			}
		}
	}

	if responseText == "" {
		logger.Debug.Println("Warning: No response text found in web search response")
		responseText = "No response text available from web search"
	}

	// Create a structured response content for storage
	// This is key to avoiding the history issues seen in Claude
	storedContent := WebSearchResponseContent{
		Type:        "web_search_response",
		OutputText:  responseText,
		WebSearchId: webSearchId,
		MessageId:   messageId,
		Annotations: allAnnotations,
	}

	contentJson, err := json.Marshal(storedContent)
	if err != nil {
		logger.Debug.Printf("Error marshalling web search response content: %s", err)
	}

	logger.Debug.Printf("Storing web search response with %d annotations", len(allAnnotations))

	// Display the response text first
	model.ResponseHandler.RecievedText(responseText, nil)

	// Display citations if present
	if len(allAnnotations) > 0 {
		citationText := "\n\nSources:\n"
		for i, ann := range allAnnotations {
			citationText += fmt.Sprintf("[%d] %s - %s\n", i+1, ann.Title, ann.URL)
		}
		model.ResponseHandler.RecievedText(citationText, nil)
	}

	// Save to history - using empty tool results since web search is handled by the API
	model.ResponseHandler.FinalText(
		model.ContextId,
		model.Prompt,
		responseText,
		string(contentJson),
		"", // No tool results - web search is transparent to us
		model.ModelName,
	)
}

// CreateWebSearchPayload builds a payload for the /v1/responses endpoint with web search
func CreateWebSearchPayload(prompt string, history []data.History, model string, context *data.Context) ResponseRequest {
	logger.Debug.Printf("\nMODEL USE: creating web search payload for: %s", model)

	// Build input messages array
	messages := []InputMessage{}

	// Add history as context (limit to recent history to avoid token limits)
	startIdx := len(history) - 5 // Last 5 exchanges
	if startIdx < 0 {
		startIdx = 0
	}
	for i := startIdx; i < len(history); i++ {
		h := history[i]
		messages = append(messages, InputMessage{Role: "user", Content: h.Prompt})

		// For web search responses, use the output text
		responseText := h.Response
		if h.ResponseContent != "" {
			var webContent WebSearchResponseContent
			if err := json.Unmarshal([]byte(h.ResponseContent), &webContent); err == nil && webContent.Type == "web_search_response" {
				responseText = webContent.OutputText
			}
		}
		messages = append(messages, InputMessage{Role: "assistant", Content: responseText})
	}

	// Add current prompt
	messages = append(messages, InputMessage{Role: "user", Content: prompt})

	// Note: Stream is set to false for web search as it may not be supported
	return ResponseRequest{Model: model, Input: messages, Tools: []interface{}{WebSearchTool{Type: "web_search"}}, Stream: false}
}
