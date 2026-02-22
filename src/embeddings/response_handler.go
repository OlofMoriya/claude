package embeddings

import (
	"owl/data"
	"owl/logger"
)

// ResponseHandler is used with the OpenAI embeddings model.
// When Store is true, it persists embeddings to the configured store.
// When Store is false, it returns the embedding string on ResponseChannel.
// Reference is stored alongside the text when Store is true.

type ResponseHandler struct {
	Db              data.EmbeddingsStore
	Store           bool
	ResponseChannel chan string
	Reference       string
}

func (rh *ResponseHandler) Init() {
	rh.ResponseChannel = make(chan string, 100)
}

func (rh *ResponseHandler) RecievedText(text string, useColor *string) {}

func (rh *ResponseHandler) FinalText(contextId int64, prompt string, response string, responseContent string, toolResults string, modelName string) {
	logger.Debug.Printf("\nFIND ME:embedding: %s\n", response)

	if rh.Store {
		res, err := rh.Db.InsertEmbedding(prompt, response, rh.Reference)
		if err != nil {
			logger.Debug.Printf("error storing embedding: %s", err)
		} else {
			logger.Debug.Printf("res from storing embedding: %d", res)
		}
	}

	if rh.ResponseChannel != nil {
		// Send the embedding back to callers (e.g. search) and close.
		rh.ResponseChannel <- response
		close(rh.ResponseChannel)
	}
}
