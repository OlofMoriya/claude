package tools

import (
	"fmt"
	"os/exec"
)

type ListFilesTool struct {
}

type FileListInput struct {
	Filter string
}

func (ool *ListFilesTool) GetName() string {
	return "read_files"
}

func (tool *ListFilesTool) GetDefinition() Tool {
	return Tool{
		Name:        tool.GetName(),
		Description: "Fetches the contents of the files specified by name and dynamic path. Path starts from where script is being executed. Only read files with .go, .md, .tsx, .ts, .csv, .js, .txt extentions.",

		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]Property{
				"FileNames": {
					Type:        "string",
					Description: "A list of file names for which the tool will fetch the content and return it to the model making the request. The list is seperated with the ; character",
				},
			},
		},
	}
}

func (tool *ListFilesTool) Run(i map[string]string) (string, error) {
	out, err := exec.Command("/bin/ls", "-R").Output()
	if err != nil {
		fmt.Printf("Failed to read files, %s", err)
	}
	value := string(out)
	return value, nil
}

func init() {
	Register(&ListFilesTool{})
}
