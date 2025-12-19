package tools

import (
	"fmt"
	"os"
	"owl/logger"
	"strings"

	"github.com/fatih/color"
)

type FileWriteInput struct {
	FileName string
	Content  string
}

type FileWriterTool struct {
}

func (tool *FileWriterTool) Run(i map[string]string) (string, error) {
	var results []string
	var errors []string

	FileName, ok := i["FileName"]
	if !ok {
		return "", fmt.Errorf("Could not parse FileWriteInput from input")
	}

	Content, ok := i["Content"]
	if !ok {
		return "", fmt.Errorf("Could not parse FileWriteInput from input")
	}

	if strings.Contains(FileName, "..") {
		errors = append(errors, fmt.Sprintf("Error: Invalid path '%s' - parent directory references not allowed", FileName))
	}
	if strings.HasPrefix(FileName, "/") {
		errors = append(errors, fmt.Sprintf("Error: Invalid path '%s' - root not allowed", FileName))
	}
	if strings.HasPrefix(FileName, "~") {
		errors = append(errors, fmt.Sprintf("Error: Invalid path '%s' - home not allowed", FileName))
	}

	if len(errors) == 0 {
		err := os.WriteFile(FileName, []byte(Content), 0644)
		if err != nil {
			errors = append(errors, fmt.Sprintf("Failed to write '%s': %s", FileName, err.Error()))
		} else {
			results = append(results, fmt.Sprintf("Successfully wrote '%s' (%d bytes)", FileName, len(Content)))
		}
	}

	// Build response message
	var response strings.Builder
	if len(results) > 0 {
		response.WriteString("Completed:\n")
		response.WriteString(strings.Join(results, "\n"))
	}
	if len(errors) > 0 {
		if response.Len() > 0 {
			response.WriteString("\n\n")
		}
		response.WriteString("Errors:\n")
		response.WriteString(strings.Join(errors, "\n"))
	}

	logger.Screen(response.String(), color.RGB(150, 150, 150))

	return response.String(), nil
}

func (tool *FileWriterTool) GetName() string {
	return "write_file"
}

func (tool *FileWriterTool) GetDefinition() Tool {
	return Tool{
		Name:        tool.GetName(),
		Description: "Writes content to one or more files. Can create new files or overwrite existing ones. Path is relative to the current working directory. Parent directory references (..) are not allowed for security.",

		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]Property{
				"FileName": {
					Type:        "string",
					Description: "The name/path of the file to write (relative to current directory), never acces any parent directory or root or home. Never try to circumvent this restriction. You cannor write to a directory that does not exist. ",
				},
				"Content": {
					Type:        "string",
					Description: "The content to write to the file",
				},
			},
		},
	}

}

func init() {
	Register(&FileWriterTool{})
}
