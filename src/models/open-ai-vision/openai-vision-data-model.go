package openai_vision_model

type Payload struct {
	Model     string    `json:"model"`
	Messages  []Message `json:"messages"`
	MaxTokens int       `json:"max_tokens"`
	Stream    bool      `json:"stream"`
}

type Message struct {
	Role    string    `json:"role"`
	Content []Content `json:"content"`
}

type Content struct {
	Type     string `json:"type"`
	Text     string `json:"text,omitempty"`
	ImageURL Image  `json:"image_url,omitempty"`
}

type Image struct {
	URL string `json:"url"`
}

type ApiResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Usage   struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
	Choices []struct {
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishDetails struct {
			Type string `json:"type"`
			Stop string `json:"stop"`
		} `json:"finish_details"`
		Index int `json:"index"`
	} `json:"choices"`
}

//StreamData response example for stop reason
// {
//   "id": "chatcmpl-97f5iA568rrc7H7oPVEuFbXJLoI6f",
//   "object": "chat.completion.chunk",
//   "created": 1711613278,
//   "model": "gpt-4-1106-vision-preview",
//   "system_fingerprint": null,
//   "choices": [
//     {
//       "index": 0,
//       "delta": {},
//       "logprobs": null,
//       "finish_reason": "stop"
//     }
//   ]
// }

type StreamData struct {
	Choices []Choice `json:"choices"`
}

type Choice struct {
	Delta        Delta   `json:"delta"`
	FinishReason *string `json:"finish_reason"`
}

type Delta struct {
	Content string `json:"content"`
}
