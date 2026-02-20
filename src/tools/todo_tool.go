package tools

import (
	"fmt"
	"os/exec"
	"owl/data"
	"owl/logger"
	"strings"

	"github.com/fatih/color"
)

type TodoTool struct {
}

func (tool *TodoTool) SetHistory(repo *data.HistoryRepository, context *data.Context) {
}

func (tool *TodoTool) Run(i map[string]string) (string, error) {
	title, ok := i["Title"]
	if !ok || title == "" {
		return "", fmt.Errorf("Title is required")
	}

	logger.Screen(fmt.Sprintf("Creating todo: %s", title), color.RGB(150, 150, 150))

	// Build command arguments
	args := []string{"add", title}

	// Add description if provided
	description := i["Description"]
	if description != "" {
		args = append(args, "-D", description)
	}

	// Add due date if provided
	dueDate := i["DueDate"]
	if dueDate != "" {
		args = append(args, "-d", dueDate)
	}

	// Execute todo-tui command
	cmd := exec.Command("todo-tui", args...)
	out, err := cmd.CombinedOutput()
	
	if err != nil {
		return "", fmt.Errorf("Failed to create todo: %s\nOutput: %s", err, string(out))
	}

	result := strings.TrimSpace(string(out))
	if result == "" {
		result = fmt.Sprintf("Todo created successfully: '%s'", title)
	}

	return result, nil
}

func (tool *TodoTool) GetName() string {
	return "create_todo"
}

func (tool *TodoTool) GetDefinition() (Tool, string) {
	return Tool{
		Name:        tool.GetName(),
		Description: "Creates a todo item using the todo-tui application. Use this when you identify action items from emails, messages, or other input that the user needs to follow up on. This creates todos for the human user, not for the LLM.",

		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]Property{
				"Title": {
					Type:        "string",
					Description: "The title/summary of the todo item. Required. Should be concise and actionable.",
				},
				"Description": {
					Type:        "string",
					Description: "Optional detailed description of the todo item. Provides additional context or information.",
				},
				"DueDate": {
					Type:        "string",
					Description: "Optional due date specified as number of days from today. Example: '3' for 3 days from now, '7' for one week from now.",
				},
			},
			Required: []string{"Title"},
		},
	}, LOCAL
}

func init() {
	Register(&TodoTool{})
}
