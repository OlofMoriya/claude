package open_ai_embedings

type Payload struct {
	Input          string `json:"input"`
	Model          string `json:"model"`
	EncodingFormat string `json:"encoding_format"`
	// Dimensions     int    `json:"dimensions"`
}

type Response struct {
	Object string      `json:"object"`
	Data   []Embedding `json:"data"`
	Model  string      `json:"model"`
	Usage  Usage       `json:"usage"`
}

type Usage struct {
	PromptTokens int `json:"promptTokens"`
	TotalTokens  int `json:"totalTokens"`
}

type Embedding struct {
	Object    string    `json:"object"`
	Embedding []float32 `json:"embedding"`
	Index     int       `json:"index"`
}
