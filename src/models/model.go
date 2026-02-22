package models

import (
	"net/http"
	"owl/common_types"
	"owl/data"
)

type Model interface {
	CreateRequest(context *data.Context, prompt string, streaming bool, history []data.History, modifiers *commontypes.PayloadModifiers) *http.Request
	HandleStreamedLine(line []byte)
	HandleBodyBytes(bytes []byte)
	SetResponseHandler(responseHandler ResponseHandler)
}
