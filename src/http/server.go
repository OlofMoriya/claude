package server

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	data "owl/data"
	models "owl/models"
	"owl/services"
)

type server_data struct {
	model               models.Model
	responseHandler     *HttpResponseHandler
	streaming           bool
	db_connectionString string
}

func Run(secure bool, port int, responseHandler *HttpResponseHandler, model models.Model, streaming bool, dbConnectionString string) {

	server_data := server_data{
		model:           model,
		responseHandler: responseHandler,
		streaming:       streaming,
	}

	http.HandleFunc("/", server_data.handleRoot)
	http.HandleFunc("/prompt", server_data.handlePrompt)
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

type owlRequest struct {
	Prompt      string  `json:"prompt"`
	ContextName string  `json:"contextName"`
	User        string  `json:"user"`
	SlackId     *string `json:"slackId"`
}

func parseOwlRequest(r *http.Request) (owlRequest, error) {
	var req owlRequest

	// Check if the Content-Type is application/json
	if r.Header.Get("Content-Type") != "application/json" {
		return req, fmt.Errorf("Content-Type must be application/json")
	}

	// Read the body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return req, fmt.Errorf("error reading request body: %v", err)
	}
	defer r.Body.Close()

	// Unmarshal the JSON into the owlRequest struct
	err = json.Unmarshal(body, &req)
	if err != nil {
		return req, fmt.Errorf("error parsing JSON: %v", err)
	}

	return req, nil
}

// Helper to get or create user. Should be moved somewhere else?
func getUser(repository *data.PostgresHistoryRepository, req owlRequest) (data.User, error) {
	emailUser, err := repository.GetUserByEmail(req.User)
	if err != nil {
		log.Panic("error trying to find user by email", err)
	}

	if emailUser != nil {
		return *emailUser, nil
	}

	if req.SlackId != nil {
		slackUser, err := repository.GetUserBySlackId(*req.SlackId)
		if err != nil {
			log.Panic("error trying to find user by email", err)
		}
		if slackUser != nil {
			return *slackUser, nil
		}
	}

	newUser := data.User{Name: &req.User, SlackId: req.SlackId, Email: &req.User}
	id, err := repository.CreateUser(newUser)
	if err != nil {
		return newUser, err
	}

	newUser.Id = id

	return newUser, nil
}

func (server_data *server_data) handlePrompt(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		// Parse the request body

		req, err := parseOwlRequest(r)
		if err != nil {
			http.Error(w, "Bad input", http.StatusBadRequest)
		}

		repository, ok := server_data.responseHandler.Repository.(*data.PostgresHistoryRepository)
		if !ok {
			log.Fatal("Repository is not of type *PostgresHistoryRepository")
		}

		user, err := getUser(repository, req)
		if err != nil {
			log.Panic("could not get or create user", err)
		}
		repository.User = user

		context, _ := repository.GetContextByName(req.ContextName)
		var context_id int64
		if context == nil {
			new_context := data.Context{Name: req.ContextName, UserId: int64(user.Id)}
			id, err := repository.InsertContext(new_context)

			if err != nil {
				log.Println(fmt.Sprintf("Could not create a new context with name %s, %s", req.ContextName, err))
			}
			context_id = id
		} else {
			context_id = context.Id
		}

		w.Header().Set("Connection", "Keep-Alive")
		w.Header().Set("Transfer-Encoding", "chunked")
		w.WriteHeader(http.StatusOK)

		//trigger awaited query
		server_data.responseHandler.SetResponseWriter(w)
		if server_data.streaming {
			services.StreamedQuery(req.Prompt, server_data.model, server_data.responseHandler.Repository, 5, context_id)
		} else {
			services.AwaitedQuery(req.Prompt, server_data.model, server_data.responseHandler.Repository, 5, context_id)
		}

		// Send a response
		// fmt.Fprintf(w, "POST request processed successfully")
	} else {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}

}

type HttpResponseHandler struct {
	responseWriter http.ResponseWriter
	Repository     data.HistoryRepository
}

func (httpResponseHandler *HttpResponseHandler) SetResponseWriter(writer http.ResponseWriter) {
	httpResponseHandler.responseWriter = writer
}

func (httpResponseHandler *HttpResponseHandler) RecievedText(text string) {
	fmt.Fprintf(httpResponseHandler.responseWriter, text)
	httpResponseHandler.responseWriter.(http.Flusher).Flush()
}
func (httpResponseHandler *HttpResponseHandler) FinalText(contextId int64, prompt string, response string) {

	repository, ok := httpResponseHandler.Repository.(*data.PostgresHistoryRepository)

	if !ok {
		log.Panic("This needs to be called with a repository that supplies user")
	}

	history := data.History{
		ContextId:    contextId,
		Prompt:       prompt,
		Response:     response,
		Abbreviation: "",
		TokenCount:   0,
		UserId:       int64(repository.User.Id),
		//TODO abreviation
		//TODO tokencount
	}

	_, err := httpResponseHandler.Repository.InsertHistory(history)
	if err != nil {
		println(fmt.Sprintf("Error while trying to save history: %s", err))
	}
	fmt.Fprintf(httpResponseHandler.responseWriter, response)
}

func (server_data *server_data) handleStatus(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (server_data *server_data) handleRoot(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, fmt.Sprintf("%v", server_data.model))
}
