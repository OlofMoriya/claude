package interaction

type FileDisplayPrompt struct {
	Path         string
	Title        string
	Content      string
	StartLine    int
	EndLine      int
	ResponseChan chan FileDisplayResult
}

type FileDisplayResult struct {
	Err error
}

var FileDisplayPromptChan chan FileDisplayPrompt
