package models

import (
	"net/http"
	"owl/data"
)

type Model interface {
	CreateRequest(context *data.Context, prompt string, streaming bool, history []data.History, image bool, pdf string) *http.Request
	HandleStreamedLine(line []byte)
	HandleBodyBytes(bytes []byte)
	SetResponseHandler(responseHandler ResponseHandler)
}
