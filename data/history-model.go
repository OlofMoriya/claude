package data

import "time"

type Context struct {
	Id      int64     `json:"id"`
	Name    string    `json:"name"`
	History []History `json:"history"`
	UserId  int64     `json:"userId"`
	Created time.Time `json:"created"`
}

type History struct {
	Id           int64     `json:"id"`
	ContextId    int64     `json:"context_id"`
	Prompt       string    `json:"prompt"`
	Response     string    `json:"response"`
	Abbreviation string    `json:"abreviation"`
	TokenCount   int       `json:"token_count"`
	UserId       int64     `json:"userId"`
	Created      time.Time `json:"created"`
}
