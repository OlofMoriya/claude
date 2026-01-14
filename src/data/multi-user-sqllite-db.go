package data

import (
	"fmt"
	"owl/logger"

	"github.com/fatih/color"
	_ "github.com/mattn/go-sqlite3"
)

type MultiUserContext struct {
	User User
}

func (mu_context *MultiUserContext) SetCurrentDb(username string) {
	logger.Debug.Printf("\nSetting multi user context to username: %s", username)
	logger.Screen(fmt.Sprintf("\nSetting multi user context to username: %s", username), color.RGB(150, 150, 150))
	user := User{Name: &username}
	logger.Screen(fmt.Sprintf("\ncreating user: %v", user), color.RGB(150, 150, 150))

	mu_context.User = user
}

func (mu_context *MultiUserContext) InsertContext(context Context) (int64, error) {
	return mu_context.User.InsertContext(context)
}

func (mu_context *MultiUserContext) GetContextById(contextId int64) (Context, error) {
	return mu_context.User.GetContextById(contextId)
}

func (mu_context *MultiUserContext) InsertHistory(history History) (int64, error) {
	return mu_context.User.InsertHistory(history)
}

func (mu_context *MultiUserContext) GetHistoryByContextId(contextId int64, maxCount int) ([]History, error) {
	return mu_context.User.GetHistoryByContextId(contextId, maxCount)
}

func (mu_context *MultiUserContext) GetContextByName(name string) (*Context, error) {

	logger.Screen(fmt.Sprintf(" Getting context with name %s", name), color.RGB(150, 150, 150))
	logger.Screen(fmt.Sprintf("User is set to %v", mu_context.User), color.RGB(150, 150, 150))
	return mu_context.User.GetContextByName(name)
}

func (mu_context *MultiUserContext) GetAllContexts() ([]Context, error) {
	return mu_context.User.GetAllContexts()
}

func (mu_context *MultiUserContext) DeleteContext(contextId int64) (int64, error) {
	return mu_context.User.DeleteContext(contextId)
}

func (mu_context *MultiUserContext) DeleteHistory(historyId int64) (int64, error) {
	return mu_context.User.DeleteHistory(historyId)
}

func (mu_context *MultiUserContext) UpdateSystemPrompt(contextId int64, systemPrompt string) error {
	return mu_context.User.UpdateSystemPrompt(contextId, systemPrompt)
}
