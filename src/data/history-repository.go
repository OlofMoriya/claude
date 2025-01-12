package data

type HistoryRepository interface {
	GetContextById(contextId int64) (Context, error)
	InsertHistory(history History) (int64, error)
	InsertContext(context Context) (int64, error)
	GetHistoryByContextId(contextId int64, maxCount int) ([]History, error)
	GetContextByName(name string) (*Context, error)
	GetAllContexts() ([]Context, error)
	DeleteContext(contextId int64) (int64, error)
	DeleteHistory(historyId int64) (int64, error)
}
