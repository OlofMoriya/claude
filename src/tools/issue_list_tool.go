package tools

import (
	"fmt"
	"os/exec"
	"owl/data"
	"owl/logger"

	"github.com/fatih/color"
)

type IssueListTool struct {
}

type IssueListLookupInput struct {
	Span string
}

func (tool *IssueListTool) SetHistory(repo *data.HistoryRepository, context *data.Context) {
}

func (tool *IssueListTool) GetDefinition() Tool {

	return Tool{
		Name:        tool.GetName(),
		Description: "Fetches a list of completed issue from my companies issue tracker. It will return itemes from last 7 days that has been marked as Done or Released. This list can be useful for putting together a demo or reporting status on weekly meetings.",

		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]Property{
				"Span": {
					Type:        "string",
					Description: "The duration of time that should be used to look up the issues. Finite list of values [Day, Week, Month]",
				},
			},
		},
	}
}

func (tool *IssueListTool) GetName() string {
	return "issue_list"
}

func (toolRunner *IssueListTool) Run(i map[string]string) (string, error) {

	logger.Screen(fmt.Sprintf("fetching the completed issue list"), color.RGB(150, 150, 150))

	out, err := exec.Command("item-list.sh").Output()
	if err != nil {
		fmt.Printf("Failed to fetch data, %s", err)
	}
	value := string(out)
	return value, nil
}

func init() {
	Register(&IssueListTool{})
}
