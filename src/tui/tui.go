package tui

import (
	"owl/data"
	"owl/models"

	tea "github.com/charmbracelet/bubbletea"
)

type TUIConfig struct {
	Repository   data.HistoryRepository
	Model        models.Model
	HistoryCount int
}

// Run starts the TUI application
func Run(config TUIConfig) error {
	p := tea.NewProgram(
		initialModel(config),
		tea.WithAltScreen(),
	)

	_, err := p.Run()
	return err
}

func initialModel(config TUIConfig) tea.Model {
	return newListViewModel(config)
}
