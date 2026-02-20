package tools

import (
	"fmt"
	"owl/data"
	"owl/logger"
	"owl/models"
	"sync"
)

var LOCAL string = "LOCAL"
var REMOTE string = "REMOTE"

type ToolModel interface {
	GetDefinition() (Tool, string)
	Run(input map[string]string) (string, error)
	GetName() string
	SetHistory(*data.HistoryRepository, *data.Context)
}

type ToolRunner struct {
	ResponseHandler   *models.ResponseHandler
	HistoryRepository *data.HistoryRepository
	Context           *data.Context
}

type ToolRegistry struct {
	mu    sync.RWMutex
	tools map[string]ToolModel
}

var defaultRegistry = &ToolRegistry{
	tools: make(map[string]ToolModel),
}

func Register(tool ToolModel) {
	defaultRegistry.mu.Lock()
	defer defaultRegistry.mu.Unlock()
	defaultRegistry.tools[tool.GetName()] = tool
}

func GetTool(name string) (ToolModel, error) {
	defaultRegistry.mu.RLock()
	defer defaultRegistry.mu.RUnlock()

	tool, exists := defaultRegistry.tools[name]

	if !exists {
		return nil, fmt.Errorf("tool not found: %s", name)
	}

	return tool, nil
}

func (runner *ToolRunner) ExecuteTool(ctx data.Context, name string, rawInput map[string]string) (string, error) {
	tool, err := GetTool(name)
	tool.SetHistory(runner.HistoryRepository, runner.Context)

	if err != nil {
		return "", err
	}

	return tool.Run(rawInput)
}

func GetCustomTools(mode string) []Tool {
	tools := []Tool{}
	for _, tool := range defaultRegistry.tools {
		definition, tool_mode := tool.GetDefinition()
		logger.Debug.Printf("\ntool: %s, Mode is %s, tool mode is %s", tool.GetName(), mode, tool_mode)
		if mode != REMOTE || tool_mode == REMOTE {
			tools = append(tools, definition)
		}
	}
	return tools
}
