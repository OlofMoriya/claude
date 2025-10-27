package tools

import (
	"fmt"
	"os/exec"
	"owl/data"
	"owl/models"
	"owl/services"

	open_ai_responses "owl/models/open-ai-responses"
)

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

func (toolResponseHandler *ToolResponseHandler) FinalText(contextId int64, prompt string, response string, resposneContent string) {
	toolResponseHandler.ResponseHandler.FinalText(contextId, prompt, response, "")
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

	services.AwaitedQuery(input.Prompt, model, *toolRunner.HistoryRepository, 0, input.Context, &models.PayloadModifiers{})
	//I need to await the answer on the channel toolHandler.ResponseChannel and then return with that value.
	response := <-toolHandler.ResponseChannel
	return response
}

type TrackingNumberLookupInput struct {
	TrackingNumber string
}

func (toolRunner *ToolRunner) RunTrackingNumberLookup(input TrackingNumberLookupInput) string {

	out, err := exec.Command("login-and-status.sh", input.TrackingNumber).Output()
	if err != nil {
		fmt.Printf("Failed to fetch data, %s", err)
	}
	value := string(out)
	return value
}

type IssueListLookupInput struct {
	Span string
}

func (toolRunner *ToolRunner) RunIssueListLookup(input IssueListLookupInput) string {
	out, err := exec.Command("item-list.sh").Output()
	if err != nil {
		fmt.Printf("Failed to fetch data, %s", err)
	}
	value := string(out)
	return value
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
