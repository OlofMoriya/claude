package commontypes

type ToolResponse struct {
	Id              string
	Response        string
	ResponseMessage interface{}
}

type PayloadModifiers struct {
	ToolResponses []ToolResponse
	Pdf           string
	Web           bool
	Image         bool
}
