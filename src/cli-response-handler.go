package main

import (
	"fmt"
	data "owl/data"

	"github.com/charmbracelet/glamour"
)

type CliResponseHandler struct {
	Repository data.HistoryRepository
}

func (cli CliResponseHandler) RecievedText(text string) {
	print(text)
}

// All models should call this regardless of if they stream or not.
func (cli CliResponseHandler) FinalText(contextId int64, prompt string, response string) {
	history := data.History{
		ContextId:    contextId,
		Prompt:       prompt,
		Response:     response,
		Abbreviation: "",
		TokenCount:   0,
		//TODO abreviation
		//TODO tokencount
	}

	_, err := cli.Repository.InsertHistory(history)
	if err != nil {
		println(fmt.Sprintf("Error while trying to save history: %s", err))
	}

	out, err := glamour.Render(response, "dark")
	if err != nil {
		println(fmt.Sprintf("%v", err))
	}

	fmt.Println(out)
}
