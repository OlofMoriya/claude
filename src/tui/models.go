package tui

import (
	"owl/data"
)

type viewMode int

const (
	listView viewMode = iota
	chatView
)

type contextItem struct {
	context      data.Context
	messageCount int
}

// Shared state across views
type sharedState struct {
	config      TUIConfig
	contexts    []contextItem
	selectedCtx *data.Context
	err         error
	width       int
	height      int
}
