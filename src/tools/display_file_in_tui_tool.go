package tools

import (
	"fmt"
	"os"
	"owl/data"
	"owl/interaction"
	"strconv"
	"strings"
	"time"
)

type DisplayFileInTUITool struct{}

func (tool *DisplayFileInTUITool) SetHistory(repo *data.HistoryRepository, context *data.Context) {}

func (tool *DisplayFileInTUITool) GetName() string {
	return "display_file_in_tui"
}

func (tool *DisplayFileInTUITool) GetGroups() []ToolGroup {
	return []ToolGroup{ToolGroupPlanner, ToolGroupDeveloper}
}

func (tool *DisplayFileInTUITool) GetDefinition() (Tool, string) {
	return Tool{
		Name:         tool.GetName(),
		Description:  "Displays a file in TUI for the user. Returns only an acknowledgement, never file content.",
		Groups:       []ToolGroup{ToolGroupPlanner, ToolGroupDeveloper},
		Dependencies: []ToolDependency{ToolDependencyLocalExec},
		InputSchema: InputSchema{
			Type:     "object",
			Required: []string{"path"},
			Properties: map[string]Property{
				"path": {
					Type:        "string",
					Description: "File path to display in TUI.",
				},
				"start_line": {
					Type:        "integer",
					Description: "Optional 1-indexed start line.",
				},
				"end_line": {
					Type:        "integer",
					Description: "Optional 1-indexed end line (inclusive).",
				},
			},
		},
	}, LOCAL
}

func (tool *DisplayFileInTUITool) Run(i map[string]string) (string, error) {
	if interaction.FileDisplayPromptChan == nil {
		return "", fmt.Errorf("display_file_in_tui requires TUI interactive mode")
	}

	path := strings.TrimSpace(getInputValue(i, "path", "Path", "file", "File"))
	if path == "" {
		return "", fmt.Errorf("path is required")
	}

	startLine, endLine, err := parseLineRange(i)
	if err != nil {
		return "", err
	}

	bytes, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	content := string(bytes)
	if startLine > 0 || endLine > 0 {
		content, err = sliceByLines(content, startLine, endLine)
		if err != nil {
			return "", err
		}
	}

	resultChan := make(chan interaction.FileDisplayResult, 1)
	prompt := interaction.FileDisplayPrompt{
		Path:         path,
		Title:        "File Preview",
		Content:      content,
		StartLine:    startLine,
		EndLine:      endLine,
		ResponseChan: resultChan,
	}

	select {
	case interaction.FileDisplayPromptChan <- prompt:
	case <-time.After(3 * time.Second):
		return "", fmt.Errorf("display_file_in_tui failed to reach TUI prompt handler")
	}

	select {
	case result := <-resultChan:
		if result.Err != nil {
			return "", result.Err
		}
		return fmt.Sprintf("Displayed file in TUI: %s", path), nil
	case <-time.After(10 * time.Minute):
		return "", fmt.Errorf("display_file_in_tui timed out waiting for user")
	}
}

func parseLineRange(input map[string]string) (int, int, error) {
	startLine := 0
	endLine := 0

	if raw := strings.TrimSpace(getInputValue(input, "start_line", "StartLine")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 {
			return 0, 0, fmt.Errorf("start_line must be a positive integer")
		}
		startLine = parsed
	}

	if raw := strings.TrimSpace(getInputValue(input, "end_line", "EndLine")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 {
			return 0, 0, fmt.Errorf("end_line must be a positive integer")
		}
		endLine = parsed
	}

	if startLine > 0 && endLine > 0 && startLine > endLine {
		return 0, 0, fmt.Errorf("start_line cannot be greater than end_line")
	}

	return startLine, endLine, nil
}

func sliceByLines(content string, startLine, endLine int) (string, error) {
	lines := strings.Split(content, "\n")
	if len(lines) == 0 {
		return "", nil
	}

	start := 1
	if startLine > 0 {
		start = startLine
	}
	if start > len(lines) {
		return "", fmt.Errorf("start_line exceeds file length (%d lines)", len(lines))
	}

	end := len(lines)
	if endLine > 0 && endLine < end {
		end = endLine
	}

	if end < start {
		return "", fmt.Errorf("invalid line range")
	}

	return strings.Join(lines[start-1:end], "\n"), nil
}

func init() {
	Register(&DisplayFileInTUITool{})
}
