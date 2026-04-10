package main

import (
	"fmt"
	commontypes "owl/common_types"
	data "owl/data"
	"owl/services"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/glamour"
	color "github.com/fatih/color"
)

type CliResponseHandler struct {
	Repository data.HistoryRepository
}

func (cli CliResponseHandler) RecievedText(text string, useColor *string) {
	if useColor != nil {
		color.RGB(150, 150, 150).Print(text)
	} else {
		fmt.Print(text)
	}
}

// All models should call this regardless of if they stream or not.
func (cli CliResponseHandler) FinalText(contextId int64, prompt string, response string, toolUse []data.ToolUse, modelName string, usage *commontypes.TokenUsage) {
	history := data.History{
		ContextId:    contextId,
		Prompt:       prompt,
		Response:     response,
		Abbreviation: "",
		TokenCount:   0,
		Model:        modelName,
		ToolUse:      toolUse,
	}

	if usage != nil {
		history.PromptTokens = usage.PromptTokens
		history.CompletionTokens = usage.CompletionTokens
		history.CacheReadTokens = usage.CacheReadTokens
		history.CacheWriteTokens = usage.CacheWriteTokens
	}

	_, err := cli.Repository.InsertHistory(history)
	if err != nil {
		println(fmt.Sprintf("Error while trying to save history: %s", err))
	}

	code := services.ExtractCodeBlocks(response)
	allCode := strings.Join(code, "\n\n")

	err = clipboard.WriteAll(allCode)
	if err != nil {
		fmt.Printf("Error copying to clipboard: %v\n", err)
	}

	out, err := glamour.Render(response, "dark")
	if err != nil {
		println(fmt.Sprintf("%v", err))
	}

	fmt.Println(out)
}
