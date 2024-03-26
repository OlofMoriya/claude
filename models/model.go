package models

import (
	"claude/data"
	"net/http"
)

type Model interface {
	CreateRequest(contextId int64, prompt string, streaming bool, history []data.History) *http.Request
	HandleStreamedLine(line []byte)
	HandleBodyBytes(bytes []byte)
}
