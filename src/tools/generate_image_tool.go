package tools

import (
	"fmt"
	"owl/data"
	"owl/logger"
	"owl/models"
	open_ai_responses "owl/models/open-ai-responses"
	"owl/services"

	"github.com/fatih/color"
)

type GenerateImageTool struct {
	ResponseHandler   models.ResponseHandler
	HistoryRepository *data.HistoryRepository
	Context           *data.Context
}

type GenerateImageInput struct {
	Prompt string
}

func (tool *GenerateImageTool) SetHistory(repo *data.HistoryRepository, context *data.Context) {
	tool.Context = context
	tool.HistoryRepository = repo
}

func (tool *GenerateImageTool) Run(i map[string]string) (string, error) {
	prompt, exists := i["Prompt"]
	if !exists {
		return "", fmt.Errorf("Could not parse FileWriteInput from input")
	}

	logger.Screen(fmt.Sprintf("Asked to generate image with prompt: %v", prompt), color.RGB(150, 150, 150))

	toolHandler := ToolResponseHandler{
		ResponseHandler: tool.ResponseHandler,
	}
	toolHandler.Init()

	model := &open_ai_responses.OpenAiResponseModel{ResponseHandler: &toolHandler}

	services.AwaitedQuery(prompt, model, *tool.HistoryRepository, 0, tool.Context, &models.PayloadModifiers{})
	//I need to await the answer on the channel toolHandler.ResponseChannel and then return with that value.
	response := <-toolHandler.ResponseChannel
	return response, nil
}

func (tool *GenerateImageTool) GetName() string {
	return "image_generator"
}

func (tool *GenerateImageTool) GetDefinition() Tool {
	return Tool{
		Name:        tool.GetName(),
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

func init() {
	//todo: need the context?
	Register(&GenerateImageTool{})
}
