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
)

func Name_new_context(user_prompt string, repository data.HistoryRepository) string {
	logger.Screen("Naming context...", color.RGB(150, 150, 150))
	logger.Debug.Println("Sending Haiku request to name context")
	toolHandler := tools.ToolResponseHandler{}
	toolHandler.Init()

	model, _ := picker.GetModelForQuery("haiku", nil, &toolHandler, repository, false, false, false, false)

	prompt := fmt.Sprintf("Create a short name for this prompt so that I can store it with a name in a database. Maximum 100 characters but try to keep it short. ONLY EVER answer with the name and nothing else!!!! here's the prompt to name the context for: %s", user_prompt)
	services.AwaitedQuery(prompt, model, repository, 0, &data.Context{
		Name:    "Create name for context",
		Id:      9999,
		History: []data.History{},
	}, &commontypes.PayloadModifiers{}, "haiku")

	response := <-toolHandler.ResponseChannel

	logger.Debug.Printf("naming reponse: %s", response)
	logger.Screen(fmt.Sprintf("naming reponse: %s", response), color.RGB(150, 150, 150))

	return response
}
