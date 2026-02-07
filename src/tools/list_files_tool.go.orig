package tools

import (
	"os/exec"
	"owl/data"
	"owl/logger"

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
		Name:        tool.GetName(),
		Description: "Lists all files in and under this directory. Can be used to understand the project structure.",

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

	logger.Screen("\nAsked to list files", color.RGB(150, 150, 150))

	out, err := exec.Command("/bin/ls", "-R").Output()
	if err != nil {
		logger.Debug.Printf("error while fetching history for context", err)
	}

	value := string(out)
	return value, nil
}

func init() {
	Register(&ListFilesTool{})
}
