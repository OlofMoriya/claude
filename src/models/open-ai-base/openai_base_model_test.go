package openai_base

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"
	"testing"

	commontypes "owl/common_types"
	"owl/data"
	"owl/logger"
	"owl/services"
	testhelpers "owl/test_helpers"
)

func TestOpenAIModelToolCallbacksTriggerFinalText(t *testing.T) {
	ensureTestLogger()
	dummyTool := testhelpers.NewDummyTool("dummy_tool_openai_test")
	dummyTool.Register()
	dummyTool.ResetCalls()

	repo := testhelpers.NewMockHistoryRepository()
	ctx := data.Context{Id: 2, Name: "ctx"}
	repo.Contexts[ctx.Id] = ctx

	handler := testhelpers.NewMockResponseHandler()

	awaitedCalls := 0
	services.SetAwaitedQueryHook(func(prompt string, model commontypes.Model, historyRepository data.HistoryRepository, historyCount int, context *data.Context, modifiers *commontypes.PayloadModifiers, modelName string) {
		awaitedCalls++
	})
	defer services.SetAwaitedQueryHook(nil)

	model := newTestOpenAIModel(handler, repo, &ctx)
	model.Prompt = "inspect"
	model.ModelName = "gpt"

	chat := ChatCompletion{
		Usage: Usage{PromptTokens: 12, CompletionTokens: 34},
		Choices: []Choice{
			{
				Message: Message{
					Role:    "assistant",
					Content: "partial",
					ToolCalls: []ToolCall{
						{Id: "tool-1", Type: "function", Function: FunctionCall{Name: dummyTool.GetName(), Arguments: `{"value":"ping"}`}},
					},
				},
			},
		},
	}

	body, err := json.Marshal(chat)
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
	if finalEvents[0].Usage == nil || finalEvents[0].Usage.PromptTokens != 12 || finalEvents[0].Usage.CompletionTokens != 34 {
		t.Fatalf("expected token usage to be recorded, got %+v", finalEvents[0].Usage)
	}
	if awaitedCalls != 1 {
		t.Fatalf("expected awaited query hook to run once, got %d", awaitedCalls)
	}
}

func TestOpenAIModelStreamingToolCallbacks(t *testing.T) {
	ensureTestLogger()
	dummyTool := testhelpers.NewDummyTool("dummy_tool_openai_stream")
	dummyTool.Register()
	dummyTool.ResetCalls()

	repo := testhelpers.NewMockHistoryRepository()
	ctx := data.Context{Id: 3, Name: "stream_ctx"}
	repo.Contexts[ctx.Id] = ctx

	handler := testhelpers.NewMockResponseHandler()
	awaitedCalls := 0
	services.SetAwaitedQueryHook(func(prompt string, model commontypes.Model, historyRepository data.HistoryRepository, historyCount int, context *data.Context, modifiers *commontypes.PayloadModifiers, modelName string) {
		awaitedCalls++
	})
	defer services.SetAwaitedQueryHook(nil)

	model := newTestOpenAIModel(handler, repo, &ctx)
	model.Prompt = "inspect"
	model.ModelName = "gpt"
	model.StreamedToolCalls = make(map[int]*StreamingToolCall)

	chunkText := ChatCompletionChunk{
		Choices: []ChatCompletionChunkChoice{{
			Delta: Delta{Content: "Hello "},
		}},
	}
	streamOpenAIChunk(t, model, chunkText)

	chunkTool := ChatCompletionChunk{
		Choices: []ChatCompletionChunkChoice{{
			Delta: Delta{ToolCalls: []ToolCall{{
				Index: 0,
				Id:    "tool-1",
				Type:  "function",
				Function: FunctionCall{
					Name:      dummyTool.GetName(),
					Arguments: `{"value":"ping"}`,
				},
			}}},
		}},
	}
	streamOpenAIChunk(t, model, chunkTool)

	chunkUsage := ChatCompletionChunk{
		Usage: Usage{PromptTokens: 30, CompletionTokens: 60},
	}
	streamOpenAIChunk(t, model, chunkUsage)
	model.HandleStreamedLine([]byte("data: [DONE]\n"))

	textEvents := handler.CopyTextEvents()
	if len(textEvents) == 0 {
		t.Fatalf("expected streamed text events")
	}
	if !strings.Contains(textEvents[0].Text, "Hello") {
		t.Fatalf("expected streamed text content, got %s", textEvents[0].Text)
	}
	if len(dummyTool.Calls) != 1 {
		t.Fatalf("expected dummy tool to run once, got %d", len(dummyTool.Calls))
	}
	if awaitedCalls != 1 {
		t.Fatalf("expected awaited query hook once, got %d", awaitedCalls)
	}
	finalEvents := handler.CopyFinalEvents()
	if len(finalEvents) != 1 {
		t.Fatalf("expected final event, got %d", len(finalEvents))
	}
	if finalEvents[0].Usage == nil || finalEvents[0].Usage.PromptTokens != 30 || finalEvents[0].Usage.CompletionTokens != 60 {
		t.Fatalf("expected streaming usage to be recorded, got %+v", finalEvents[0].Usage)
	}
}

func ensureTestLogger() {
	if logger.Debug == nil {
		logger.Debug = log.New(io.Discard, "", 0)
	}
}

type testOpenAIModel struct {
	OpenAICompatibleModel
}

func newTestOpenAIModel(handler commontypes.ResponseHandler, repo data.HistoryRepository, ctx *data.Context) *testOpenAIModel {
	m := &testOpenAIModel{}
	m.ResponseHandler = handler
	m.HistoryRepository = repo
	m.Context = ctx
	m.ContextId = ctx.Id
	m.Modifiers = &commontypes.PayloadModifiers{}
	m.StreamedToolCalls = make(map[int]*StreamingToolCall)
	return m
}

func (t *testOpenAIModel) CreateRequest(context *data.Context, prompt string, streaming bool, history []data.History, modifiers *commontypes.PayloadModifiers) *http.Request {
	return nil
}

func (t *testOpenAIModel) HandleStreamedLine(line []byte) {
	t.OpenAICompatibleModel.HandleStreamedLine(line, t)
}

func (t *testOpenAIModel) HandleBodyBytes(bytes []byte) {
	t.OpenAICompatibleModel.HandleBodyBytes(bytes, t)
}

func (t *testOpenAIModel) SetResponseHandler(responseHandler commontypes.ResponseHandler) {
	t.ResponseHandler = responseHandler
}

func streamOpenAIChunk(t *testing.T, model *testOpenAIModel, chunk ChatCompletionChunk) {
	t.Helper()
	bytes, err := json.Marshal(chunk)
	if err != nil {
		t.Fatalf("failed to marshal chunk: %v", err)
	}
	model.HandleStreamedLine([]byte("data: " + string(bytes) + "\n"))
}
