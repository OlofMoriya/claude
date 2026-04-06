package tools

import (
	"fmt"
	"owl/data"
	"owl/logger"
	"strconv"
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
	start, ok_start := i["StartLine"]
	stop, ok_stop := i["EndLine"]

	logger.Screen(fmt.Sprintf("\nAsked to read file %v", input), color.RGB(150, 150, 150))

	logger.Debug.Printf("\nAsked to read file %v", input)

	if !ok {
		return "", fmt.Errorf("Could not parse FileWriteInput from input")
	}

	// Parse start and stop line numbers (optional)
	var startLine int = 0
	var endLine int = -1 // -1 means read to end

	if ok_start && start != "" {
		parsedStart, err := strconv.Atoi(start)
		if err != nil {
			return "", fmt.Errorf("Invalid StartLine value: %s", start)
		}
		if parsedStart < 1 {
			return "", fmt.Errorf("StartLine must be >= 1")
		}
		startLine = parsedStart
	}

	if ok_stop && stop != "" {
		parsedStop, err := strconv.Atoi(stop)
		if err != nil {
			return "", fmt.Errorf("Invalid EndLine value: %s", stop)
		}
		if parsedStop < 1 {
			return "", fmt.Errorf("EndLine must be >= 1")
		}
		endLine = parsedStop
	}

	// Validate that start <= end if both are specified
	if startLine > 0 && endLine > 0 && startLine > endLine {
		return "", fmt.Errorf("StartLine (%d) cannot be greater than EndLine (%d)", startLine, endLine)
	}

	files := strings.Split(input, ";")
	complete_output := ""
	complete_error := ""
	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			complete_error += err.Error()
			logger.Debug.Printf("Error reading file: %v", err)
			continue
		}

		// Apply line filtering if requested
		fileContent := string(content)
		if startLine > 0 || endLine > 0 {
			lines := strings.Split(fileContent, "\n")
			start := startLine - 1 // Convert to 0-indexed
			if start < 0 {
				start = 0
			}
			end := endLine
			if end < 0 || end > len(lines) {
				end = len(lines)
			}
			fileContent = strings.Join(lines[start:end], "\n")
		}

		complete_output += fmt.Sprintf("\n%s\n%s", file, fileContent)
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
			Type:     "object",
			Required: []string{"FileNames"},
			Properties: map[string]Property{
				"FileNames": {
					Type:        "string",
					Description: "filenames with dynamic path. multiple files can be read by adding them together with ; between. i.e 'README.md;package.json'",
				},
				"StartLine": {
					Type:        "integer",
					Description: "Optional line number to start reading from (1-indexed). If not specified, starts from beginning of file.",
				},
				"EndLine": {
					Type:        "integer",
					Description: "Optional line number to stop reading at (1-indexed, inclusive). If not specified, reads to end of file.",
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
