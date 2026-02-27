package tools

import (
	"fmt"
	commontypes "owl/common_types"
	"owl/data"
	"slices"
	"sync"
)

var LOCAL string = "LOCAL"
var REMOTE string = "REMOTE"

type ToolModel interface {
	GetDefinition() (Tool, string)
	Run(input map[string]string) (string, error)
	GetName() string
	SetHistory(*data.HistoryRepository, *data.Context)
	GetGroups() []string
}

type ToolRunner struct {
	ResponseHandler   *commontypes.ResponseHandler
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
	if err != nil {
		return "", err
	}
	tool.SetHistory(runner.HistoryRepository, runner.Context)

	return tool.Run(rawInput)
}

func GetCustomTools(mode string, filterGroups ...string) []Tool {
	tools := []Tool{}
	for _, tool := range defaultRegistry.tools {
		definition, tool_mode := tool.GetDefinition()

		// 1. Mode check
		if mode == REMOTE && tool_mode != REMOTE {
			continue
		}

		// 2. Group check (if filters are provided)
		if len(filterGroups) > 0 {
			toolGroups := tool.GetGroups()
			match := false
			for _, fg := range filterGroups {
				if slices.Contains(toolGroups, fg) {
					match = true
				}
				if match {
					break
				}
			}
			if !match {
				continue
			}
		}

		tools = append(tools, definition)
	}
	return tools
}
