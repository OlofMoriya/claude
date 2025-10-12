package tui

import (
	"fmt"
	"owl/data"
	"owl/logger"
	"owl/services"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
)

type chatViewModel struct {
	shared          *sharedState
	history         []data.History
	textarea        textarea.Model
	viewport        viewport.Model
	loading         bool
	ready           bool
	sending         bool
	width           int
	height          int
	err             error
	historyLoaded   bool
	currentResponse string
	currentPrompt   string
	responseChan    chan string
	doneChan        chan struct{}
}

type historyLoadedMsg []data.History
type messageReceivedMsg string
type messageDoneMsg struct {
	prompt   string
	response string
}

func newChatViewModel(shared *sharedState) *chatViewModel {
	ta := textarea.New()
	ta.Placeholder = "Type your message..."
	ta.Focus()
	ta.CharLimit = 5000
	ta.SetWidth(shared.width - 4)
	ta.SetHeight(3)

	vp := viewport.New(shared.width, shared.height-10)
	vp.YPosition = 0

	return &chatViewModel{
		shared:   shared,
		textarea: ta,
		viewport: vp,
		loading:  true,
		ready:    false,
		width:    shared.width,
		height:   shared.height,
	}
}

func (m *chatViewModel) Init() tea.Cmd {

	logger.Debug.Printf("init")

	return tea.Batch(
		textarea.Blink,
		m.loadHistory(),
	)
}

func (m *chatViewModel) loadHistory() tea.Cmd {
	logger.Debug.Println("loadHistory")
	return func() tea.Msg {
		history, err := m.shared.config.Repository.GetHistoryByContextId(
			m.shared.selectedCtx.Id,
			m.shared.config.HistoryCount,
		)
		// debugLog.Printf("history %s", history)

		if err != nil {
			return errorMsg{err}
		}
		logger.Debug.Println("returning historyLoadedMsg")
		return historyLoadedMsg(history)
	}
}

type chatChunkMsg struct {
	text         string
	responseChan chan string
	doneChan     chan struct{}
	prompt       string
}

type chatCompleteMsg struct {
	prompt string
}

type chatErrorMsg struct {
	err error
}

func (m *chatViewModel) sendMessage(prompt string) tea.Cmd {
	return func() tea.Msg {
		// Create channels
		responseChan := make(chan string, 100)
		doneChan := make(chan struct{})

		handler := &tuiResponseHandler{
			responseChan: responseChan,
			doneChan:     doneChan,
			fullResponse: "",
			Repository:   m.shared.config.Repository,
		}

		model := m.shared.config.Model
		model.SetResponseHandler(handler)

		// Start query in goroutine
		go func() {
			services.StreamedQuery(
				prompt,
				model,
				m.shared.config.Repository,
				m.shared.config.HistoryCount,
				m.shared.selectedCtx,
				false,
				"",
			)
		}()

		// Execute the wait function immediately
		waitCmd := waitForChatActivity(responseChan, doneChan, prompt)
		return waitCmd() // ← Execute the function!
	}
}

func waitForChatActivity(responseChan chan string, doneChan chan struct{},
	prompt string) tea.Cmd {

	logger.Debug.Println("waitForChatActivity started")

	return func() tea.Msg {
		logger.Debug.Println("inside the fuction")
		logger.Debug.Printf("responseChan %s", responseChan)
		select {
		case text, ok := <-responseChan:
			logger.Debug.Printf("responseChan: %v: %s", ok, text)
			if !ok {
				// Channel closed, streaming done
				return chatCompleteMsg{prompt: prompt}
			}
			return chatChunkMsg{
				text:         text,
				responseChan: responseChan, // Pass channels forward
				doneChan:     doneChan,
				prompt:       prompt,
			}
		case <-doneChan:
			logger.Debug.Printf("doneChan from waitForChatActivity")
			return chatCompleteMsg{prompt: prompt}
		}
	}
}

func (m *chatViewModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		tiCmd tea.Cmd
		vpCmd tea.Cmd
	)

	switch msg := msg.(type) {
	case chatChunkMsg:
		logger.Debug.Println("got chatChunkMsg")
		// Append the chunk to your response
		//TODO: update ui with this text
		m.currentResponse += msg.text

		// CRITICAL: Continue listening for more chunks
		return m, waitForChatActivity(msg.responseChan, msg.doneChan, msg.prompt)

	case chatCompleteMsg:
		logger.Debug.Println("got chatCompleteMsg")
		m.sending = false
		return m, m.loadHistory()

	case chatErrorMsg:
		// Handle error
		m.err = msg.err
		return m, nil

	case tea.KeyMsg:
		if m.sending {
			return m, nil
		}

		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit

		case "esc":
			// Return to list view
			listView := newListViewModel(m.shared.config)
			return listView, tea.Sequence(
				listView.Init(),
				func() tea.Msg {
					return tea.WindowSizeMsg{
						Width:  m.shared.width,
						Height: m.shared.height,
					}
				},
			)

		case "ctrl+s":
			// Copy command to clipboard
			cmd := fmt.Sprintf("owl --context_name %s --prompt \"\"",
				m.shared.selectedCtx.Name)
			clipboard.WriteAll(cmd)

		case "ctrl+d":
			// Send message
			if !m.sending && m.textarea.Value() != "" {
				m.sending = true
				prompt := m.textarea.Value()
				m.textarea.Reset()
				return m, m.sendMessage(prompt)
			}
		}

	case historyLoadedMsg:
		logger.Debug.Printf("historyLoaded msg handled")
		m.history = []data.History(msg)
		// debugLog.Printf("historyLoaded msg handled %s, %s", len(m.history), msg)
		m.loading = false
		m.historyLoaded = true
		m.updateViewportContent()

	case messageDoneMsg:
		m.sending = false
		// Reload history
		return m, m.loadHistory()

	case errorMsg:
		m.err = msg.err
		m.loading = false
		m.sending = false

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		logger.Debug.Printf("windowSizeMsg")
		if !m.ready {
			logger.Debug.Printf("windowSizeMsg not ready")
			m.viewport = viewport.New(msg.Width, msg.Height-10)
			m.viewport.YPosition = 0
			m.ready = true
			m.updateViewportContent()
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - 10
		}

		m.textarea.SetWidth(msg.Width - 4)
	}

	m.textarea, tiCmd = m.textarea.Update(msg)
	m.viewport, vpCmd = m.viewport.Update(msg)

	return m, tea.Batch(tiCmd, vpCmd)
}

func (m *chatViewModel) updateViewportContent() {

	logger.Debug.Printf("historyLoaded %s", m.historyLoaded)
	if !m.historyLoaded {
		return
	}
	logger.Debug.Printf("passed guard")

	var b strings.Builder

	logger.Debug.Printf("history count %i", len(m.history))
	for i, h := range m.history {
		logger.Debug.Printf("add history %i", i)
		// Render user prompt
		b.WriteString(userPromptStyle.Render(fmt.Sprintf("You: %s", h.Prompt)))
		b.WriteString("\n\n")

		// Render AI response with glamour
		rendered, err := glamour.Render(h.Response, "dark")
		if err != nil {
			rendered = h.Response
		}
		b.WriteString(aiResponseStyle.Render(rendered))
		b.WriteString("\n")
		b.WriteString(strings.Repeat("─", m.width))
		b.WriteString("\n\n")
	}
	// debugLog.Printf(b.String())

	m.viewport.SetContent(b.String())
	if m.viewport.Height > 0 && m.viewport.Width > 0 {
		m.viewport.GotoBottom()
	}
}

func (m *chatViewModel) View() string {
	if m.loading || !m.historyLoaded {
		return loadingStyle.Render("Loading conversation...")
	}

	if m.err != nil {
		return errorStyle.Render(fmt.Sprintf("Error: %v\n\nPress ESC to go back", m.err))
	}

	status := ""
	if m.sending {
		status = sendingStyle.Render(" Sending...")
	}

	return fmt.Sprintf(
		"%s\n\n%s\n\n%s\n%s%s",
		headerStyle.Render(fmt.Sprintf("💬 %s", m.shared.selectedCtx.Name)),
		m.viewport.View(),
		m.textarea.View(),
		helpStyle.Render("ctrl+d send • ctrl+s copy cmd • esc back • ctrl+c quit"),
		status,
	)
}

// TUI-specific response handler
type tuiResponseHandler struct {
	responseChan chan string
	doneChan     chan struct{}
	fullResponse string
	Repository   data.HistoryRepository
}

func (h *tuiResponseHandler) RecievedText(text string, color *string) {
	h.fullResponse += text
	h.responseChan <- text
}

func (h *tuiResponseHandler) FinalText(contextId int64, prompt string,
	response string) {
	h.fullResponse = response

	history := data.History{
		ContextId:    contextId,
		Prompt:       prompt,
		Response:     response,
		Abbreviation: "",
		TokenCount:   0,
	}

	_, err := h.Repository.InsertHistory(history)
	if err != nil {
		println(fmt.Sprintf("Error while trying to save history: %s", err))
	}

	code := services.ExtractCodeBlocks(response)
	allCode := strings.Join(code, "\n\n")

	err = clipboard.WriteAll(allCode)
	if err != nil {
		fmt.Printf("Error copying to clipboard: %v\n", err)
	}

	// Signal done BEFORE closing responseChan
	logger.Debug.Println("Final text in tui response channel")
	logger.Debug.Println("closing doneChan and responseChan")
	close(h.doneChan)
	close(h.responseChan)
}
