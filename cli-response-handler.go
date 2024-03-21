package main

import "fmt"

type CliResponseHandler struct {
	Repository HistoryRepository
}

func (cli CliResponseHandler) recievedText(text string) {
	fmt.Print(text)
}

// All models should call this regardless of if they stream or not.
func (cli CliResponseHandler) finalText(contextId int64, prompt string, response string) {
	history := History{
		ContextId:   contextId,
		Prompt:      prompt,
		Response:    response,
		Abreviation: "",
		TokenCount:  0,
		//TODO abreviation
		//TODO tokencount
	}

	_, err := cli.Repository.insertHistory(history)
	if err != nil {
		fmt.Printf("Error while trying to save history: %s", err)
	}
}
