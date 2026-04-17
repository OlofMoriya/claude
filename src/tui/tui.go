package tui

import (
	commontypes "owl/common_types"
	"owl/data"
	"owl/logger"

	tea "github.com/charmbracelet/bubbletea"
)

type TUIConfig struct {
	Repository   data.HistoryRepository
	Model        commontypes.Model
	HistoryCount int
}

// Run starts the TUI application
func Run(config TUIConfig) error {
	// Create the status channel for logger messages
	logger.StatusChan = make(chan string, 50) // buffered channel
	logger.HistoryPersistedChan = make(chan int64, 50)
	defer func() {
		// Clean up when TUI exits
		close(logger.StatusChan)
		close(logger.HistoryPersistedChan)
		logger.StatusChan = nil
		logger.HistoryPersistedChan = nil
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
