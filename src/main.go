package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	commontypes "owl/common_types"
	data "owl/data"
	"owl/embeddings"
	server "owl/http"
	"owl/logger"
	mode "owl/mode"
	claude_model "owl/models/claude"
	picker "owl/picker"
	"owl/services"
	"owl/tools"
	"owl/tui"

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
	store            bool
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
	search           string
	chunk            string
)

func init() {
	if err := logger.Init("~/.owl/debug.log"); err != nil {
		log.Fatal("Failed to initialize logger:", err)
	}

	flag.StringVar(&prompt, "prompt", "", "The prompt to use for the conversation")
	flag.StringVar(&context_name, "context_name", "misc", "The context to provide for the conversation")
	flag.IntVar(&history_count, "history", 1, "The number of previous messages to include in the context")
	flag.BoolVar(&serve, "serve", false, "Enable server mode")
	flag.IntVar(&port, "port", 3000, "Port to listen on")
	flag.BoolVar(&secure, "secure", false, "Enable HTTPS")
	flag.BoolVar(&stream, "stream", false, "Enable streaming response")
	flag.BoolVar(&store, "embeddings", false, "Enable embeddings generation (no streaming)")
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
	flag.StringVar(&search, "search", "", "search for phrase in embedding")
	flag.StringVar(&chunk, "chunk", "", "path to markdown document that should be chunked and stored as embeddings")
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

	if prompt == "" && !serve && !view && search == "" && chunk == "" {
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
		return
	}

	if store {
		if _, err := embeddings.Run(embeddings.Config{
			Store:     true,
			ChunkPath: chunk,
			Prompt:    prompt,
		}); err != nil {
			panic(err)
		}
		return
	}

	if search != "" {
		matches, err := embeddings.Run(embeddings.Config{
			Store:       false,
			SearchQuery: search,
		})
		if err != nil {
			panic(err)
		}

		rag_string := ""
		for i, match := range matches {
			if i == 0 || match.Distance < 1 {
				rag_string = fmt.Sprintf("%s\n----\n%s\n%s", rag_string, match.Text, match.Reference)
			}
		}

		search_prompt := fmt.Sprintf("Q: %s\nMatches from RAG: %s", search, rag_string)

		db := os.Getenv("OWL_LOCAL_DATABASE")
		if db == "" {
			db = "owl"
		}
		user := data.User{Name: &db}
		cliResponseHandler := CliResponseHandler{Repository: user}
		context := getContext(user, &system_prompt)
		model, modelName := picker.GetModelForQuery(llm_model, context, cliResponseHandler, user, stream, thinking, stream_thinkning, output_thinkning)
		services.AwaitedQuery(search_prompt, model, user, history_count, context, &commontypes.PayloadModifiers{Image: image, Pdf: pdf, Web: web}, modelName)
		return
	}

	if view {
		view_history()
		return
	}

	db := os.Getenv("OWL_LOCAL_DATABASE")
	if db == "" {
		db = "owl"
	}

	user := data.User{Name: &db}
	cliResponseHandler := CliResponseHandler{Repository: user}
	context := getContext(user, &system_prompt)

	model, modelName := picker.GetModelForQuery(llm_model, context, cliResponseHandler, user, stream, thinking, stream_thinkning, output_thinkning)

	if stream {
		services.StreamedQuery(prompt, model, user, history_count, context, &commontypes.PayloadModifiers{Image: image, Pdf: pdf, Web: web}, modelName)
	} else {
		services.AwaitedQuery(prompt, model, user, history_count, context, &commontypes.PayloadModifiers{Image: image, Pdf: pdf, Web: web}, modelName)
	}
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
	model := &claude_model.ClaudeModel{ResponseHandler: cliResponseHandler, HistoryRepository: user, UseThinking: true, StreamThought: false, OutputThought: false}

	config := tui.TUIConfig{Repository: user, Model: model, HistoryCount: 10}

	if err := tui.Run(config); err != nil {
		log.Fatal(err)
	}
}
