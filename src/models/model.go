package models

import (
	"net/http"
	"owl/data"
)

type ToolResponse struct {
	Id              string
	Response        string
	ResponseMessage interface{}
}

type PayloadModifiers struct {
	ToolResponses []ToolResponse
	Pdf           string
	Web           bool
	Image         bool
}

type Model interface {
	CreateRequest(context *data.Context, prompt string, streaming bool, history []data.History, modifiers *PayloadModifiers) *http.Request
	HandleStreamedLine(line []byte)
	HandleBodyBytes(bytes []byte)
	SetResponseHandler(responseHandler ResponseHandler)
}
