package models

import (
	"net/http"
	"owl/data"
)

type Model interface {
	CreateRequest(context *data.Context, prompt string, streaming bool, history []data.History) *http.Request
	HandleStreamedLine(line []byte)
	HandleBodyBytes(bytes []byte)
}
