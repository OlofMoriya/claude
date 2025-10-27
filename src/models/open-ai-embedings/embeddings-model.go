package open_ai_embedings

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"owl/data"
	models "owl/models"
)

type OpenAiEmbeddingsModel struct {
	ResponseHandler models.ResponseHandler
	prompt          string
}

func (model *OpenAiEmbeddingsModel) SetResponseHandler(responseHandler models.ResponseHandler) {
	model.ResponseHandler = responseHandler

}

func (model *OpenAiEmbeddingsModel) CreateRequest(context *data.Context, prompt string, streaming bool, history []data.History, modifiers *models.PayloadModifiers) *http.Request {
	payload := createPayload(prompt, streaming, history)
	model.prompt = prompt
	return createRequest(payload, history, modifiers.Image)
}

func createPayload(prompt string, streamed bool, history []data.History) Payload {
	payload := Payload{
		Model:          "text-embedding-ada-002",
		Input:          prompt,
		EncodingFormat: "float",
		// Dimensions:     69,
	}

	return payload
}

func createRequest(payload Payload, history []data.History, image bool) *http.Request {
	//use gcloud to fetch the token
	apiKey, ok := os.LookupEnv("OPENAI_API_KEY")
	if !ok {
		panic(fmt.Errorf("Could not fetch api key"))
	}

	jsonpayload, err := json.Marshal(payload)
	if err != nil {
		panic("failed to marshal payload")
	}

	url := "https://api.openai.com/v1/embeddings"
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonpayload))
	if err != nil {
		panic(fmt.Errorf("failed to create request: %v", err))
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))

	return req
}

func (model *OpenAiEmbeddingsModel) HandleStreamedLine(line []byte) {
}

func (model *OpenAiEmbeddingsModel) HandleBodyBytes(bytes []byte) {
	var apiResponse Response
	if err := json.Unmarshal(bytes, &apiResponse); err != nil {
		// Handle error, maybe return or log
		println(fmt.Sprintf("Error unmarshalling response body: %v\n", err))
	}

	//TODO: Why is data indexed
	if len(apiResponse.Data) > 1 {
		println("Multiple data in Data array. Only handling 1 at the moment")
	}

	embeddingsArray, err := json.Marshal(apiResponse.Data[0].Embedding)
	if err != nil {
		println(err)
	}

	model.ResponseHandler.FinalText(0, model.prompt, string(embeddingsArray), "")
}
