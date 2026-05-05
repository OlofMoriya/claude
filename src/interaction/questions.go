package interaction

import commontypes "owl/common_types"

type QuestionPrompt struct {
	Request      commontypes.QuestionBatchRequest
	ResponseChan chan QuestionPromptResult
}

type QuestionPromptResult struct {
	Response *commontypes.QuestionBatchResponse
	Err      error
}

var QuestionPromptChan chan QuestionPrompt
