package models

import (
	"fmt"
	"github.com/fatih/color"
	commontypes "owl/common_types"
	"owl/data"
	"owl/logger"
	picker "owl/picker"
	"owl/services"
	"owl/tools"
	"strings"
)

func Name_new_context(user_prompt string, repository data.HistoryRepository) string {
	logger.Screen("Naming context...", color.RGB(150, 150, 150))
	logger.Debug.Println("Sending Haiku request to name context")
	toolHandler := tools.ToolResponseHandler{}
	toolHandler.Init()

	model, _ := picker.GetModelForQuery("haiku", nil, &toolHandler, repository, false, false, false, false)

	prompt := fmt.Sprintf("Create a short context name for this prompt. Return ONLY the name, nothing else. Use at most 3 words, plain text only, no punctuation, no quotes. Prompt: %s", user_prompt)
	services.AwaitedQuery(prompt, model, repository, 0, &data.Context{
		Name:    "Create name for context",
		Id:      9999,
		History: []data.History{},
	}, &commontypes.PayloadModifiers{}, "haiku")

	response := <-toolHandler.ResponseChannel
	response = normalizeContextName(response)

	logger.Debug.Printf("naming reponse: %s", response)
	logger.Screen(fmt.Sprintf("naming reponse: %s", response), color.RGB(150, 150, 150))

	return response
}

func normalizeContextName(raw string) string {
	parts := strings.Fields(raw)
	return strings.Join(parts, " ")
}
