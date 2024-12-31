package main

import (
	"fmt"
	data "owl/data"
)

func getContextId(user data.HistoryRepository) int64 {
	context, _ := user.GetContextByName(context_name)

	var context_id int64
	if context == nil {
		new_context := data.Context{Name: context_name}
		id, err := user.InsertContext(new_context)
		if err != nil {
			panic(fmt.Sprintf("Could not create a new context with name %s, %s", context_name, err))
		}
		context_id = id
	} else {
		context_id = context.Id
	}

	return context_id
}
