package tools

import "owl/models"

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

func (toolResponseHandler *ToolResponseHandler) FinalText(contextId int64, prompt string, response string, resposneContent string, toolResults string, modelName string) {
	if toolResponseHandler.ResponseHandler != nil {
		toolResponseHandler.ResponseHandler.FinalText(contextId, prompt, response, resposneContent, toolResults, modelName)
	}
	toolResponseHandler.ResponseChannel = make(chan string, 100)
	toolResponseHandler.ResponseChannel <- response
	close(toolResponseHandler.ResponseChannel)
}
