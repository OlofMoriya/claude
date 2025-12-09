package tools

import (
	"fmt"
	"owl/data"
	"owl/models"
	"sync"
)

type ToolModel interface {
	GetDefinition() Tool
	Run(input map[string]string) (string, error)
	GetName() string
}

type ToolRunner struct {
	ResponseHandler   *models.ResponseHandler
	HistoryRepository *data.HistoryRepository
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
	if err != nil {
		return "", err
	}

	return tool.Run(rawInput)
}

func GetCustomTools() []Tool {
	tools := []Tool{}
	for _, tool := range defaultRegistry.tools {
		tools = append(tools, tool.GetDefinition())
	}
	return tools
}
