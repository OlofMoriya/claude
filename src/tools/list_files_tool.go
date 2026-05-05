package tools

import (
	"fmt"
	"os/exec"
	"owl/data"
	"owl/logger"
	"strings"
	"unicode/utf8"

	"github.com/fatih/color"
)

type ListFilesTool struct {
}

type FileListInput struct {
	Filter string
}

func (tool *ListFilesTool) SetHistory(repo *data.HistoryRepository, context *data.Context) {
}

func (ool *ListFilesTool) GetName() string {
	return "list_files"
}

func (tool *ListFilesTool) GetDefinition() (Tool, string) {
	return Tool{
		Name:         tool.GetName(),
		Description:  "Lists all files in and under this directory. Can be used to understand the project structure.",
		Groups:       []ToolGroup{ToolGroupPlanner, ToolGroupDeveloper},
		Dependencies: []ToolDependency{ToolDependencyLocalExec},

		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]Property{
				"Filter": {
					Type:        "string",
					Description: "This is just a placeholder for now. The parameter is not used but needs to be in the definition for future use. For now, send in the extensions of interest, seperated by comma, but don't expect it to be honored.",
				},
			},
		},
	}, LOCAL
}

func (tool *ListFilesTool) Run(i map[string]string) (string, error) {
	const (
		maxLines = 500
		maxBytes = 50000 // 50KB
	)

	logger.Screen("\nAsked to list files", color.RGB(150, 150, 150))

	out, err := exec.Command("find", ".", "-not", "(", "-path", "./.git", "-prune", ")", "-not", "(", "-path", "./node_modules", "-prune", ")").Output()
	if err != nil {
		logger.Debug.Printf("error while listing files: %v", err)
		return "", err
	}

	value := string(out)

	// Apply limits to prevent overflow
	lines := strings.Split(value, "\n")
	if len(lines) > maxLines {
		lines = lines[:maxLines]
		value = strings.Join(lines, "\n")
		value += fmt.Sprintf("\n\n... [Output truncated: showing first %d of %d total lines]", maxLines, len(strings.Split(string(out), "\n")))
	}

	// Also check byte size
	if utf8.RuneCountInString(value) > maxBytes {
		runes := []rune(value)
		value = string(runes[:maxBytes]) + fmt.Sprintf("\n\n... [Output truncated: showing first %d bytes]", maxBytes)
	}

	return value, nil
}

func (tool *ListFilesTool) GetGroups() []ToolGroup {
	return []ToolGroup{ToolGroupPlanner, ToolGroupDeveloper}
}

func (tool *ListFilesTool) FormatToolUse(toolUse data.ToolUse) []string {
	input := ParseToolUseInput(toolUse)
	status := "✓"
	if !toolUse.Result.Success {
		status = "✗"
	}

	lines := []string{fmt.Sprintf("list_files %s", status)}
	if filter := strings.TrimSpace(input["Filter"]); filter != "" {
		lines = append(lines, fmt.Sprintf("filter: %s", singleLine(filter, 80)))
	}

	resultLines := strings.Split(strings.TrimSpace(toolUse.Result.Content), "\n")
	nonEmpty := 0
	for _, l := range resultLines {
		if strings.TrimSpace(l) != "" {
			nonEmpty++
		}
	}
	if nonEmpty > 0 {
		lines = append(lines, fmt.Sprintf("items: %d", nonEmpty))
	}

	return lines
}

func init() {
	Register(&ListFilesTool{})
}
