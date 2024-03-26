package server

import (
	data "claude/data"
	models "claude/models"
	"claude/services"
	"fmt"
	"net/http"
)

type server_data struct {
	model           models.Model
	responseHandler *HttpResponseHandler
	streaming       bool
}

func Run(secure bool, port int, responseHandler *HttpResponseHandler, model models.Model, streaming bool) {

	server_data := server_data{
		model:           model,
		responseHandler: responseHandler,
		streaming:       streaming,
	}

	http.HandleFunc("/", server_data.handleRoot)
	http.HandleFunc("/prompt", server_data.handlePrompt)

	var err error
	if secure {
		err = http.ListenAndServeTLS(fmt.Sprintf(":%d", port), "cert.pem", "key.pem", nil)
	} else {
		err = http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
	}

	if err != nil {
		fmt.Printf("\nerr: %v", err)
	}
}

func (server_data *server_data) handlePrompt(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		// Parse the request body
		err := r.ParseForm()
		if err != nil {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		// Access the POST data
		prompt := r.FormValue("prompt")
		context_name := r.FormValue("context_name")
		user_name := r.FormValue("user")

		user := data.User{Id: user_name, Name: user_name}

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

		w.Header().Set("Connection", "Keep-Alive")
		w.Header().Set("Transfer-Encoding", "chunked")
		w.WriteHeader(http.StatusOK)

		//trigger awaited query
		server_data.responseHandler.SetResponseWriter(w)
		if server_data.streaming {
			services.StreamedQuery(prompt, server_data.model, user, 0, context_id)
		} else {
			services.AwaitedQuery(prompt, server_data.model, user, 0, context_id)
		}

		// Send a response
		// fmt.Fprintf(w, "POST request processed successfully")
	} else {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}

}

type HttpResponseHandler struct {
	responseWriter http.ResponseWriter
}

func (httpResponseHandler *HttpResponseHandler) SetResponseWriter(writer http.ResponseWriter) {
	httpResponseHandler.responseWriter = writer
}

func (httpResponseHandler *HttpResponseHandler) RecievedText(text string) {
	// fmt.Println(text)
	fmt.Fprintf(httpResponseHandler.responseWriter, text)
	httpResponseHandler.responseWriter.(http.Flusher).Flush()
}
func (httpResponseHandler *HttpResponseHandler) FinalText(contextId int64, prompt string, response string) {
	fmt.Fprintf(httpResponseHandler.responseWriter, response)
}

func (server_data *server_data) handleRoot(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, fmt.Sprintf("%v", server_data.model))
}
