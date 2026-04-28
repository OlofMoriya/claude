package testhelpers

import (
	"fmt"
	"sync"

	"owl/data"
	"owl/tools"
)

type DummyToolCall struct {
	Input map[string]string
}

type DummyTool struct {
	Name        string
	Response    string
	callsMu     sync.Mutex
	Calls       []DummyToolCall
	historyRepo *data.HistoryRepository
	context     *data.Context
}

func NewDummyTool(name string) *DummyTool {
	return &DummyTool{Name: name, Response: "DUMMY_OK"}
}

func (t *DummyTool) Register() {
	tools.Register(t)
}

func (t *DummyTool) GetDefinition() (tools.Tool, string) {
	return tools.Tool{
		Name:         t.GetName(),
		Description:  "dummy tool for tests",
		Groups:       []tools.ToolGroup{tools.ToolGroup("test")},
		Dependencies: []tools.ToolDependency{tools.ToolDependencyLocalExec},
		InputSchema: tools.InputSchema{
			Type: "object",
			Properties: map[string]tools.Property{
				"value": {Type: "string", Description: "test value"},
			},
		},
	}, tools.LOCAL
}

func (t *DummyTool) Run(input map[string]string) (string, error) {
	if input == nil {
		input = map[string]string{}
	}
	t.callsMu.Lock()
	defer t.callsMu.Unlock()
	copyInput := make(map[string]string, len(input))
	for k, v := range input {
		copyInput[k] = v
	}
	t.Calls = append(t.Calls, DummyToolCall{Input: copyInput})
	return fmt.Sprintf("%s:%s", t.Response, copyInput["value"]), nil
}

func (t *DummyTool) GetName() string {
	if t.Name != "" {
		return t.Name
	}
	return "dummy_tool_test"
}

func (t *DummyTool) SetHistory(repo *data.HistoryRepository, context *data.Context) {
	t.historyRepo = repo
	t.context = context
}

func (t *DummyTool) GetGroups() []tools.ToolGroup {
	return []tools.ToolGroup{tools.ToolGroup("test")}
}

func (t *DummyTool) ResetCalls() {
	t.callsMu.Lock()
	defer t.callsMu.Unlock()
	t.Calls = nil
}

func (t *DummyTool) HistoryRepo() *data.HistoryRepository {
	return t.historyRepo
}

func (t *DummyTool) Context() *data.Context {
	return t.context
}
