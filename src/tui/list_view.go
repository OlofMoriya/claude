package tui

import (
	"fmt"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"owl/data"
)

type listViewModel struct {
	shared        *sharedState
	cursor        int
	loading       bool
	viewport      int
	quitting      bool
	mode          listMode
	textInput     textinput.Model
	inputMode     inputMode
	showingPrompt bool
}

type listMode int
type inputMode int

const (
	normalMode listMode = iota
	confirmDeleteMode
	inputDialogMode
	showPromptMode
)

const (
	inputNewContext inputMode = iota
	inputSystemPrompt
)

type contextsLoadedMsg []contextItem
type contextCreatedMsg struct{ id int64 }
type contextDeletedMsg struct{}
type contextArchivedMsg struct{}
type promptUpdatedMsg struct{}
type errorMsg struct{ err error }

func newListViewModel(config TUIConfig) *listViewModel {
	shared := &sharedState{
		config:   config,
		contexts: []contextItem{},
	}

	ti := textinput.New()
	ti.Placeholder = "Enter text..."
	ti.Focus()
	ti.CharLimit = 5000
	ti.Width = 100

	return &listViewModel{
		shared:    shared,
		loading:   true,
		viewport:  0,
		mode:      normalMode,
		textInput: ti,
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

		var items []contextItem
		for _, ctx := range contexts {
			if ctx.Archived {
				continue
			}
			history, _ := m.shared.config.Repository.GetHistoryByContextId(ctx.Id, 1000)
			items = append(items, contextItem{
				context:      ctx,
				messageCount: len(history),
			})
		}

		return contextsLoadedMsg(items)
	}
}

func (m *listViewModel) createContext(name string) tea.Cmd {
	return func() tea.Msg {
		ctx := data.Context{Name: name}
		id, err := m.shared.config.Repository.InsertContext(ctx)
		if err != nil {
			return errorMsg{err}
		}
		return contextCreatedMsg{id}
	}
}

func (m *listViewModel) deleteContext(contextId int64) tea.Cmd {
	return func() tea.Msg {
		_, err := m.shared.config.Repository.DeleteContext(contextId)
		if err != nil {
			return errorMsg{err}
		}
		return contextDeletedMsg{}
	}
}

func (m *listViewModel) archiveContext(contextId int64) tea.Cmd {
	return func() tea.Msg {
		err := m.shared.config.Repository.ArchiveContext(contextId, true)
		if err != nil {
			return errorMsg{err}
		}
		return contextArchivedMsg{}
	}
}

func (m *listViewModel) updateSystemPrompt(contextId int64, prompt string) tea.Cmd {
	return func() tea.Msg {
		err := m.shared.config.Repository.UpdateSystemPrompt(contextId, prompt)
		if err != nil {
			return errorMsg{err}
		}
		return promptUpdatedMsg{}
	}
}

func (m *listViewModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.loading {
			return m, nil
		}

		// Handle input dialog mode
		if m.mode == inputDialogMode {
			switch msg.String() {
			case "enter":
				value := m.textInput.Value()
				m.textInput.SetValue("")
				m.mode = normalMode

				if value != "" {
					switch m.inputMode {
					case inputNewContext:
						return m, tea.Sequence(
							m.createContext(value),
							m.loadContexts(),
						)
					case inputSystemPrompt:
						if len(m.shared.contexts) > 0 {
							contextId := m.shared.contexts[m.cursor].context.Id
							return m, tea.Sequence(
								m.updateSystemPrompt(contextId, value),
								m.loadContexts(),
							)
						}
					}
				}
				return m, nil

			case "esc":
				m.textInput.SetValue("")
				m.mode = normalMode
				return m, nil
			default:
				var cmd tea.Cmd
				m.textInput, cmd = m.textInput.Update(msg)
				return m, cmd
			}
		}

		// Handle confirm delete mode
		if m.mode == confirmDeleteMode {
			switch msg.String() {
			case "y", "Y":
				if len(m.shared.contexts) > 0 {
					contextId := m.shared.contexts[m.cursor].context.Id
					m.mode = normalMode
					if m.cursor >= len(m.shared.contexts)-1 && m.cursor > 0 {
						m.cursor--
					}
					return m, tea.Sequence(
						m.deleteContext(contextId),
						m.loadContexts(),
					)
				}
				m.mode = normalMode
				return m, nil
			case "n", "N", "esc":
				m.mode = normalMode
				return m, nil
			}
			return m, nil
		}

		// Handle show prompt mode
		if m.mode == showPromptMode {
			switch msg.String() {
			case "esc", "s", "q":
				m.mode = normalMode
				return m, nil
			}
			return m, nil
		}

		// Normal mode key handling
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

		case "g":
			m.cursor = 0

		case "G":
			m.cursor = len(m.shared.contexts) - 1

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
				cmd := fmt.Sprintf("owl --context_name=\"%s\" --stream --history=10 --prompt='...", ctx.Name)
				clipboard.WriteAll(cmd)
			}

		case "n":
			// Create new context
			m.mode = inputDialogMode
			m.inputMode = inputNewContext
			m.textInput.Placeholder = "Enter context name..."
			m.textInput.Focus()

		case "p":
			// Set system prompt
			if len(m.shared.contexts) > 0 {
				m.mode = inputDialogMode
				m.inputMode = inputSystemPrompt
				currentPrompt := m.shared.contexts[m.cursor].context.SystemPrompt
				m.textInput.SetValue(currentPrompt)
				m.textInput.Placeholder = "Enter system prompt..."
				m.textInput.Focus()
			}

		case "s":
			// Show system prompt
			if len(m.shared.contexts) > 0 {
				m.mode = showPromptMode
			}

		case "a":
			// Archive context
			if len(m.shared.contexts) > 0 {
				contextId := m.shared.contexts[m.cursor].context.Id
				if m.cursor >= len(m.shared.contexts)-1 && m.cursor > 0 {
					m.cursor--
				}
				return m, tea.Sequence(
					m.archiveContext(contextId),
					m.loadContexts(),
				)
			}

		case "d":
			// Delete context (with confirmation)
			if len(m.shared.contexts) > 0 {
				m.mode = confirmDeleteMode
			}

		case "r":
			// Refresh contexts
			m.loading = true
			return m, m.loadContexts()
		}

	case contextsLoadedMsg:
		m.shared.contexts = []contextItem(msg)
		m.loading = false

	case contextCreatedMsg:
		// Context created, will be loaded by loadContexts

	case contextDeletedMsg:
		// Context deleted, will be refreshed by loadContexts

	case contextArchivedMsg:
		// Context archived, will be refreshed by loadContexts

	case promptUpdatedMsg:
		// Prompt updated, will be refreshed by loadContexts

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

	// Show input dialog if in input mode
	if m.mode == inputDialogMode {
		title := "Create New Context"
		if m.inputMode == inputSystemPrompt {
			title = "Set System Prompt"
		}

		b.WriteString(headerStyle.Render(title))
		b.WriteString("\n\n")
		b.WriteString(m.textInput.View())
		b.WriteString("\n\n")
		b.WriteString(helpStyle.Render("enter: confirm • esc: cancel"))
		return b.String()
	}

	// Show confirm delete dialog
	if m.mode == confirmDeleteMode && len(m.shared.contexts) > 0 {
		ctx := m.shared.contexts[m.cursor].context
		b.WriteString(errorStyle.Render(fmt.Sprintf("Delete context '%s'?", ctx.Name)))
		b.WriteString("\n\n")
		b.WriteString(helpStyle.Render("y: yes • n: no • esc: cancel"))
		return b.String()
	}

	// Show system prompt modal
	if m.mode == showPromptMode && len(m.shared.contexts) > 0 {
		ctx := m.shared.contexts[m.cursor].context
		b.WriteString(headerStyle.Render(fmt.Sprintf("System Prompt: %s", ctx.Name)))
		b.WriteString("\n\n")

		if ctx.SystemPrompt == "" {
			b.WriteString(dimStyle.Render("No system prompt set"))
		} else {
			b.WriteString(itemStyle.Render(ctx.SystemPrompt))
		}

		b.WriteString("\n\n")
		b.WriteString(helpStyle.Render("esc/s/q: close"))
		return b.String()
	}

	// Context list (normal mode)
	if len(m.shared.contexts) == 0 {
		b.WriteString(dimStyle.Render("No contexts found. Press 'n' to create one."))
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
				preview := item.context.SystemPrompt
				if len(preview) > 40 {
					preview = preview[:40] + "..."
				}
				line += " " + dimStyle.Render(fmt.Sprintf("[%s]", preview))
			}

			b.WriteString(style.Render(line))
			b.WriteString("\n")
		}
	}

	// Footer
	b.WriteString("\n")
	b.WriteString(helpStyle.Render(
		"↑/k up • ↓/j down • enter select • n new • p set prompt • s show prompt • a archive • d delete • c copy • r refresh • q quit",
	))

	return b.String()
}
