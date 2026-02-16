package main

import (
	"bufio"
	"log"
	data "owl/data"
	server "owl/http"
	"owl/logger"
	"owl/models"
	claude_model "owl/models/claude"
	grok_model "owl/models/grok"
	ollama_model "owl/models/ollama"
	openai_4o_model "owl/models/open-ai-4o"
	openai_base "owl/models/open-ai-base"
	embeddings_model "owl/models/open-ai-embedings"
	open_ai_gpt_model "owl/models/open-ai-gpt"
	"owl/tools"

	"flag"
	"fmt"
	"os"
	services "owl/services"
	"owl/tui"
	"strings"

	mode "owl/mode"

	"github.com/charmbracelet/glamour"
	"github.com/fatih/color"
	"github.com/joho/godotenv"
)

var (
	prompt           string
	context_name     string
	history_count    int
	serve            bool
	port             int
	secure           bool
	stream           bool
	embeddings       bool
	view             bool
	llm_model        string
	thinking         bool
	stream_thinkning bool
	output_thinkning bool
	system_prompt    string
	image            bool
	pdf              string
	web              bool
	tui_mode         bool
)

func init() {
	if err := logger.Init("~/.owl/debug.log"); err != nil {
		log.Fatal("Failed to initialize logger:", err)
	}

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
		1,
		"The number of previous messages to include in the context",
	)
	flag.BoolVar(&serve, "serve", false, "Enable server mode")
	flag.IntVar(&port, "port", 3000, "Port to listen on")
	flag.BoolVar(&secure, "secure", false, "Enable HTTPS")
	flag.BoolVar(&stream, "stream", false, "Enable streaming response")
	flag.BoolVar(&embeddings, "embeddings", false, "Enable embeddings generation (no streaming)")
	flag.StringVar(&llm_model, "model", "claude", "set model used for the call")

	flag.BoolVar(&thinking, "thinking", true, "use thinking in request")
	flag.BoolVar(&stream_thinkning, "stream_thinking", true, "stream thinking")
	flag.BoolVar(&output_thinkning, "output_thinking", false, "output thinking")
	flag.StringVar(&system_prompt, "system", "", "set a system promt for the context")
	flag.BoolVar(&view, "view", false, "view")
	flag.BoolVar(&image, "image", false, "image (used clipboard as image)")
	flag.BoolVar(&web, "web", false, "web search enabled")
	flag.StringVar(&pdf, "pdf", "", "path to pdf")
	flag.BoolVar(&tui_mode, "tui", false, "Launch TUI mode")
}

func main() {
	godotenv.Load()

	flag.Parse()

	if serve {
		logger.Screen("\nsetting mode to REMOTE\n", color.RGB(150, 150, 150))
		mode.Mode = tools.REMOTE
	} else {
		mode.Mode = tools.LOCAL
	}

	if tui_mode {
		launchTUI()
		return
	}

	if system_prompt != "" && context_name != "" {
		db := os.Getenv("OWL_LOCAL_DATABASE")
		if db == "" {
			db = "owl"
		}

		user := data.User{Name: &db}
		context := getContext(user, &system_prompt)

		err := user.UpdateSystemPrompt(context.Id, system_prompt)
		if err != nil {
			println(err)
		}
		return
	}

	if prompt == "" && !serve && !view {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Prompt:")
		prompt, _ = reader.ReadString('\n')
		prompt = strings.TrimSpace(prompt)
	}

	if serve {
		httpResponseHandler := &server.HttpResponseHandler{}
		user := &data.MultiUserContext{}
		httpResponseHandler.Repository = user

		model := &claude_model.ClaudeModel{UseStreaming: stream, HistoryRepository: user, ResponseHandler: httpResponseHandler, UseThinking: thinking, StreamThought: stream_thinkning, OutputThought: output_thinkning, ModelVersion: "haiku"}

		server.Run(secure, port, httpResponseHandler, model, stream)
	} else if embeddings {
		db := os.Getenv("OWL_LOCAL_DATABASE")
		if db == "" {
			db = "owl"
		}

		user := data.User{Name: &db}
		embeddingsResponseHandler := EmbeddingsResponseHandler{}
		model := embeddings_model.OpenAiEmbeddingsModel{ResponseHandler: &embeddingsResponseHandler}

		services.AwaitedQuery(prompt, &model, user, 0, nil, &models.PayloadModifiers{}, "embeddings")
	} else if view {
		view_history()
	} else {
		db := os.Getenv("OWL_LOCAL_DATABASE")
		if db == "" {
			db = "owl"
		}

		user := data.User{Name: &db}
		cliResponseHandler := CliResponseHandler{Repository: user}
		context := getContext(user, &system_prompt)

		// Use model selector to get the appropriate model and model name
		model, modelName := getModelForQuery(llm_model, context, cliResponseHandler, user, stream, thinking, stream_thinkning, output_thinkning)

		if stream {
			services.StreamedQuery(prompt, model, user, history_count, context, &models.PayloadModifiers{Image: image, Pdf: pdf, Web: web}, modelName)
		} else {
			services.AwaitedQuery(prompt, model, user, history_count, context, &models.PayloadModifiers{Image: image, Pdf: pdf, Web: web}, modelName)
		}
	}
}

// getModelForQuery returns the appropriate model based on the request and context preferences
func getModelForQuery(
	requestedModel string,
	context *data.Context,
	responseHandler models.ResponseHandler,
	historyRepository data.HistoryRepository,
	streamMode bool,
	thinkingMode bool,
	streamThinkingMode bool,
	outputThinkingMode bool,
) (models.Model, string) {

	// Determine which model to use
	modelToUse := requestedModel

	// If no model was explicitly requested, check context preferences
	if modelToUse == "" && context != nil && context.PreferredModel != "" {
		modelToUse = context.PreferredModel
	}

	// Default to claude if still not set
	if modelToUse == "" {
		modelToUse = "claude"
	}

	// Create and return the appropriate model
	var model models.Model

	switch modelToUse {
	case "grok":
		model = &grok_model.GrokModel{
			OpenAICompatibleModel: openai_base.OpenAICompatibleModel{
				ResponseHandler:   responseHandler,
				HistoryRepository: historyRepository,
			},
		}
	case "4o":
		model = &openai_4o_model.OpenAi4oModel{
			ResponseHandler:   responseHandler,
			HistoryRepository: historyRepository,
		}
	case "gpt":
		model = &open_ai_gpt_model.OpenAIGPTModel{
			OpenAICompatibleModel: openai_base.OpenAICompatibleModel{
				ResponseHandler:   responseHandler,
				HistoryRepository: historyRepository,
			},
			ModelVersion: "gpt",
		}
	case "codex":
		model = &open_ai_gpt_model.OpenAIGPTModel{
			OpenAICompatibleModel: openai_base.OpenAICompatibleModel{
				ResponseHandler:   responseHandler,
				HistoryRepository: historyRepository,
			},
			ModelVersion: "codex",
		}
	case "qwen3":
		model = ollama_model.NewOllamaModel(responseHandler, "")
	case "opus":
		model = &claude_model.ClaudeModel{
			UseStreaming:      streamMode,
			HistoryRepository: historyRepository,
			ResponseHandler:   responseHandler,
			UseThinking:       thinkingMode,
			StreamThought:     streamThinkingMode,
			OutputThought:     outputThinkingMode,
			ModelVersion:      "opus",
		}
	case "sonnet":
		model = &claude_model.ClaudeModel{
			UseStreaming:      streamMode,
			HistoryRepository: historyRepository,
			ResponseHandler:   responseHandler,
			UseThinking:       thinkingMode,
			StreamThought:     streamThinkingMode,
			OutputThought:     outputThinkingMode,
			ModelVersion:      "sonnet",
		}
	case "haiku":
		model = &claude_model.ClaudeModel{
			UseStreaming:      streamMode,
			HistoryRepository: historyRepository,
			ResponseHandler:   responseHandler,
			UseThinking:       thinkingMode,
			StreamThought:     streamThinkingMode,
			OutputThought:     outputThinkingMode,
			ModelVersion:      "haiku",
		}
	case "claude":
		fallthrough
	default:
		model = &claude_model.ClaudeModel{
			UseStreaming:      streamMode,
			HistoryRepository: historyRepository,
			ResponseHandler:   responseHandler,
			UseThinking:       thinkingMode,
			StreamThought:     streamThinkingMode,
			OutputThought:     outputThinkingMode,
		}
		modelToUse = "claude"
	}

	return model, modelToUse
}

func view_history() {
	if context_name == "" {
		panic("No context name to output")
	}

	db := os.Getenv("OWL_LOCAL_DATABASE")
	if db == "" {
		db = "owl"
	}

	user := data.User{Name: &db}

	context, err := user.GetContextByName(context_name)
	if err != nil {
		panic(err)
	}

	count := 100
	if history_count > 0 {
		count = history_count
	}

	history, err := user.GetHistoryByContextId(context.Id, count)
	if err != nil {
		panic(err)
	}

	out, err := glamour.Render(fmt.Sprintf("# %s\n%s", context_name, context.Created), "dark")
	if err != nil {
		println(fmt.Sprintf("%v", err))
	}
	fmt.Println(out)

	for _, h := range history {
		out, err := glamour.Render(fmt.Sprintf("--- \n## Q\n\n %s \n\n## A\n\n %s", h.Prompt, h.Response), "dark")
		if err != nil {
			println(fmt.Sprintf("%v", err))
		}
		fmt.Println(out)
	}
}

func launchTUI() {
	db := os.Getenv("OWL_LOCAL_DATABASE")
	if db == "" {
		db = "owl"
	}

	user := data.User{Name: &db}

	cliResponseHandler := CliResponseHandler{Repository: user}
	model := &claude_model.ClaudeModel{
		ResponseHandler:   cliResponseHandler,
		HistoryRepository: user,
		UseThinking:       true,
		StreamThought:     false,
		OutputThought:     false,
	}

	config := tui.TUIConfig{
		Repository:   user,
		Model:        model,
		HistoryCount: 10,
	}

	if err := tui.Run(config); err != nil {
		log.Fatal(err)
	}
}
