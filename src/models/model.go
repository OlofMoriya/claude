package models

import (
	"net/http"
	"owl/data"
)

type PayloadModifiers struct {
	Pdf   string
	Web   bool
	Image bool
}

type Model interface {
	CreateRequest(context *data.Context, prompt string, streaming bool, history []data.History, modifiers *PayloadModifiers) *http.Request
	HandleStreamedLine(line []byte)
	HandleBodyBytes(bytes []byte)
}
