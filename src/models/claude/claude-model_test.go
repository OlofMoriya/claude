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
		Content: []ResponseMessage{
			{Type: "text", Text: "partial"},
			{Type: "tool_use", Id: "tool-1", Name: dummyTool.GetName(), Input: map[string]string{"value": "ping"}},
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
	if !strings.Contains(finalEvents[0].ToolResults, dummyTool.Response) {
		t.Fatalf("expected tool results to contain dummy response, got %s", finalEvents[0].ToolResults)
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
