package tools

import (
	"fmt"
	"owl/data"
	"owl/logger"
	"strings"

	"github.com/fatih/color"
	"os"
)

type ReadFileTool struct {
}

type ReadFileInput struct {
	FileNames string
}

func (tool *ReadFileTool) SetHistory(repo *data.HistoryRepository, context *data.Context) {
}

func (tool *ReadFileTool) Run(i map[string]string) (string, error) {

	input, ok := i["FileNames"]

	logger.Screen(fmt.Sprintf("\nAsked to read file %v", input), color.RGB(150, 150, 150))

	logger.Debug.Printf("\nAsked to read file %v", input)

	if !ok {
		return "", fmt.Errorf("Could not parse FileWriteInput from input")
	}

	files := strings.Split(input, ";")
	complete_output := ""
	complete_error := ""
	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			complete_error += err.Error()
			logger.Debug.Printf("Error reading file: %v", err)
		}

		complete_output += fmt.Sprintf("\n%s\n%s", file, content)
		logger.Debug.Printf("\ncomplete_output: %v", complete_output)
	}

	if complete_error != "" {
		return complete_output, fmt.Errorf("%s", complete_error)
	}
	return complete_output, nil
}

func (tool *ReadFileTool) GetName() string {
	return "read_file"
}

func (tool *ReadFileTool) GetDefinition() (Tool, string) {
	return Tool{
		Name:        tool.GetName(),
		Description: "Fetches the contents of the files specified by name and dynamic path. Path starts from where script is being executed. Prefere reading files with .go, .md, .tsx, .ts, .csv, .js, .txt, .mod, .cs, .csproj, .gitignore, .tsx, .jsx, .json extentions. Don't overuse this tool as it increase token use a lot",

		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]Property{
				"FileNames": {
					Type:        "string",
					Description: "filenames with dynamic path. multiple files can be read by adding them together with ; between. i.e 'README.md;package.json'",
				},
			},
		},
	}, LOCAL
}

func (tool *ReadFileTool) GetGroups() []string {
	return []string{"dev"}
}

func init() {
	Register(&ReadFileTool{})
}
