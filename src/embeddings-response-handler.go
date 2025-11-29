package main

import "fmt"

type EmbeddingsResponseHandler struct {
}

func (rh *EmbeddingsResponseHandler) RecievedText(text string, useColor *string) {
}

func (rh *EmbeddingsResponseHandler) FinalText(contextId int64, prompt string, response string, responseContent string, toolResults string) {
	fmt.Printf("{\"prompt\":%s, \"embedding\": %s}", prompt, response)
}
