package main

import (
	"owl/data"
	"owl/logger"
)

type EmbeddingsResponseHandler struct {
	Db              data.EmbeddingsStore
	Store           bool
	ResponseChannel chan string
	Reference       string
}

func (rh *EmbeddingsResponseHandler) Init() {
	rh.ResponseChannel = make(chan string, 100)
}

func (rh *EmbeddingsResponseHandler) RecievedText(text string, useColor *string) {}

func (rh *EmbeddingsResponseHandler) FinalText(contextId int64, prompt string, response string, responseContent string, toolResults string, modelName string) {

	logger.Debug.Printf("\nFIND ME:embedding: %s\n", response)

	if rh.Store {
		res, err := rh.Db.InsertEmbedding(prompt, response, rh.Reference)
		if err != nil {
			logger.Debug.Printf("error storing embedding: %s", err)
		} else {
			logger.Debug.Printf("res from storing embedding: %s", res)
		}

	}

	if rh.ResponseChannel != nil {
		rh.ResponseChannel = make(chan string, 100)
		rh.ResponseChannel <- response
		close(rh.ResponseChannel)
	}
}
