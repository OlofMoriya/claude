package main

import (
	"bufio"
	data "claude/data"
	server "claude/http"
	"claude/models"
	claude_model "claude/models/claude"
	openai_4o_model "claude/models/open-ai-4o"
	embeddings_model "claude/models/open-ai-embedings"
	// openai_vision_model "claude/models/open-ai-vision"
	services "claude/services"
	"flag"
	"fmt"
	"os"
	"strings"
)

var (
	prompt        string
	context_name  string
	history_count int
	serve         bool
	port          int
	secure        bool
	stream        bool
	embeddings    bool
	llm_model     string
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
		&history_count,
		"history",
		0,
		"The number of previous messages to include in the context",
	)
	flag.BoolVar(&serve, "serve", false, "Enable server mode")
	flag.IntVar(&port, "port", 8080, "Port to listen on")
	flag.BoolVar(&secure, "secure", false, "Enable HTTPS")
	flag.BoolVar(&stream, "stream", false, "Enable streaming response")
	flag.BoolVar(&embeddings, "embeddings", false, "Enable embeddings generation (no streaming)")
	flag.StringVar(&llm_model, "model", "claude", "set model used for the call")
}

func main() {

	flag.Parse()

	if prompt == "" && !serve {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Prompt:")
		prompt, _ = reader.ReadString('\n')
		prompt = strings.TrimSpace(prompt)
	}

	if serve {
		httpResponseHandler := &server.HttpResponseHandler{}
		model := claude_model.ClaudeModel{ResponseHandler: httpResponseHandler}
		server.Run(secure, port, httpResponseHandler, &model, stream)
	} else if embeddings {
		user := data.User{Id: "olof", Name: "olof"}
		embeddingsResponseHandler := EmbeddingsResponseHandler{}
		model := embeddings_model.OpenAiEmbeddingsModel{ResponseHandler: &embeddingsResponseHandler}
		services.AwaitedQuery(prompt, &model, user, 0, 0)
	} else {
		user := data.User{Id: "olof", Name: "olof"}

		var model models.Model
		cliResponseHandler := CliResponseHandler{Repository: user}

		switch llm_model {
		case "4o":
			println("using 4o")
			model = &openai_4o_model.OpenAi4oModel{ResponseHandler: cliResponseHandler}
		case "claude":
			model = &claude_model.ClaudeModel{ResponseHandler: cliResponseHandler}
		default:
			model = &claude_model.ClaudeModel{ResponseHandler: cliResponseHandler}
		}
		//TODO: Select database
		context_id := getContextId(user)

		if stream {
			services.StreamedQuery(prompt, model, user, history_count, context_id)
		} else {
			services.AwaitedQuery(prompt, model, user, history_count, context_id)
		}
	}
}

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
