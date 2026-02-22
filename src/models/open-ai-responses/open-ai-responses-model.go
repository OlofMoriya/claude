package open_ai_responses

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image/png"
	"net/http"
	"os"
	commontypes "owl/common_types"
	"owl/data"
	"owl/logger"
	"owl/models"
	"strings"
	"time"

	"github.com/skratchdot/open-golang/open"
)

var MODELNAME = "open_ai_responses"

type OpenAiResponseModel struct {
	ResponseHandler   models.ResponseHandler
	prompt            string
	accumulatedAnswer string
	contextId         int64
}

func (model *OpenAiResponseModel) CreateRequest(context *data.Context, prompt string, streaming bool, history []data.History, modifiers *commontypes.PayloadModifiers) *http.Request {
	payload := createResponsePayload(prompt, streaming, history, modifiers.Image)
	model.prompt = prompt
	model.accumulatedAnswer = ""
	model.contextId = context.Id
	return createRequest(payload)
}

func createRequest(payload RequestPayload) *http.Request {
	apiKey, ok := os.LookupEnv("OPENAI_API_KEY")
	if !ok {
		panic(fmt.Errorf("Could not fetch api key"))
	}

	jsonpayload, err := json.Marshal(payload)
	logger.Debug.Println("Will send payload")
	logger.Debug.Println(jsonpayload)
	if err != nil {
		panic("failed to marshal payload")
	}

	url := "https://api.openai.com/v1/responses"
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonpayload))
	if err != nil {
		panic(fmt.Errorf("failed to create request: %v", err))
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))

	return req
}

func createResponsePayload(prompt string, streaming bool, history []data.History, b bool) RequestPayload {
	request := RequestPayload{
		Model: "gpt-4.1",
		Input: prompt,
		Tools: []Tool{Tool{Type: "image_generation"}},
	}

	logger.Debug.Println("Will send payload")
	logger.Debug.Printf("request %v", request)
	return request
}

func (model *OpenAiResponseModel) HandleStreamedLine(line []byte) {
	responseLine := string(line)

	// fmt.Printf("\n\n%v\n", responseLine)
	if strings.HasPrefix(responseLine, "data: ") {
		var apiResponse ChatCompletionChunk
		data, _ := strings.CutPrefix(responseLine, "data: ")

		logger.Debug.Printf("json")
		logger.Debug.Printf("%v", apiResponse)
		if err := json.Unmarshal([]byte(data), &apiResponse); err != nil {
			// fmt.Printf("Error unmarshalling response: %v\n %s", err, line)
		}

		if len(apiResponse.Choices) > 0 {
			choice := apiResponse.Choices[0]

			model.accumulatedAnswer = model.accumulatedAnswer + choice.Delta.Content
			model.ResponseHandler.RecievedText(choice.Delta.Content, nil)

			if choice.FinishReason != nil {
				fmt.Println(*&choice.FinishReason)
				model.ResponseHandler.FinalText(model.contextId, model.prompt, model.accumulatedAnswer, "", "", MODELNAME)
			}
		}
	}
}

func (model *OpenAiResponseModel) HandleBodyBytes(byte_list []byte) {
	var apiResponse Response
	if err := json.Unmarshal(byte_list, &apiResponse); err != nil {
		// Handle error, maybe return or log
		println(fmt.Sprintf("Error unmarshalling response body: %v\n", err))
	}

	text := ""

	for _, output := range apiResponse.Output {
		logger.Debug.Printf("%s", output)
		switch v := output.(type) {
		case ImageGenerationCall:
			unbased, err := base64.StdEncoding.DecodeString(v.Result)
			if err != nil {
				panic("Cannot decode b64")
			}

			r := bytes.NewReader(unbased)
			im, err := png.Decode(r)
			if err != nil {
				panic("Bad png")
			}

			filename := fmt.Sprintf("/Users/olofmoriya/.owl/img/%d-%s.png", model.contextId, time.Now().Format("2006-01-02:15:04:05"))
			f, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE, 0777)
			if err != nil {
				panic("Cannot open file")
			}

			png.Encode(f, im)

			err = open.Start(filename)
			if err != nil {
				fmt.Printf("Failed to open image: %v\n", err)
			}

			text += fmt.Sprintf("\n![Image](%s)\n", filename)

		case Message:
			for i, content := range v.Content {
				text += content.Text
				if i < (len(v.Content) - 1) {
					text += "\n\n"
				}
			}
		}
	}

	logger.Debug.Printf("Final text from responses: %s", text)
	model.ResponseHandler.FinalText(model.contextId, model.prompt, text, "", "", MODELNAME)
}

func (model *OpenAiResponseModel) SetResponseHandler(responseHandler models.ResponseHandler) {
	model.ResponseHandler = responseHandler
}
