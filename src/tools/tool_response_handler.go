package tools

import (
	commontypes "owl/common_types"
	"owl/data"
)

type ToolResponseHandler struct {
	ResponseHandler commontypes.ResponseHandler
	ResponseChannel chan string
}

func (toolResponseHandler *ToolResponseHandler) Init() {
	toolResponseHandler.ResponseChannel = make(chan string, 100)
}

func (toolResponseHandler *ToolResponseHandler) RecievedText(text string, color *string) {
	//TODO: Implement streaming tool use
}

func (toolResponseHandler *ToolResponseHandler) FinalText(contextId int64, prompt string, response string, toolUse []data.ToolUse, modelName string, usage *commontypes.TokenUsage) {
	if toolResponseHandler.ResponseHandler != nil {
		toolResponseHandler.ResponseHandler.FinalText(contextId, prompt, response, toolUse, modelName, usage)
	}
	toolResponseHandler.ResponseChannel = make(chan string, 100)
	toolResponseHandler.ResponseChannel <- response
	close(toolResponseHandler.ResponseChannel)
}
