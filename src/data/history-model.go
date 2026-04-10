package data

import (
	"time"
)

type Context struct {
	Id             int64     `json:"id"`
	Name           string    `json:"name"`
	History        []History `json:"history"`
	UserId         int64     `json:"userId"`
	Created        time.Time `json:"created"`
	SystemPrompt   string    `json:"system_prompt"`
	PreferredModel string    `json:"preferred_model"`
	Archived       bool      `json:"archived"`
}

type History struct {
	Id               int64     `json:"id"`
	ContextId        int64     `json:"context_id"`
	Prompt           string    `json:"prompt"`
	Response         string    `json:"response"`
	ResponseContent  string    `json:"response_content"`
	Abbreviation     string    `json:"abreviation"`
	TokenCount       int       `json:"token_count"`
	PromptTokens     int       `json:"prompt_tokens"`
	CompletionTokens int       `json:"completion_tokens"`
	CacheReadTokens  int       `json:"cache_read_tokens"`
	CacheWriteTokens int       `json:"cache_write_tokens"`
	UserId           int64     `json:"userId"`
	Created          string    `json:"created"`
	ToolResults      string    `json:"tool_results"`
	Model            string    `json:"model"`
	Archived         bool      `json:"archived"`
	ToolUse          []ToolUse `json:"toolUse"`
}

type ToolUse struct {
	Id         string     `json:"id"`
	Name       string     `json:"name"`
	Input      string     `json:"input"`
	CallerType string     `json:"callerType"`
	Result     ToolResult `json:"result"`
}

type ToolResult struct {
	ToolUseId string `json:"toolUseId"`
	Content   string `json:"content"`
	Success   bool   `json:"success"`
}
