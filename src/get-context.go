package main

import (
	"fmt"
	data "owl/data"
)

func getContext(user data.HistoryRepository, system_prompt *string) *data.Context {

	context, _ := user.GetContextByName(context_name)

	if context == nil {
		new_context := data.Context{Name: context_name, SystemPrompt: *system_prompt}
		id, err := user.InsertContext(new_context)
		if err != nil {
			panic(fmt.Sprintf("Could not create a new context with name %s, %s", context_name, err))
		}
		new_context.Id = id
		context = &new_context
	}
	return context
}
