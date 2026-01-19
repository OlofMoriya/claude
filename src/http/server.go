package server

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	data "owl/data"
	"owl/logger"
	models "owl/models"
	claude_model "owl/models/claude"
	grok_model "owl/models/grok"
	ollama_model "owl/models/ollama"
	openai_4o_model "owl/models/open-ai-4o"
	openai_base "owl/models/open-ai-base"
	"owl/services"
	tools "owl/tools"
	"strconv"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/golang-jwt/jwt/v5"
)

type server_data struct {
	model               models.Model
	responseHandler     *HttpResponseHandler
	streaming           bool
	db_connectionString string
}

func Run(secure bool, port int, responseHandler *HttpResponseHandler, model models.Model, streaming bool) {

	server_data := server_data{
		model:           model,
		responseHandler: responseHandler,
		streaming:       streaming,
	}

	log.Println("server running on port", port)

	http.HandleFunc("/", server_data.handleRoot)
	http.HandleFunc("/api/prompt", server_data.handlePrompt)
	http.HandleFunc("/api/login", server_data.handleLogin)
	http.HandleFunc("/api/context", server_data.handleContexts)
	http.HandleFunc("/api/context/{id}", server_data.handleContext)
	http.HandleFunc("/api/context/{id}/systemprompt", server_data.handleSetSystemPrompt)
	http.HandleFunc("/api/context/{id}/setmodel", server_data.handleSetModel)
	http.HandleFunc("/status", server_data.handleStatus)

	var err error
	if secure {
		err = http.ListenAndServeTLS(fmt.Sprintf(":%d", port), "cert.pem", "key.pem", nil)
	} else {
		err = http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
	}

	if err != nil {
		println(fmt.Sprintf("\nerr: %v", err))
	}
}

type promptRequest struct {
	Prompt      string `json:"prompt"`
	ContextName string `json:"contextName"`
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type SetModelRequest struct {
	Model string `json:"model"`
}

func parseModelRequest(r *http.Request) (SetModelRequest, error) {
	var req SetModelRequest

	if r.Header.Get("Content-Type") != "application/json" {
		return req, fmt.Errorf("Content-Type must be application/json")
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		logger.Screen(fmt.Sprintf("\nerror reading request body\n: %v", err), color.RGB(250, 150, 150))
		return req, fmt.Errorf("error reading request body: %v", err)
	}
	defer r.Body.Close()

	err = json.Unmarshal(body, &req)
	if err != nil {
		logger.Screen(fmt.Sprintf("\nerror parsing JSON: %v\n", err), color.RGB(250, 150, 150))
		return req, fmt.Errorf("error parsing JSON: %v", err)
	}

	return req, nil
}

func parseSetSystemPromptRequest(r *http.Request) (SetSystemPromptRequest, error) {
	var req SetSystemPromptRequest

	if r.Header.Get("Content-Type") != "application/json" {
		return req, fmt.Errorf("Content-Type must be application/json")
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		logger.Screen(fmt.Sprintf("\nerror reading request body\n: %v", err), color.RGB(250, 150, 150))
		return req, fmt.Errorf("error reading request body: %v", err)
	}
	defer r.Body.Close()

	err = json.Unmarshal(body, &req)
	if err != nil {
		logger.Screen(fmt.Sprintf("\nerror parsing JSON: %v\n", err), color.RGB(250, 150, 150))
		return req, fmt.Errorf("error parsing JSON: %v", err)
	}

	return req, nil
}

func parseLoginRequest(r *http.Request) (loginRequest, error) {
	logger.Screen("Recieved login through http", color.RGB(150, 150, 150))
	var req loginRequest

	if r.Header.Get("Content-Type") != "application/json" {
		return req, fmt.Errorf("Content-Type must be application/json")
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		logger.Screen(fmt.Sprintf("\nerror reading request body\n: %v", err), color.RGB(250, 150, 150))
		return req, fmt.Errorf("error reading request body: %v", err)
	}
	defer r.Body.Close()

	err = json.Unmarshal(body, &req)
	if err != nil {
		logger.Screen(fmt.Sprintf("\nerror parsing JSON: %v\n", err), color.RGB(250, 150, 150))
		return req, fmt.Errorf("error parsing JSON: %v", err)
	}

	return req, nil
}

func parsePromptRequest(r *http.Request) (promptRequest, error) {
	logger.Screen("Recieved prompt through http", color.RGB(150, 150, 150))
	var req promptRequest

	if r.Header.Get("Content-Type") != "application/json" {
		return req, fmt.Errorf("Content-Type must be application/json")
	}

	body, err := io.ReadAll(r.Body)
	logger.Screen(fmt.Sprintf("\nReceived body %s\n", body), color.RGB(150, 150, 150))

	if err != nil {
		logger.Screen(fmt.Sprintf("\nerror reading request body\n: %v", err), color.RGB(250, 150, 150))
		return req, fmt.Errorf("error reading request body: %v", err)
	}
	defer r.Body.Close()

	err = json.Unmarshal(body, &req)
	if err != nil {
		logger.Screen(fmt.Sprintf("\nerror parsing JSON: %v\n", err), color.RGB(250, 150, 150))
		return req, fmt.Errorf("error parsing JSON: %v", err)
	}

	if req.Prompt == "" {
		return req, fmt.Errorf("no prompt in request body")
	}

	return req, nil
}

type LoginResponse struct {
	Token string `json:"token"`
}

func GetUsernameFromToken(tokenString string) (string, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return secretKey, nil
	})

	if err != nil {
		return "", err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		return claims["username"].(string), nil
	}

	return "", fmt.Errorf("invalid token")
}

func (server_data *server_data) handleLogin(w http.ResponseWriter, r *http.Request) {
	enableCors(w)

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	req, err := parseLoginRequest(r)
	if err != nil {
		http.Error(w, "Bad input", http.StatusBadRequest)
		return
	}
	token, err := CreateToken(req.Username)
	if err != nil {
		http.Error(w, "Could not generate token", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(LoginResponse{
		Token: token,
	})
}

var secretKey = []byte("my-test-secret-key-change-in-production")

func CreateToken(username string) (string, error) {
	claims := jwt.MapClaims{
		"username": username,
		"exp":      time.Now().Add(7 * 24 * time.Hour).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secretKey)
}

type ContextsResponse struct {
	Contexts []data.Context `json:"contexts"`
}

func (server_data *server_data) handleContexts(w http.ResponseWriter, r *http.Request) {
	enableCors(w)
	logger.Screen("hit the handle contexts endpoint handler", color.RGB(150, 150, 150))

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	username, err := authenticate(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	repository, _ := server_data.responseHandler.Repository.(*data.MultiUserContext)
	repository.SetCurrentDb(username)

	contexts, err := repository.GetAllContexts()
	if err != nil {
		logger.Debug.Printf("error when fetching contexts %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(ContextsResponse{
		Contexts: contexts,
	})
}

type ContextResponse struct {
	Name           string         `json:"name"`
	Id             string         `json:"id"`
	Created        time.Time      `json:"created"`
	History        []data.History `json:"history"`
	SystemPrompt   string         `json:"systemPrompt"`
	PreferredModel string         `json:"preferredModel"`
}

type SetSystemPromptRequest struct {
	SystemPrompt string `json:"systemPrompt"`
}

func (server_data *server_data) handleSetModel(w http.ResponseWriter, r *http.Request) {
	logger.Screen(fmt.Sprintf("\nhit set model with method %s", r.Method), color.RGB(150, 150, 150))
	enableCors(w)

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	} else if r.Method != "POST" {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	id := r.PathValue("id")
	logger.Debug.Printf("Called for context at id: {%s}", id)
	username, err := authenticate(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	repository, _ := server_data.responseHandler.Repository.(*data.MultiUserContext)
	repository.SetCurrentDb(username)

	intId, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		logger.Screen(fmt.Sprintf("\nFailed to convert id to int: %s", id), color.RGB(250, 150, 150))
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	req, err := parseModelRequest(r)
	if err != nil {
		logger.Screen(fmt.Sprintf("\nFailed to parse body: %s", err), color.RGB(250, 150, 150))
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = repository.UpdatePreferredModel(intId, req.Model)
	if err != nil {
		logger.Debug.Printf("error while setting system prompt %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (server_data *server_data) handleSetSystemPrompt(w http.ResponseWriter, r *http.Request) {
	logger.Screen(fmt.Sprintf("\nhit set system prompt with method %s", r.Method), color.RGB(150, 150, 150))
	enableCors(w)

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	} else if r.Method != "POST" {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	id := r.PathValue("id")
	logger.Screen(fmt.Sprintf("\nCalled for context at id: {%s}", id), color.RGB(150, 150, 150))
	logger.Debug.Printf("Called for context at id: {%s}", id)
	username, err := authenticate(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	repository, _ := server_data.responseHandler.Repository.(*data.MultiUserContext)
	repository.SetCurrentDb(username)

	intId, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		logger.Screen(fmt.Sprintf("\nFailed to convert id to int: %s", id), color.RGB(250, 150, 150))
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	req, err := parseSetSystemPromptRequest(r)
	if err != nil {
		logger.Screen(fmt.Sprintf("\nFailed to parse body: %s", err), color.RGB(250, 150, 150))
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = repository.UpdateSystemPrompt(intId, req.SystemPrompt)
	if err != nil {
		logger.Debug.Printf("error while setting system prompt %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (server_data *server_data) handleContext(w http.ResponseWriter, r *http.Request) {
	enableCors(w)

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	} else if r.Method == "GET" {
		id := r.PathValue("id")
		logger.Screen(fmt.Sprintf("Called for context at id: {%s}", id), color.RGB(150, 150, 150))
		logger.Debug.Printf("Called for context at id: {%s}", id)
		username, err := authenticate(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		repository, _ := server_data.responseHandler.Repository.(*data.MultiUserContext)
		repository.SetCurrentDb(username)

		intId, err := strconv.ParseInt(id, 10, 64)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		context, err := repository.GetContextById(intId)
		if err != nil {
			logger.Debug.Printf("error when fetching context %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		history, err := repository.GetHistoryByContextId(intId, 1000)
		if err != nil {
			logger.Debug.Printf("error when fetching context %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(ContextResponse{
			Id:             id,
			Name:           context.Name,
			SystemPrompt:   context.SystemPrompt,
			Created:        context.Created,
			History:        history,
			PreferredModel: context.PreferredModel,
		})
		return

	} else {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}

func authenticate(r *http.Request) (string, error) {
	authenticationHeader := r.Header.Get("Authorization")
	if authenticationHeader == "" {
		logger.Screen("Recieved empty authorization header", color.RGB(150, 150, 150))
		return "", fmt.Errorf("Did not receive authentication header")
	}
	possibleToken, _ := strings.CutPrefix(authenticationHeader, "Bearer ")
	username, err := GetUsernameFromToken(possibleToken)
	if err != nil {
		logger.Screen("Could not get username from token", color.RGB(150, 150, 150))
		return "", fmt.Errorf("incorrectly formated authentication header")
	}
	return username, nil
}

func enableCors(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Access-Control-Allow-Headers, Authorization, X-Requested-With")
}

// getModelForQuery returns the appropriate model based on context preferences
func getModelForQuery(
	requestedModel string,
	context *data.Context,
	responseHandler models.ResponseHandler,
	historyRepository data.HistoryRepository,
	streamMode bool,
) (models.Model, string) {

	modelToUse := requestedModel

	if modelToUse == "" && context != nil && context.PreferredModel != "" {
		modelToUse = context.PreferredModel
	}

	if modelToUse == "" {
		modelToUse = "claude"
	}

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
	case "qwen3":
		model = ollama_model.NewOllamaModel(responseHandler, "")
	case "opus":
		model = &claude_model.ClaudeModel{
			UseStreaming:      streamMode,
			HistoryRepository: historyRepository,
			ResponseHandler:   responseHandler,
			ModelVersion:      "opus",
		}
	case "sonnet":
		model = &claude_model.ClaudeModel{
			UseStreaming:      streamMode,
			HistoryRepository: historyRepository,
			ResponseHandler:   responseHandler,
			ModelVersion:      "sonnet",
		}
	case "haiku":
		model = &claude_model.ClaudeModel{
			UseStreaming:      streamMode,
			HistoryRepository: historyRepository,
			ResponseHandler:   responseHandler,
			ModelVersion:      "haiku",
		}
	case "claude":
		fallthrough
	default:
		model = &claude_model.ClaudeModel{
			UseStreaming:      streamMode,
			HistoryRepository: historyRepository,
			ResponseHandler:   responseHandler,
		}
		modelToUse = "claude"
	}

	return model, modelToUse
}

func (server_data *server_data) handlePrompt(w http.ResponseWriter, r *http.Request) {
	enableCors(w)
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	username, err := authenticate(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	req, err := parsePromptRequest(r)
	if err != nil {
		http.Error(w, "Bad input", http.StatusBadRequest)
		return
	}

	repository, _ := server_data.responseHandler.Repository.(*data.MultiUserContext)

	logger.Debug.Printf("Handling prompt request: %v", req)

	repository.SetCurrentDb(username)

	logger.Screen(fmt.Sprintf("\nCurrent user in repository %v\n", repository.User), color.RGB(150, 150, 150))

	if req.ContextName == "" {
		logger.Screen("Naming context...", color.RGB(150, 150, 150))
		logger.Debug.Println("Sending Haiku request to name context")
		toolHandler := tools.ToolResponseHandler{}
		toolHandler.Init()

		model := &claude_model.ClaudeModel{ResponseHandler: &toolHandler, ModelVersion: "Haiku"}

		prompt := fmt.Sprintf("Create a short name for this prompt so that I can store it with a name in a database. Maximum 100 characters but try to keep it short. ONLY EVER answer with the name and nothing else!!!! here's the prompt to name the context for: %s", req.Prompt)

		services.AwaitedQuery(prompt, model, repository, 0, &data.Context{
			Name:    "Create name for context",
			Id:      999,
			History: []data.History{},
		}, &models.PayloadModifiers{}, "haiku")

		response := <-toolHandler.ResponseChannel

		logger.Debug.Printf("naming reponse: %s", response)
		logger.Screen(fmt.Sprintf("naming reponse: %s", response), color.RGB(150, 150, 150))
		req.ContextName = response
	}

	context, _ := repository.GetContextByName(req.ContextName)
	if context == nil {
		new_context := data.Context{Name: req.ContextName}
		id, err := repository.InsertContext(new_context)
		if err != nil {
			log.Println(fmt.Sprintf("Could not create a new context with name %s for user %s, %s", req.ContextName, username, err))
		}

		context = &new_context
		context.Id = id
	}

	w.Header().Set("Connection", "Keep-Alive")
	w.Header().Set("Transfer-Encoding", "chunked")
	w.WriteHeader(http.StatusOK)

	server_data.responseHandler.SetResponseWriter(w)

	selectedModel, modelName := getModelForQuery("", context, server_data.responseHandler, repository, server_data.streaming)

	if server_data.streaming {
		services.StreamedQuery(req.Prompt, selectedModel, server_data.responseHandler.Repository, 5, context, &models.PayloadModifiers{}, modelName)
	} else {
		services.AwaitedQuery(req.Prompt, selectedModel, server_data.responseHandler.Repository, 5, context, &models.PayloadModifiers{}, modelName)
	}
}

type HttpResponseHandler struct {
	responseWriter http.ResponseWriter
	Repository     data.HistoryRepository
}

func (httpResponseHandler *HttpResponseHandler) SetResponseWriter(writer http.ResponseWriter) {
	httpResponseHandler.responseWriter = writer
}

func (httpResponseHandler *HttpResponseHandler) RecievedText(text string, useColor *string) {
	fmt.Fprint(httpResponseHandler.responseWriter, text)
	httpResponseHandler.responseWriter.(http.Flusher).Flush()
}

func (httpResponseHandler *HttpResponseHandler) FinalText(contextId int64, prompt string, response string, responseContent string, toolResults string, modelName string) {
	repository, ok := httpResponseHandler.Repository.(*data.MultiUserContext)

	if !ok {
		log.Panic("This needs to be called with a repository that supplies user")
	}

	history := data.History{
		ContextId:       contextId,
		Prompt:          prompt,
		Response:        response,
		Abbreviation:    "",
		TokenCount:      0,
		UserId:          int64(repository.User.Id),
		ResponseContent: responseContent,
		Model:           modelName,
	}

	_, err := httpResponseHandler.Repository.InsertHistory(history)
	if err != nil {
		println(fmt.Sprintf("Error while trying to save history: %s", err))
	}
	fmt.Fprintf(httpResponseHandler.responseWriter, response)
}

func (server_data *server_data) handleStatus(w http.ResponseWriter, r *http.Request) {
	enableCors(w)
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "OK")
}

func (server_data *server_data) handleRoot(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, fmt.Sprintf("%v", server_data.model))
}
