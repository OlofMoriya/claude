package tui

import (
	"owl/data"
	"owl/logger"
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
	// Create the status channel for logger messages
	logger.StatusChan = make(chan string, 50) // buffered channel
	defer func() {
		// Clean up when TUI exits
		close(logger.StatusChan)
		logger.StatusChan = nil
	}()

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
