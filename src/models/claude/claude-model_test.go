package claude_model

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strings"
	"testing"

	commontypes "owl/common_types"
	"owl/data"
	"owl/logger"
	"owl/services"
	testhelpers "owl/test_helpers"
)

func TestClaudeModelToolCallbacksTriggerFinalText(t *testing.T) {
	ensureTestLogger()
	dummyTool := testhelpers.NewDummyTool("dummy_tool_claude_test")
	dummyTool.Register()
	dummyTool.ResetCalls()

	repo := testhelpers.NewMockHistoryRepository()
	ctx := data.Context{Id: 1, Name: "ctx"}
	repo.Contexts[ctx.Id] = ctx

	handler := testhelpers.NewMockResponseHandler()

	awaitedCalls := 0
	services.SetAwaitedQueryHook(func(prompt string, model commontypes.Model, historyRepository data.HistoryRepository, historyCount int, context *data.Context, modifiers *commontypes.PayloadModifiers, modelName string) {
		awaitedCalls++
	})
	defer services.SetAwaitedQueryHook(nil)

	model := &ClaudeModel{
		HistoryRepository: repo,
		ResponseHandler:   handler,
		Context:           &ctx,
		ModelVersion:      "sonnet",
		Modifiers:         &commontypes.PayloadModifiers{},
	}
	model.Prompt = "inspect"

	response := MessageResponse{
		Usage: Usage{InputTokens: 15, OutputTokens: 25, CacheReadInputTokens: 5, CacheCreationInputTokens: 7},
		Content: []ResponseMessage{
			{Type: "text", Text: "partial"},
			{Type: "tool_use", Id: "tool-1", Name: dummyTool.GetName(), Input: map[string]interface{}{"value": "ping"}},
		},
	}

	body, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("failed to marshal response: %v", err)
	}

	model.HandleBodyBytes(body)

	if len(dummyTool.Calls) != 1 {
		t.Fatalf("expected dummy tool to run once, got %d", len(dummyTool.Calls))
	}

	finalEvents := handler.CopyFinalEvents()
	if len(finalEvents) != 1 {
		t.Fatalf("expected final event, got %d", len(finalEvents))
	}
	if len(finalEvents[0].ToolUse) == 0 || !strings.Contains(finalEvents[0].ToolUse[0].Result.Content, dummyTool.Response) {
		t.Fatalf("expected tool results to contain dummy response, got %+v", finalEvents[0].ToolUse)
	}
	if finalEvents[0].Usage == nil || finalEvents[0].Usage.PromptTokens != 15 || finalEvents[0].Usage.CompletionTokens != 25 || finalEvents[0].Usage.CacheReadTokens != 5 || finalEvents[0].Usage.CacheWriteTokens != 7 {
		t.Fatalf("expected usage to include claude cache metrics, got %+v", finalEvents[0].Usage)
	}
	if awaitedCalls != 1 {
		t.Fatalf("expected awaited query hook to run once, got %d", awaitedCalls)
	}
}

func TestClaudeModelStreamingToolCallbacks(t *testing.T) {
	ensureTestLogger()
	dummyTool := testhelpers.NewDummyTool("dummy_tool_claude_stream")
	dummyTool.Register()
	dummyTool.ResetCalls()

	repo := testhelpers.NewMockHistoryRepository()
	ctx := data.Context{Id: 2, Name: "stream_ctx"}
	repo.Contexts[ctx.Id] = ctx

	handler := testhelpers.NewMockResponseHandler()
	awaitedCalls := 0
	services.SetAwaitedQueryHook(func(prompt string, model commontypes.Model, historyRepository data.HistoryRepository, historyCount int, context *data.Context, modifiers *commontypes.PayloadModifiers, modelName string) {
		awaitedCalls++
	})
	defer services.SetAwaitedQueryHook(nil)

	model := &ClaudeModel{
		HistoryRepository: repo,
		ResponseHandler:   handler,
		Context:           &ctx,
		ModelVersion:      "sonnet",
		Modifiers:         &commontypes.PayloadModifiers{},
	}
	model.Prompt = "inspect"

	streamClaudeEvent(t, model, "content_block_start", map[string]interface{}{
		"type":          "content_block_start",
		"index":         0,
		"content_block": map[string]interface{}{"type": "text"},
	})
	streamClaudeEvent(t, model, "content_block_delta", map[string]interface{}{
		"type":  "content_block_delta",
		"index": 0,
		"delta": map[string]interface{}{"type": "text_delta", "text": "Hello "},
	})

	streamClaudeEvent(t, model, "content_block_start", map[string]interface{}{
		"type":  "content_block_start",
		"index": 1,
		"content_block": map[string]interface{}{
			"type": "tool_use",
			"id":   "tool-1",
			"name": dummyTool.GetName(),
		},
	})
	streamClaudeEvent(t, model, "content_block_delta", map[string]interface{}{
		"type":  "content_block_delta",
		"index": 1,
		"delta": map[string]interface{}{
			"type":         "input_json_delta",
			"partial_json": `{"value":"ping"}`,
		},
	})
	streamClaudeEvent(t, model, "message_delta", map[string]interface{}{
		"type":  "message_delta",
		"delta": map[string]interface{}{"stop_reason": "tool_use"},
		"usage": map[string]interface{}{
			"input_tokens":                20,
			"output_tokens":               40,
			"cache_read_input_tokens":     5,
			"cache_creation_input_tokens": 7,
		},
	})
	streamClaudeEvent(t, model, "message_stop", map[string]interface{}{"type": "message_stop"})

	textEvents := handler.CopyTextEvents()
	if len(textEvents) == 0 {
		t.Fatalf("expected streaming text events")
	}
	if !strings.Contains(textEvents[0].Text, "Hello") {
		t.Fatalf("expected text event to include streamed content, got %s", textEvents[0].Text)
	}

	if len(dummyTool.Calls) != 1 {
		t.Fatalf("expected dummy tool to run once, got %d", len(dummyTool.Calls))
	}
	if awaitedCalls != 1 {
		t.Fatalf("expected awaited query hook to run once, got %d", awaitedCalls)
	}
	finalEvents := handler.CopyFinalEvents()
	if len(finalEvents) != 1 {
		t.Fatalf("expected final event, got %d", len(finalEvents))
	}
	usage := finalEvents[0].Usage
	if usage == nil || usage.PromptTokens != 20 || usage.CompletionTokens != 40 || usage.CacheReadTokens != 5 || usage.CacheWriteTokens != 7 {
		t.Fatalf("expected streaming usage metrics, got %+v", usage)
	}
}

func TestClaudeModelHandleBodyBytes_RealToolUsePayload(t *testing.T) {
	ensureTestLogger()

	repo := testhelpers.NewMockHistoryRepository()
	ctx := data.Context{Id: 10, Name: "ctx-real-payload"}
	repo.Contexts[ctx.Id] = ctx

	handler := testhelpers.NewMockResponseHandler()
	awaitedCalls := 0
	services.SetAwaitedQueryHook(func(prompt string, model commontypes.Model, historyRepository data.HistoryRepository, historyCount int, context *data.Context, modifiers *commontypes.PayloadModifiers, modelName string) {
		awaitedCalls++
	})
	defer services.SetAwaitedQueryHook(nil)

	model := &ClaudeModel{
		HistoryRepository: repo,
		ResponseHandler:   handler,
		Context:           &ctx,
		ModelVersion:      "sonnet",
		Modifiers:         &commontypes.PayloadModifiers{},
	}
	model.Prompt = "can you use a helper tool to inspect customer records"

	body := []byte(`{
		"model": "claude-sonnet-4-5-20250929",
		"id": "msg_01PSiBcPCAq3CmE4KBUEEbpe",
		"type": "message",
		"role": "assistant",
		"content": [
			{
				"type": "text",
				"text": "Let me try with a different approach - checking if there's a tracking number mode or trying to query directly:"
			},
			{
				"type": "tool_use",
				"id": "toolu_01S1U5tyAjjvj6cdwCkb7grt",
				"name": "nonexistent_test_tool",
				"input": {
					"Port": "2005",
					"Database": "customer",
					"Query": "SELECT table_name FROM information_schema.tables WHERE table_schema = 'customer'"
				},
				"caller": {
					"type": "direct"
				}
			}
		],
		"stop_reason": "tool_use",
		"usage": {
			"input_tokens": 6,
			"output_tokens": 132,
			"cache_read_input_tokens": 0,
			"cache_creation_input_tokens": 3778
		}
	}`)

	model.HandleBodyBytes(body)

	finalEvents := handler.CopyFinalEvents()
	if len(finalEvents) != 1 {
		t.Fatalf("expected one final event, got %d", len(finalEvents))
	}

	if len(finalEvents[0].ToolUse) != 1 {
		t.Fatalf("expected one tool use entry, got %d", len(finalEvents[0].ToolUse))
	}

	toolUse := finalEvents[0].ToolUse[0]
	if toolUse.Id != "toolu_01S1U5tyAjjvj6cdwCkb7grt" {
		t.Fatalf("expected tool id to match fixture, got %s", toolUse.Id)
	}
	if toolUse.Name != "nonexistent_test_tool" {
		t.Fatalf("expected tool name to match fixture, got %s", toolUse.Name)
	}
	if toolUse.CallerType != "assistant" {
		t.Fatalf("expected local caller type assistant, got %s", toolUse.CallerType)
	}
	if !strings.Contains(toolUse.Input, "table_schema") {
		t.Fatalf("expected serialized tool input to include SQL query, got %s", toolUse.Input)
	}
	if toolUse.Result.Success {
		t.Fatalf("expected tool execution to fail for unknown test tool")
	}

	if awaitedCalls != 1 {
		t.Fatalf("expected awaited query hook to run once for local tool_use, got %d", awaitedCalls)
	}
}

func TestClaudePayloadReplaysToolUseFromHistoryRecord(t *testing.T) {
	history := []data.History{
		{
			Prompt:   "can you use a helper tool to inspect customer records",
			Response: "Let me try with a different approach",
			ToolUse: []data.ToolUse{
				{
					Id:         "toolu_01Eaj3Jxt9dVG8D3GU81otKR",
					Name:       "nonexistent_test_tool",
					CallerType: "assistant",
					Input:      `{"Database":"customer","Port":"2005","Query":"SELECT id, name FROM customers WHERE name LIKE 'Apotea%'"}`,
					Result: data.ToolResult{
						ToolUseId: "toolu_01Eaj3Jxt9dVG8D3GU81otKR",
						Content:   `[{"id":"clfsfe5ab000ks60nbvbghwoo","name":"Apotea AB"}]`,
						Success:   true,
					},
				},
			},
		},
	}

	payload := createClaudePayload("latest", false, history, "claude-sonnet", false, &data.Context{Id: 11}, &commontypes.PayloadModifiers{})
	messageSlice, ok := payload.Messages.([]Message)
	if !ok {
		t.Fatalf("expected payload.Messages to be []Message")
	}

	foundAssistantToolUse := false
	foundUserToolResult := false
	for _, raw := range messageSlice {
		msg, ok := raw.(RequestMessage)
		if !ok {
			continue
		}
		for _, c := range msg.Content {
			switch v := c.(type) {
			case ToolUseContent:
				if msg.Role == "assistant" && v.Id == "toolu_01Eaj3Jxt9dVG8D3GU81otKR" && v.Name == "nonexistent_test_tool" {
					foundAssistantToolUse = true
				}
			case ToolResponseContent:
				if msg.Role == "user" && v.Id == "toolu_01Eaj3Jxt9dVG8D3GU81otKR" && strings.Contains(v.Content, "Apotea AB") {
					foundUserToolResult = true
				}
			}
		}
	}

	if !foundAssistantToolUse {
		t.Fatalf("expected assistant tool_use block replay from history")
	}
	if !foundUserToolResult {
		t.Fatalf("expected user tool_result block replay from history")
	}
}

func TestClaudePayloadCachingRules(t *testing.T) {
	history := []data.History{
		{Prompt: "First question", Response: "answer"},
		{Prompt: "Second question", Response: "answer"},
		buildToolHistory("Third question", []string{"tool-a"}),
		buildToolHistory("Fourth question", []string{"tool-b1", "tool-b2"}),
	}
	context := &data.Context{Id: 1}
	payload := createClaudePayload("latest", false, history, "claude-sonnet", false, context, &commontypes.PayloadModifiers{})
	messageSlice, ok := payload.Messages.([]Message)
	if !ok {
		t.Fatalf("expected payload.Messages to be []Message")
	}
	cachedUsers := []string{}
	cachedTools := []string{}
	for _, raw := range messageSlice {
		msg, ok := raw.(RequestMessage)
		if !ok {
			continue
		}
		for _, content := range msg.Content {
			switch v := content.(type) {
			case TextContent:
				if v.CacheControl != nil {
					cachedUsers = append(cachedUsers, v.Text)
				}
			case ToolResponseContent:
				if v.CacheControl != nil {
					cachedTools = append(cachedTools, v.Id)
				}
			}
		}
	}
	if len(cachedUsers) != 1 || cachedUsers[0] != "Second question" {
		t.Fatalf("expected only second question cached, got %v", cachedUsers)
	}
	if len(cachedTools) != 2 {
		t.Fatalf("expected two cached tool responses, got %v", cachedTools)
	}
	if !(contains(cachedTools, "tool-b2") && contains(cachedTools, "tool-a")) {
		t.Fatalf("unexpected cached tool ids: %v", cachedTools)
	}
}

func buildToolHistory(prompt string, toolIDs []string) data.History {
	toolUses := make([]data.ToolUse, len(toolIDs))
	for i, id := range toolIDs {
		toolUses[i] = data.ToolUse{
			Id:         id,
			Name:       fmt.Sprintf("tool-%d", i),
			Input:      `{}`,
			CallerType: "assistant",
			Result: data.ToolResult{
				ToolUseId: id,
				Content:   fmt.Sprintf("result-%s", id),
				Success:   true,
			},
		}
	}
	return data.History{
		Prompt:   prompt,
		Response: "answer",
		ToolUse:  toolUses,
	}
}

func contains(list []string, target string) bool {
	for _, item := range list {
		if item == target {
			return true
		}
	}
	return false
}

func ensureTestLogger() {
	if logger.Debug == nil {
		logger.Debug = log.New(io.Discard, "", 0)
	}
}

func streamClaudeEvent(t *testing.T, model *ClaudeModel, event string, payload map[string]interface{}) {
	t.Helper()
	model.HandleStreamedLine([]byte(fmt.Sprintf("event: %s\n", event)))
	bytes, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}
	model.HandleStreamedLine([]byte(fmt.Sprintf("data: %s\n", bytes)))
}
