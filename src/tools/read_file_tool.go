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
	logger.Screen(fmt.Sprintf("\nAsked to read file %v", input), color.RGB(150, 150, 150))

	if !ok {
		return "", fmt.Errorf("Could not parse FileWriteInput from input")
	}

	files := strings.Split(input, ";")

	out, err := exec.Command("/bin/cat", files...).Output()
	if err != nil {
		fmt.Printf("Failed to read files, %s", err)
	}
	value := string(out)
	return value, err
}

func (tool *ReadFileTool) GetName() string {
	return "read_file"
}

func (tool *ReadFileTool) GetDefinition() Tool {
	return Tool{
		Name:        tool.GetName(),
		Description: "Fetches the contents of the files specified by name and dynamic path. Path starts from where script is being executed. Prefere reading files with .go, .md, .tsx, .ts, .csv, .js, .txt, .mod, .cs, .csproj, .gitignore, .tsx, .jsx, .json extentions. Don't overuse this tool as it increase token use a lot",

		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]Property{
				"FileNames": {
					Type:        "string",
					Description: "filename with dynamic path",
				},
			},
		},
	}
}

func init() {
	Register(&ReadFileTool{})
}
