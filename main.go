package main

import (
	"bufio"
	data "claude/data"
	server "claude/http"
	model "claude/models/open-ai"
	services "claude/services"
	"flag"
	"fmt"
	"os"
	"strings"
)

var (
	prompt       string
	context_name string
	historyCount int
	serve        bool
	port         int
	secure       bool
	stream       bool
)

func init() {
	flag.StringVar(
		&prompt,
		"prompt",
		"",
		"The prompt to use for the conversation",
	)
	flag.StringVar(
		&context_name,
		"context_name",
		"misc",
		"The context to provide for the conversation",
	)
	flag.IntVar(
		&historyCount,
		"history",
		0,
		"The number of previous messages to include in the context",
	)
	flag.BoolVar(&serve, "serve", false, "Enable server mode")
	flag.IntVar(&port, "port", 8080, "Port to listen on")
	flag.BoolVar(&secure, "secure", false, "Enable HTTPS")
	flag.BoolVar(&stream, "stream", false, "Enable streaming response")
}

func main() {

	user := data.User{Id: "olof", Name: "olof"}

	flag.Parse()

	var context_id int64 = 0
	historyCount := 0

	if prompt == "" && !serve {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Prompt:")
		prompt, _ = reader.ReadString('\n')
		prompt = strings.TrimSpace(prompt)
	}

	fmt.Println("using context_name", context_name)
	context, _ := user.GetContextByName(context_name)

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

	if serve {
		httpResponseHandler := &server.HttpResponseHandler{}
		model := model.OpenAiModel{ResponseHandler: httpResponseHandler}
		server.Run(secure, port, httpResponseHandler, &model, stream)
	} else {
		cliResponseHandler := CliResponseHandler{Repository: user}
		claudeModel := model.OpenAiModel{ResponseHandler: cliResponseHandler}

		if stream {
			fmt.Println("stream")
			services.StreamedQuery(prompt, &claudeModel, user, historyCount, context_id)
		} else {
			services.AwaitedQuery(prompt, &claudeModel, user, historyCount, context_id)
		}
	}

}
