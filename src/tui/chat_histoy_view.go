package tui

import (
	"fmt"
	"owl/data"
	"owl/services"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
)

type historyViewMode int

const (
	historyCompactMode historyViewMode = iota
	historyExpandedMode
	historyCodeViewMode
)

type chatHistoryViewModel struct {
	shared      *sharedState
	history     []data.History
	viewport    viewport.Model
	cursor      int
	mode        historyViewMode
	expandedIdx int // Which message is expanded (-1 = none)
	ready       bool
	loading     bool
	width       int
	height      int
	err         error
}

func newChatHistoryViewModel(shared *sharedState) *chatHistoryViewModel {
	vp := viewport.New(shared.width, shared.height-5)
	vp.YPosition = 0

	return &chatHistoryViewModel{
		shared:      shared,
		viewport:    vp,
		cursor:      0,
		mode:        historyCompactMode,
		expandedIdx: -1,
		loading:     true,
		width:       shared.width,
		height:      shared.height,
	}
}

func (m *chatHistoryViewModel) Init() tea.Cmd {
	return m.loadHistory()
}

func (m *chatHistoryViewModel) loadHistory() tea.Cmd {
	return func() tea.Msg {
		history, err := m.shared.config.Repository.GetHistoryByContextId(
			m.shared.selectedCtx.Id,
			1000, // Load all history for browsing
		)

		if err != nil {
			return errorMsg{err}
		}
		return historyLoadedMsg(history)
	}
}

func (m *chatHistoryViewModel) scrollToSelection() {
	if len(m.history) == 0 {
		return
	}

	// Each message in compact mode takes 3 lines (prompt + response + blank)
	linesPerMessage := 3

	// Calculate the line position of the current cursor
	cursorLine := m.cursor * linesPerMessage

	// Calculate visible range
	viewportTop := m.viewport.YOffset
	viewportBottom := viewportTop + m.viewport.Height

	// If cursor is above viewport, scroll up
	if cursorLine < viewportTop {
		m.viewport.SetYOffset(cursorLine)
	}

	// If cursor is below viewport, scroll down
	// We want the cursor item to be visible, so check if cursor + item height is visible
	cursorBottom := cursorLine + linesPerMessage
	if cursorBottom > viewportBottom {
		// Scroll so the cursor item is at the bottom of the viewport
		newOffset := cursorBottom - m.viewport.Height
		if newOffset < 0 {
			newOffset = 0
		}
		m.viewport.SetYOffset(newOffset)
	}
}

func (m *chatHistoryViewModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var vpCmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.loading {
			return m, nil
		}

		// Code view mode
		if m.mode == historyCodeViewMode {
			switch msg.String() {
			case "esc", "q", "c":
				m.mode = historyCompactMode
				m.updateContent()
				return m, nil
			case "y":
				// Copy code to clipboard
				if m.expandedIdx >= 0 && m.expandedIdx < len(m.history) {
					code := services.ExtractCodeBlocks(m.history[m.expandedIdx].Response)
					allCode := strings.Join(code, "\n\n")
					clipboard.WriteAll(allCode)
				}
				return m, nil
			}
			// Allow scrolling in code view
			m.viewport, vpCmd = m.viewport.Update(msg)
			return m, vpCmd
		}

		// Expanded view mode
		if m.mode == historyExpandedMode {
			switch msg.String() {
			case "esc", "q":
				m.mode = historyCompactMode
				m.expandedIdx = -1
				m.updateContent()
				m.scrollToSelection() // Scroll to cursor when returning to compact
				return m, nil
			case "c":
				// View code
				if m.expandedIdx >= 0 && m.expandedIdx < len(m.history) {
					m.mode = historyCodeViewMode
					m.updateContent()
				}
				return m, nil
			case "y":
				// Copy full response
				if m.expandedIdx >= 0 && m.expandedIdx < len(m.history) {
					clipboard.WriteAll(m.history[m.expandedIdx].Response)
				}
				return m, nil
			}
			// Allow scrolling in expanded view
			m.viewport, vpCmd = m.viewport.Update(msg)
			return m, vpCmd
		}

		// Compact mode navigation
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit

		case "esc", "q":
			// Return to chat view
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

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
				m.updateContent()
				m.scrollToSelection() // ← Add this
			}

		case "down", "j":
			if m.cursor < len(m.history)-1 {
				m.cursor++
				m.updateContent()
				m.scrollToSelection() // ← Add this
			}

		case "g":
			// Go to top
			m.cursor = 0
			m.updateContent()
			m.scrollToSelection() // ← Add this

		case "G":
			// Go to bottom
			if len(m.history) > 0 {
				m.cursor = len(m.history) - 1
			}
			m.updateContent()
			m.scrollToSelection() // ← Add this

		case "enter", "e":
			// Expand selected message
			m.expandedIdx = m.cursor
			m.mode = historyExpandedMode
			m.updateContent()

		case "c":
			// View code for selected message
			if m.cursor < len(m.history) {
				code := services.ExtractCodeBlocks(m.history[m.cursor].Response)
				if len(code) > 0 {
					m.expandedIdx = m.cursor
					m.mode = historyCodeViewMode
					m.updateContent()
				}
			}

		case "y":
			// Copy response of selected message
			if m.cursor < len(m.history) {
				clipboard.WriteAll(m.history[m.cursor].Response)
			}

		case "r":
			// Refresh
			m.loading = true
			return m, m.loadHistory()
		}

	case historyLoadedMsg:
		m.history = []data.History(msg)
		m.loading = false
		m.updateContent()
		m.scrollToSelection()

	case errorMsg:
		m.err = msg.err
		m.loading = false

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		if !m.ready {
			m.viewport = viewport.New(msg.Width, msg.Height-5)
			m.viewport.YPosition = 0
			m.ready = true
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - 5
		}
		m.updateContent()
		m.scrollToSelection()
	}

	return m, vpCmd
}

func (m *chatHistoryViewModel) updateContent() {
	if len(m.history) == 0 {
		m.viewport.SetContent(dimStyle.Render("No history yet"))
		return
	}

	switch m.mode {
	case historyCompactMode:
		m.viewport.SetContent(m.renderCompactMode())
	case historyExpandedMode:
		m.viewport.SetContent(m.renderExpandedMode())
	case historyCodeViewMode:
		m.viewport.SetContent(m.renderCodeViewMode())
	}
}

func (m *chatHistoryViewModel) renderCompactMode() string {
	var b strings.Builder

	for i, h := range m.history {
		cursor := "  "
		style := itemStyle

		if i == m.cursor {
			cursor = "▶ "
			style = selectedItemStyle
		}

		// Check if message contains code
		codeIndicator := ""
		code := services.ExtractCodeBlocks(h.Response)
		if len(code) > 0 {
			codeIndicator = dimStyle.Render(" 💻")
		}

		// Truncate prompt to 1 line
		prompt := h.Prompt
		if len(prompt) > m.width-10 {
			prompt = prompt[:m.width-13] + "..."
		}
		prompt = strings.ReplaceAll(prompt, "\n", " ")

		// Truncate response to 2 lines
		response := h.Response
		lines := strings.Split(response, "\n")
		if len(lines) > 2 {
			response = strings.Join(lines[:2], " ") + "..."
		} else {
			response = strings.ReplaceAll(response, "\n", " ")
		}
		if len(response) > m.width-10 {
			response = response[:m.width-13] + "..."
		}

		// Format: cursor [#] prompt | response [code]
		line := fmt.Sprintf("%s[%d]%s %s",
			cursor,
			i+1,
			codeIndicator,
			dimStyle.Render("Q:"))
		b.WriteString(style.Render(line))
		b.WriteString(" ")
		b.WriteString(style.Render(prompt))
		b.WriteString("\n")

		b.WriteString(style.Render("    " + dimStyle.Render("A: ") + response))
		b.WriteString("\n\n")
	}

	return b.String()
}

func (m *chatHistoryViewModel) renderExpandedMode() string {
	if m.expandedIdx < 0 || m.expandedIdx >= len(m.history) {
		return "Invalid message index"
	}

	h := m.history[m.expandedIdx]
	var b strings.Builder

	// Header
	b.WriteString(headerStyle.Render(fmt.Sprintf("Message %d of %d", m.expandedIdx+1, len(m.history))))
	b.WriteString("\n\n")

	// Check if has code
	code := services.ExtractCodeBlocks(h.Response)
	if len(code) > 0 {
		b.WriteString(dimStyle.Render(fmt.Sprintf("💻 Contains %d code block(s) - press 'c' to view", len(code))))
		b.WriteString("\n\n")
	}

	// Prompt
	b.WriteString(userPromptStyle.Render("Question:"))
	b.WriteString("\n")
	b.WriteString(h.Prompt)
	b.WriteString("\n\n")

	// Response
	b.WriteString(aiResponseStyle.Render("Answer:"))
	b.WriteString("\n")
	rendered, err := glamour.Render(h.Response, "dark")
	if err != nil {
		rendered = h.Response
	}
	b.WriteString(rendered)

	return b.String()
}

func (m *chatHistoryViewModel) renderCodeViewMode() string {
	if m.expandedIdx < 0 || m.expandedIdx >= len(m.history) {
		return "Invalid message index"
	}

	h := m.history[m.expandedIdx]
	code := services.ExtractCodeBlocks(h.Response)

	var b strings.Builder

	// Header
	b.WriteString(headerStyle.Render(fmt.Sprintf("Code Blocks - Message %d", m.expandedIdx+1)))
	b.WriteString("\n\n")

	if len(code) == 0 {
		b.WriteString(dimStyle.Render("No code blocks found"))
	} else {
		for i, block := range code {
			b.WriteString(dimStyle.Render(fmt.Sprintf("─── Block %d of %d ───", i+1, len(code))))
			b.WriteString("\n")
			b.WriteString(block)
			b.WriteString("\n\n")
		}
	}

	return b.String()
}

func (m *chatHistoryViewModel) View() string {
	if m.loading {
		return loadingStyle.Render("Loading history...")
	}

	if m.err != nil {
		return errorStyle.Render(fmt.Sprintf("Error: %v\n\nPress ESC to go back", m.err))
	}

	// Build help text based on mode
	var help string
	switch m.mode {
	case historyCompactMode:
		help = "↑/↓/j/k navigate • enter/e expand • c code • y copy • g top • G bottom • r refresh • esc/q back"
	case historyExpandedMode:
		help = "c view code • y copy • esc/q back"
	case historyCodeViewMode:
		help = "y copy code • esc/q back"
	}

	modeLabel := ""
	switch m.mode {
	case historyCompactMode:
		modeLabel = fmt.Sprintf("Compact (%d messages)", len(m.history))
	case historyExpandedMode:
		modeLabel = fmt.Sprintf("Message %d", m.expandedIdx+1)
	case historyCodeViewMode:
		modeLabel = fmt.Sprintf("Code View - Message %d", m.expandedIdx+1)
	}

	return fmt.Sprintf(
		"%s %s\n\n%s\n\n%s",
		headerStyle.Render(fmt.Sprintf("📜 History: %s", m.shared.selectedCtx.Name)),
		dimStyle.Render(fmt.Sprintf("[%s]", modeLabel)),
		m.viewport.View(),
		helpStyle.Render(help),
	)
}
