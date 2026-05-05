package commontypes

type QuestionOption struct {
	Label string `json:"label"`
}

type QuestionItem struct {
	ID            string           `json:"id"`
	Question      string           `json:"question"`
	Options       []QuestionOption `json:"options"`
	AllowCustom   bool             `json:"allow_custom"`
	AllowMultiple bool             `json:"allow_multiple"`
	Required      bool             `json:"required"`
}

type QuestionBatchRequest struct {
	Title     string         `json:"title"`
	Questions []QuestionItem `json:"questions"`
}

type QuestionAnswer struct {
	ID                    string   `json:"id"`
	SelectedOptionIndex   *int     `json:"selected_option_index,omitempty"`
	SelectedOptionLabel   string   `json:"selected_option_label,omitempty"`
	SelectedOptionIndexes []int    `json:"selected_option_indexes,omitempty"`
	SelectedOptionLabels  []string `json:"selected_option_labels,omitempty"`
	CustomAnswer          string   `json:"custom_answer,omitempty"`
	FinalAnswer           string   `json:"final_answer"`
}

type QuestionBatchResponse struct {
	Answers []QuestionAnswer `json:"answers"`
}
