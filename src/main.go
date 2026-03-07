package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	commontypes "owl/common_types"
	data "owl/data"
	"owl/embeddings"
	server "owl/http"
	"owl/logger"
	mode "owl/mode"
	"owl/models"
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
	create_context   bool
	mardown_path     string
	tool_groups      string
	skillsFlag       string
)

var (
	runServerFunc        = server.Run
	runEmbeddingsFunc    = embeddings.Run
	awaitedQueryFunc     = services.AwaitedQuery
	streamedQueryFunc    = services.StreamedQuery
	launchTUIFunc        = launchTUI
	viewHistoryFunc      = view_history
	nameNewContextFunc   = models.Name_new_context
	getContextFunc       = getContext
	getModelForQueryFunc = picker.GetModelForQuery
)

func init() {
	if err := logger.Init("~/.owl/debug.log"); err != nil {
		log.Fatal("Failed to initialize logger:", err)
	}
	registerFlags(flag.CommandLine)
}

func registerFlags(fs *flag.FlagSet) {
	fs.StringVar(&prompt, "prompt", "", "The prompt to use for the conversation")
	fs.StringVar(&context_name, "context_name", "misc", "The context to provide for the conversation")
	fs.IntVar(&history_count, "history", services.DefaultHistoryCount, "The number of previous messages to include in the context")
	fs.BoolVar(&serve, "serve", false, "Enable server mode")
	fs.IntVar(&port, "port", 3000, "Port to listen on")
	fs.BoolVar(&secure, "secure", false, "Enable HTTPS")
	fs.BoolVar(&stream, "stream", false, "Enable streaming response")
	fs.BoolVar(&store, "embeddings", false, "Enable embeddings generation (no streaming)")
	fs.StringVar(&llm_model, "model", "claude", "set model used for the call")

	fs.BoolVar(&thinking, "thinking", true, "use thinking in request")
	fs.BoolVar(&stream_thinkning, "stream_thinking", true, "stream thinking")
	fs.BoolVar(&output_thinkning, "output_thinking", false, "output thinking")
	fs.StringVar(&system_prompt, "system", "", "set a system promt for the context")
	fs.BoolVar(&view, "view", false, "view")
	fs.BoolVar(&image, "image", false, "image (used clipboard as image)")
	fs.BoolVar(&web, "web", false, "web search enabled")
	fs.StringVar(&pdf, "pdf", "", "path to pdf")
	fs.BoolVar(&tui_mode, "tui", false, "Launch TUI mode")
	fs.StringVar(&search, "search", "", "search for phrase in embedding")
	fs.StringVar(&chunk, "chunk", "", "path to markdown document that should be chunked and stored as embeddings")
	fs.BoolVar(&create_context, "create_context", false, "create a context with proper system prompt")
	fs.StringVar(&mardown_path, "path", "", "mardown path")
	fs.StringVar(&tool_groups, "tools", "", "comma-separated list of tool groups to enable")
	fs.StringVar(&skillsFlag, "skills", "", "comma-separated list of skill files located under ~/.owl/skills")
}

func main() {
	godotenv.Load()
	flag.Parse()
	skillNames := parseSkillNames(skillsFlag)
	sessionSystemPrompt := combineSystemPrompt(system_prompt, skillNames)

	if serve {
		logger.Screen("\nsetting mode to REMOTE\n", color.RGB(150, 150, 150))
		mode.Mode = tools.REMOTE
	} else {
		mode.Mode = tools.LOCAL
	}

	if tui_mode {
		launchTUIFunc()
		return
	}

	if system_prompt != "" && context_name != "" {
		db := os.Getenv("OWL_LOCAL_DATABASE")
		if db == "" {
			db = "owl"
		}

		user := data.User{Name: &db}
		context := getContextFunc(user, &sessionSystemPrompt)

		err := user.UpdateSystemPrompt(context.Id, sessionSystemPrompt)
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

	if create_context && prompt != "" {

		db := os.Getenv("OWL_LOCAL_DATABASE")
		if db == "" {
			db = "owl"
		}
		user := data.User{Name: &db}
		cliResponseHandler := CliResponseHandler{Repository: user}
		toolResponseHandler := tools.ToolResponseHandler{ResponseHandler: cliResponseHandler}

		context_name := nameNewContextFunc(prompt, user)
		new_context := data.Context{Name: context_name}
		id, err := user.InsertContext(new_context)
		if err != nil {
			log.Println(fmt.Sprintf("Could not create a new context with name %s for user %s, %s", context_name, *user.Name, err))
		}

		context := &new_context
		context.Id = id
		context.SystemPrompt = sessionSystemPrompt

		model, modelName := getModelForQueryFunc("haiku", context, &toolResponseHandler, user, stream, thinking, stream_thinkning, output_thinkning)

		// send with proper instructions and catch the answer
		awaitedQueryFunc("", model, user, history_count, context, &commontypes.PayloadModifiers{}, modelName)

		response := <-toolResponseHandler.ResponseChannel

		//context_name?
		//create context and store the systemsprompt
		getContext(user, &response)

		return
	}

	if serve {
		httpResponseHandler := &server.HttpResponseHandler{}
		user := &data.MultiUserContext{}
		httpResponseHandler.Repository = user

		model := &claude_model.ClaudeModel{UseStreaming: stream, HistoryRepository: user, ResponseHandler: httpResponseHandler, UseThinking: thinking, StreamThought: stream_thinkning, OutputThought: output_thinkning, ModelVersion: "haiku"}

		runServerFunc(secure, port, httpResponseHandler, model, stream)
		return
	}

	if store {
		if _, err := runEmbeddingsFunc(embeddings.Config{
			Store:     true,
			ChunkPath: chunk,
			Prompt:    prompt,
		}); err != nil {
			panic(err)
		}
		return
	}

	if search != "" {
		matches, err := runEmbeddingsFunc(embeddings.Config{
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
		context := getContextFunc(user, &sessionSystemPrompt)
		if strings.TrimSpace(sessionSystemPrompt) != "" {
			context.SystemPrompt = sessionSystemPrompt
		}
		model, modelName := getModelForQueryFunc(llm_model, context, cliResponseHandler, user, stream, thinking, stream_thinkning, output_thinkning)
		awaitedQueryFunc(search_prompt, model, user, history_count, context, &commontypes.PayloadModifiers{Image: image, Pdf: pdf, Web: web}, modelName)
		return
	}

	if view {
		viewHistoryFunc()
		return
	}

	db := os.Getenv("OWL_LOCAL_DATABASE")
	if db == "" {
		db = "owl"
	}

	user := data.User{Name: &db}
	cliResponseHandler := CliResponseHandler{Repository: user}
	context := getContextFunc(user, &sessionSystemPrompt)
	if strings.TrimSpace(sessionSystemPrompt) != "" {
		context.SystemPrompt = sessionSystemPrompt
	}

	model, modelName := getModelForQueryFunc(llm_model, context, cliResponseHandler, user, stream, thinking, stream_thinkning, output_thinkning)

	modifiers := &commontypes.PayloadModifiers{Image: image, Pdf: pdf, Web: web}

	var filterGroups []string
	if tool_groups != "" {
		filterGroups = strings.Split(tool_groups, ",")
		modifiers.ToolGroupFilters = filterGroups
	}

	if stream {
		streamedQueryFunc(prompt, model, user, history_count, context, modifiers, modelName)
	} else {
		awaitedQueryFunc(prompt, model, user, history_count, context, modifiers, modelName)
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

	count := services.DefaultHistoryCount
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

	config := tui.TUIConfig{Repository: user, Model: model, HistoryCount: services.DefaultHistoryCount}

	if err := tui.Run(config); err != nil {
		log.Fatal(err)
	}
}

func parseSkillNames(raw string) []string {
	parts := strings.Split(raw, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		result = append(result, trimmed)
	}
	return result
}

func combineSystemPrompt(base string, skillNames []string) string {
	trimmedBase := strings.TrimSpace(base)
	if len(skillNames) == 0 {
		return trimmedBase
	}
	dir, err := skillsDirectory()
	if err != nil {
		logger.Screen(fmt.Sprintf("unable to load skills: %v", err), color.RGB(250, 150, 150))
		logger.Debug.Printf("skills directory error: %v", err)
		return trimmedBase
	}
	builder := strings.Builder{}
	if trimmedBase != "" {
		builder.WriteString(trimmedBase)
	}
	for _, skill := range skillNames {
		name := strings.TrimSpace(skill)
		if name == "" {
			continue
		}
		fileName := filepath.Base(name)
		path := filepath.Join(dir, fileName)
		content, err := os.ReadFile(path)
		if err != nil {
			logger.Screen(fmt.Sprintf("skill %s missing: %v", fileName, err), color.RGB(250, 150, 150))
			logger.Debug.Printf("skill load failure for %s: %v", path, err)
			continue
		}
		if builder.Len() > 0 {
			builder.WriteString("\n\n")
		}
		displayName := strings.TrimSuffix(fileName, filepath.Ext(fileName))
		builder.WriteString(fmt.Sprintf("## Skill: %s\n%s", displayName, strings.TrimSpace(string(content))))
	}
	return builder.String()
}

func skillsDirectory() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".owl", "skills"), nil
}
