package tools

import (
	"fmt"
	"os/exec"
	"owl/data"
	"owl/models"
	"owl/services"
	"strings"

	open_ai_responses "owl/models/open-ai-responses"
)

type TrackingNumberLookupInput struct {
	TrackingNumber string
}

type IssueListLookupInput struct {
	Span string
}

type FileListInput struct {
	Filter string
}

type ReadFileInput struct {
	FileNames string
}

type FileWriteInput struct {
	Files []FileWrite
}

type FileWrite struct {
	FileName string
	Content  string
}

type ToolRunner struct {
	ResponseHandler   *models.ResponseHandler
	HistoryRepository *data.HistoryRepository
}

func (toolRunner *ToolRunner) RunTool(name string, input interface{}) interface{} {
	switch name {
	case "image_generation":
		return toolRunner.RunImageGeneration(input.(ImageInput))
	case "early_bird_track_lookup":
		return toolRunner.RunTrackingNumberLookup(input.(TrackingNumberLookupInput))
	case "issue_list":
		return toolRunner.RunIssueListLookup(input.(IssueListLookupInput))
	case "read_files":
		return toolRunner.RunReadFiles(input.(ReadFileInput))
	case "file_list":
		return toolRunner.RunListFiles(input.(FileListInput))
	default:
		return "Err: Could not find any execution for specified tool"

	}
}

type ToolResponseHandler struct {
	ResponseHandler models.ResponseHandler
	ResponseChannel chan string
}

func (toolResponseHandler *ToolResponseHandler) Init() {
	toolResponseHandler.ResponseChannel = make(chan string, 100)
}

func (toolResponseHandler *ToolResponseHandler) RecievedText(text string, color *string) {
	//TODO: Implement streaming tool use
}

func (toolResponseHandler *ToolResponseHandler) FinalText(contextId int64, prompt string, response string, resposneContent string, toolResults string) {
	toolResponseHandler.ResponseHandler.FinalText(contextId, prompt, response, "", "")
	toolResponseHandler.ResponseChannel = make(chan string, 100)
	toolResponseHandler.ResponseChannel <- response
	close(toolResponseHandler.ResponseChannel)
}

type ImageInput struct {
	Prompt  string
	Context *data.Context
}

func (toolRunner *ToolRunner) RunImageGeneration(input ImageInput) string {
	toolHandler := ToolResponseHandler{
		ResponseHandler: *toolRunner.ResponseHandler,
	}
	toolHandler.Init()

	model := &open_ai_responses.OpenAiResponseModel{ResponseHandler: &toolHandler}

	services.AwaitedQuery(input.Prompt, model, *toolRunner.HistoryRepository, 20, input.Context, &models.PayloadModifiers{})
	//I need to await the answer on the channel toolHandler.ResponseChannel and then return with that value.
	response := <-toolHandler.ResponseChannel
	return response
}

func (toolRunner *ToolRunner) RunTrackingNumberLookup(input TrackingNumberLookupInput) string {

	out, err := exec.Command("login-and-status.sh", input.TrackingNumber).Output()
	if err != nil {
		fmt.Printf("Failed to fetch data, %s", err)
	}
	value := string(out)
	return value
}

func (toolRunner *ToolRunner) RunReadFiles(input ReadFileInput) string {
	files := strings.Split(input.FileNames, ";")

	out, err := exec.Command("/bin/cat", files...).Output()
	if err != nil {
		fmt.Printf("Failed to read files, %s", err)
	}
	value := string(out)
	return value
}

func (toolRunner *ToolRunner) RunListFiles(input FileListInput) string {
	// out, err := exec.Command("eza -R").Output()
	// out, err := exec.Command("ls -R").Output()
	out, err := exec.Command("/bin/ls", "-R").Output()
	if err != nil {
		fmt.Printf("Failed to read files, %s", err)
	}
	value := string(out)
	return value
}

func (toolRunner *ToolRunner) RunIssueListLookup(input IssueListLookupInput) string {
	out, err := exec.Command("item-list.sh").Output()
	if err != nil {
		fmt.Printf("Failed to fetch data, %s", err)
	}
	value := string(out)
	return value
}

func GetCustomTools() []Tool {
	return []Tool{
		getImageTool(),
		getIssueListTool(),
		getTrackingNumberLookupTool(),
		getReadFilesTool(),
		getListFilesTool(),
	}
}

func getImageTool() Tool {
	return Tool{
		Name:        "image_generator",
		Description: "Generates images from a prompt. This call takes time so don't generate more than two at the time. Can take a subject and a style and creates a image which it returns as base64 and saves it to disc as png",

		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]Property{
				"Prompt": {
					Type:        "string",
					Description: "Promt with description of the image that should be created",
				},
			},
		},
	}
}

func getListFilesTool() Tool {
	return Tool{
		Name:        "read_files",
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

func getReadFilesTool() Tool {
	return Tool{
		Name:        "file_list",
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

func getIssueListTool() Tool {
	return Tool{
		Name:        "issue_list",
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

func getTrackingNumberLookupTool() Tool {
	return Tool{
		Name:        "early_bird_track_lookup",
		Description: "Fetches status from a shipment trackingnumber in the early bird logistics chain. Can help anwser questions about where a delivery is in the process or if anything went wrong with the shipment.",

		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]Property{
				"TrackingNumber": {
					Type:        "string",
					Description: "TrackingNumber that can be used to look up the order.",
				},
			},
		},
	}
}

// func (toolRunner *ToolRunner) RunTrackingNumberLookup(input TrackingNumberLookupInput) string {
//
// 	out, err := exec.Command("dbask", "-t", input.TrackingNumber).Output()
// 	if err != nil {
// 		fmt.Printf("Failed to fetch data, %s", err)
// 	}
// 	value := string(out)
// 	return value
// }
