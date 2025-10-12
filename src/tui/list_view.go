package tui

import (
	"fmt"
	"strings"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
)

type listViewModel struct {
	shared   *sharedState
	cursor   int
	loading  bool
	viewport int
	quitting bool
}

type contextsLoadedMsg []contextItem
type errorMsg struct{ err error }

func newListViewModel(config TUIConfig) *listViewModel {
	shared := &sharedState{
		config:   config,
		contexts: []contextItem{},
	}

	return &listViewModel{
		shared:   shared,
		loading:  true,
		viewport: 0,
	}
}

func (m *listViewModel) Init() tea.Cmd {
	return m.loadContexts()
}

func (m *listViewModel) loadContexts() tea.Cmd {
	return func() tea.Msg {
		contexts, err := m.shared.config.Repository.GetAllContexts()
		if err != nil {
			return errorMsg{err}
		}

		items := make([]contextItem, len(contexts))
		for i, ctx := range contexts {
			history, _ := m.shared.config.Repository.GetHistoryByContextId(ctx.Id, 1000)
			items[i] = contextItem{
				context:      ctx,
				messageCount: len(history),
			}
		}

		return contextsLoadedMsg(items)
	}
}

func (m *listViewModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.loading {
			return m, nil
		}

		switch msg.String() {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			if m.cursor < len(m.shared.contexts)-1 {
				m.cursor++
			}

		case "enter":
			// Open chat view for selected context
			if len(m.shared.contexts) > 0 {

				m.shared.selectedCtx = &m.shared.contexts[m.cursor].context
				chatView := newChatViewModel(m.shared)
				return chatView, tea.Sequence(
					chatView.Init(),
					func() tea.Msg {
						return tea.WindowSizeMsg{
							Width:  m.shared.width,
							Height: m.shared.height,
						}
					},
				)
			}

		case "c":
			// Copy command to clipboard
			if len(m.shared.contexts) > 0 {
				ctx := m.shared.contexts[m.cursor].context
				cmd := fmt.Sprintf("owl --context_name %s --prompt \"your prompt here\"", ctx.Name)
				clipboard.WriteAll(cmd)
			}

		case "r":
			// Refresh contexts
			m.loading = true
			return m, m.loadContexts()
		}

	case contextsLoadedMsg:
		m.shared.contexts = []contextItem(msg)
		m.loading = false

	case errorMsg:
		m.shared.err = msg.err
		m.loading = false

	case tea.WindowSizeMsg:
		m.viewport = msg.Height
		m.shared.width = msg.Width
		m.shared.height = msg.Height
	}

	return m, nil
}

func (m *listViewModel) View() string {
	if m.quitting {
		return ""
	}

	if m.loading {
		return loadingStyle.Render("Loading contexts...")
	}

	if m.shared.err != nil {
		return errorStyle.Render(fmt.Sprintf("Error: %v", m.shared.err))
	}

	var b strings.Builder

	// Header
	b.WriteString(headerStyle.Render("🦉 OWL Contexts"))
	b.WriteString("\n\n")

	// Context list
	if len(m.shared.contexts) == 0 {
		b.WriteString(dimStyle.Render("No contexts found. Create one using the CLI first."))
	} else {
		for i, item := range m.shared.contexts {
			cursor := " "
			style := itemStyle

			if i == m.cursor {
				cursor = ">"
				style = selectedItemStyle
			}

			line := fmt.Sprintf("%s %s (%d messages)",
				cursor,
				item.context.Name,
				item.messageCount,
			)

			if item.context.SystemPrompt != "" {
				line += dimStyle.Render(fmt.Sprintf(" [system: %.40s...]", item.context.SystemPrompt))
			}

			b.WriteString(style.Render(line))
			b.WriteString("\n")
		}
	}

	// Footer
	b.WriteString("\n")
	b.WriteString(helpStyle.Render(
		"↑/k up • ↓/j down • enter select • c copy command • r refresh • q quit",
	))

	return b.String()
}
