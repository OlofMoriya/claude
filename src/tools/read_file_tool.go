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
	var errors []string
	successCount := 0
	const maxBytesPerFile = 100 * 1024 // 100KB per file
	
	for _, file := range files {
		file = strings.TrimSpace(file)
		if file == "" {
			continue
		}
		
		content, err := os.ReadFile(file)
		if err != nil {
			errorMsg := fmt.Sprintf("  - %s: %s", file, err.Error())
			errors = append(errors, errorMsg)
			logger.Debug.Printf("Error reading file %s: %v", file, err)
			continue
		}

		// Apply line filtering if requested
		fileContent := string(content)
		if startLine > 0 || endLine > 0 {
			lines := strings.Split(fileContent, "\n")
			startIdx := startLine - 1 // Convert to 0-indexed
			if startIdx < 0 {
				startIdx = 0
			}
			
			// Check if the requested range is valid
			if startIdx >= len(lines) {
				errorMsg := fmt.Sprintf("  - %s: StartLine %d exceeds file length (%d lines)", file, startLine, len(lines))
				errors = append(errors, errorMsg)
				logger.Debug.Printf("Line range error for %s: start exceeds length", file)
				continue
			}
			
			endIdx := endLine
			if endIdx < 0 || endIdx > len(lines) {
				endIdx = len(lines)
			}
			
			fileContent = strings.Join(lines[startIdx:endIdx], "\n")
			
			// Apply max length limit per file
			if len(fileContent) > maxBytesPerFile {
				fileContent = fileContent[:maxBytesPerFile] + fmt.Sprintf("\n... [truncated, showing first %d KB of range]", maxBytesPerFile/1024)
			}
		} else {
			// Apply max length limit per file when no line range specified
			if len(fileContent) > maxBytesPerFile {
				fileContent = fileContent[:maxBytesPerFile] + fmt.Sprintf("\n... [truncated, showing first %d KB]", maxBytesPerFile/1024)
			}
		}

		// Add file separator with clean header
		if complete_output != "" {
			complete_output += "\n\n"
		}
		complete_output += fmt.Sprintf("=== %s ===\n%s", file, fileContent)
		successCount++

		logger.Debug.Printf("Successfully read file: %s", file)
	}

	// Prepend summary if there were any errors
	if len(errors) > 0 {
		summary := fmt.Sprintf("Read %d/%d files successfully.\n\nFailed files:\n%s\n\n", successCount, len(files), strings.Join(errors, "\n"))
		complete_output = summary + complete_output
	}
	
	// Only return error if ALL files failed
	if successCount == 0 {
		if len(errors) > 0 {
			return "", fmt.Errorf("Failed to read any files:\n%s", strings.Join(errors, "\n"))
		}
		return "", fmt.Errorf("No valid files to read")
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

func (tool *ReadFileTool) FormatToolUse(toolUse data.ToolUse) []string {
	input := ParseToolUseInput(toolUse)
	status := "✓"
	if !toolUse.Result.Success {
		status = "✗"
	}

	lines := []string{fmt.Sprintf("read_file %s", status)}
	if fileNames := strings.TrimSpace(input["FileNames"]); fileNames != "" {
		lines = append(lines, fmt.Sprintf("files: %s", singleLine(fileNames, 100)))
	}

	start := strings.TrimSpace(input["StartLine"])
	end := strings.TrimSpace(input["EndLine"])
	if start != "" || end != "" {
		if start == "" {
			start = "1"
		}
		if end == "" {
			end = "end"
		}
		lines = append(lines, fmt.Sprintf("lines: %s-%s", start, end))
	}

	return lines
}

func init() {
	Register(&ReadFileTool{})
}
