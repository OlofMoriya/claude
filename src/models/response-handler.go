package models

type ResponseHandler interface {
	RecievedText(text string, color *string)
	FinalText(contextId int64, prompt string, response string, responseContent string)
	// func recievedImage(encoded string)
}
