package testhelpers

import "owl/data"

type MockHistoryRepository struct {
	Histories map[int64][]data.History
	Contexts  map[int64]data.Context
	Preferred map[int64]string
}

func NewMockHistoryRepository() *MockHistoryRepository {
	return &MockHistoryRepository{
		Histories: make(map[int64][]data.History),
		Contexts:  make(map[int64]data.Context),
		Preferred: make(map[int64]string),
	}
}

func (m *MockHistoryRepository) GetContextById(contextId int64) (data.Context, error) {
	return m.Contexts[contextId], nil
}

func (m *MockHistoryRepository) InsertHistory(history data.History) (int64, error) {
	m.Histories[history.ContextId] = append(m.Histories[history.ContextId], history)
	return int64(len(m.Histories[history.ContextId])), nil
}

func (m *MockHistoryRepository) InsertContext(context data.Context) (int64, error) {
	m.Contexts[context.Id] = context
	return context.Id, nil
}

func (m *MockHistoryRepository) GetHistoryByContextId(contextId int64, maxCount int) ([]data.History, error) {
	h := m.Histories[contextId]
	if maxCount > 0 && len(h) > maxCount {
		h = h[:maxCount]
	}
	result := make([]data.History, len(h))
	copy(result, h)
	return result, nil
}

func (m *MockHistoryRepository) GetContextByName(name string) (*data.Context, error) {
	for _, ctx := range m.Contexts {
		if ctx.Name == name {
			copyCtx := ctx
			return &copyCtx, nil
		}
	}
	return nil, nil
}

func (m *MockHistoryRepository) GetAllContexts() ([]data.Context, error) {
	result := make([]data.Context, 0, len(m.Contexts))
	for _, ctx := range m.Contexts {
		result = append(result, ctx)
	}
	return result, nil
}

func (m *MockHistoryRepository) DeleteContext(contextId int64) (int64, error) { return 0, nil }
func (m *MockHistoryRepository) DeleteHistory(historyId int64) (int64, error) { return 0, nil }
func (m *MockHistoryRepository) UpdateSystemPrompt(contextId int64, systemPrompt string) error {
	return nil
}

func (m *MockHistoryRepository) UpdatePreferredModel(contextId int64, model string) error {
	m.Preferred[contextId] = model
	return nil
}

func (m *MockHistoryRepository) UpdatePreferredAgent(contextId int64, agent string) error {
	ctx := m.Contexts[contextId]
	ctx.PreferredAgent = agent
	m.Contexts[contextId] = ctx
	return nil
}

func (m *MockHistoryRepository) UpdatePreferredSkills(contextId int64, skills string) error {
	ctx := m.Contexts[contextId]
	ctx.PreferredSkills = skills
	m.Contexts[contextId] = ctx
	return nil
}

func (m *MockHistoryRepository) ArchiveContext(contextId int64, archived bool) error { return nil }
func (m *MockHistoryRepository) ArchiveHistory(historyId int64, archived bool) error { return nil }
