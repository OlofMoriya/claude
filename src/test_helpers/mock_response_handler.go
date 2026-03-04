package testhelpers

import "sync"

type TextEvent struct {
	Text  string
	Color *string
}

type FinalEvent struct {
	ContextID       int64
	Prompt          string
	Response        string
	ResponseContent string
	ToolResults     string
	ModelName       string
}

type MockResponseHandler struct {
	mu          sync.Mutex
	TextEvents  []TextEvent
	FinalEvents []FinalEvent
}

func NewMockResponseHandler() *MockResponseHandler {
	return &MockResponseHandler{}
}

func (m *MockResponseHandler) RecievedText(text string, color *string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.TextEvents = append(m.TextEvents, TextEvent{Text: text, Color: color})
}

func (m *MockResponseHandler) FinalText(contextId int64, prompt string, response string, responseContent string, toolResults string, modelName string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.FinalEvents = append(m.FinalEvents, FinalEvent{
		ContextID:       contextId,
		Prompt:          prompt,
		Response:        response,
		ResponseContent: responseContent,
		ToolResults:     toolResults,
		ModelName:       modelName,
	})
}

func (m *MockResponseHandler) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.TextEvents = nil
	m.FinalEvents = nil
}

func (m *MockResponseHandler) CopyTextEvents() []TextEvent {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]TextEvent, len(m.TextEvents))
	copy(result, m.TextEvents)
	return result
}

func (m *MockResponseHandler) CopyFinalEvents() []FinalEvent {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]FinalEvent, len(m.FinalEvents))
	copy(result, m.FinalEvents)
	return result
}
