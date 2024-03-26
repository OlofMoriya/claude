package data

type Context struct {
	Id      int64     `json:"id"`
	Name    string    `json:"name"`
	History []History `json:"history"`
}

type History struct {
	Id          int64  `json:"id"`
	ContextId   int64  `json:"context_id"`
	Prompt      string `json:"prompt"`
	Response    string `json:"response"`
	Abreviation string `json:"abreviation"`
	TokenCount  int    `json:"token_count"`
}
