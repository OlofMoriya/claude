package tools

import (
	"fmt"
	"os/exec"
	"owl/logger"
	"strings"

	"github.com/fatih/color"
)

type ReadFileTool struct {
}

type ReadFileInput struct {
	FileNames string
}

func (tool *ReadFileTool) Run(i map[string]string) (string, error) {

	input, ok := i["FileNames"]

	if !ok {
		return "", fmt.Errorf("Could not parse FileWriteInput from input")
	}

	logger.Screen(fmt.Sprintf("Asked to read file %v", input), color.RGB(150, 150, 150))

	files := strings.Split(input, ";")

	out, err := exec.Command("/bin/cat", files...).Output()
	if err != nil {
		fmt.Printf("Failed to read files, %s", err)
	}
	value := string(out)
	return value, err
}

func (tool *ReadFileTool) GetName() string {
	return "file_list"
}

func (tool *ReadFileTool) GetDefinition() Tool {
	return Tool{
		Name:        tool.GetName(),
		Description: "Fetches the list of files under the current directory recursively. This enable the model to see the current project to analyze which files are present in a code prodject or similar. In combination with the read_file tool this should enable the model to gather what information is needed to assist with a project. Especially important in codeing assignments. ",

		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]Property{
				"Filter": {
					Type:        "string",
					Description: "Just a placeholder for now. Send in a greeting for now.",
				},
			},
		},
	}
}

func init() {
	Register(&ReadFileTool{})
}
