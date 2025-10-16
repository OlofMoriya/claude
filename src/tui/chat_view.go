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
	claude_model "owl/models/claude"
	grok_model "owl/models/grok"
	openai_4o_model "owl/models/open-ai-4o"
)

type chatMode int

const (
	chatInputMode chatMode = iota
	chatNormalMode
	chatModelSelectMode
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
	mode            chatMode

	// Model selection
	availableModels  []string
	selectedModelIdx int
	modelCursor      int

	historyCount int
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

	// TODO: Get this list from config or a model provider
	availableModels := []string{
		"claude",
		"grok",
		"4o",
		"opus",
		"sonnet",
	}

	return &chatViewModel{
		shared:           shared,
		textarea:         ta,
		viewport:         vp,
		loading:          true,
		ready:            false,
		width:            shared.width,
		height:           shared.height,
		availableModels:  availableModels,
		selectedModelIdx: 0,
		mode:             chatInputMode,
		historyCount:     shared.config.HistoryCount,
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
	return func() tea.Msg {
		history, err := m.shared.config.Repository.GetHistoryByContextId(
			m.shared.selectedCtx.Id,
			1000,
		)

		if err != nil {
			return errorMsg{err}
		}
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
	prompt   string
	response string
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

		logger.Debug.Printf("MODEL SELECTION: %s", m.availableModels[m.selectedModelIdx])
		switch m.availableModels[m.selectedModelIdx] {
		case "grok":
			logger.Debug.Println("setting grok as model")
			model = &grok_model.GrokModel{ResponseHandler: handler}
		case "4o":
			model = &openai_4o_model.OpenAi4oModel{ResponseHandler: handler}
		case "claude":
			model = &claude_model.ClaudeModel{ResponseHandler: handler, UseThinking: true, StreamThought: true, OutputThought: false}
		case "opus":
			model = &claude_model.ClaudeModel{ResponseHandler: handler, UseThinking: true, StreamThought: true, OutputThought: false, ModelVersion: "opus"}
		case "sonnet":
			model = &claude_model.ClaudeModel{ResponseHandler: handler, UseThinking: true, StreamThought: true, OutputThought: false, ModelVersion: "sonnet"}
		default:
			model = &claude_model.ClaudeModel{ResponseHandler: handler, UseThinking: true, StreamThought: true, OutputThought: false}
		}

		m.shared.config.Model = model

		logger.Debug.Printf("MODEL SELECTION after: %s", model)

		model.SetResponseHandler(handler)

		// Start query in goroutine
		go func() {
			services.StreamedQuery(
				prompt,
				model,
				m.shared.config.Repository,
				m.historyCount,
				m.shared.selectedCtx,
				false,
				"",
			)
		}()

		// Execute the wait function immediately
		waitCmd := waitForChatActivity(responseChan, doneChan, prompt)
		return waitCmd()
	}
}

func waitForChatActivity(responseChan chan string, doneChan chan struct{},
	prompt string) tea.Cmd {

	logger.Debug.Println("waitForChatActivity started")

	return func() tea.Msg {
		logger.Debug.Println("inside the function")
		select {
		case text, ok := <-responseChan:
			logger.Debug.Printf("responseChan: %v: %s", ok, text)
			if !ok {
				// Channel closed, streaming done
				return chatCompleteMsg{prompt: prompt}
			}
			return chatChunkMsg{
				text:         text,
				responseChan: responseChan,
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

	shouldUpdateViewport := false

	switch msg := msg.(type) {
	case chatChunkMsg:
		// Update UI with new text chunk
		m.currentResponse += msg.text
		m.updateViewportContent()
		// Important: return another command to keep listening
		return m, waitForChatActivity(msg.responseChan, msg.doneChan, msg.prompt)

	case chatCompleteMsg:
		logger.Debug.Println("got chatCompleteMsg")
		// Streaming complete, finalize UI
		m.sending = false
		m.loading = false
		// Reload history to show the new message
		return m, m.loadHistory()

	case chatErrorMsg:
		// Handle error
		m.err = msg.err
		m.loading = false
		m.sending = false

	case tea.KeyMsg:
		if m.sending {
			return m, nil
		}

		// Handle NORMAL mode - vim-like navigation
		if m.mode == chatNormalMode {
			switch msg.String() {
			case "i":
				// Enter input mode
				m.mode = chatInputMode
				m.textarea.Focus()
				return m, nil

			case "esc", "q":
				// Return to list view from normal mode
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

			case "=", "+":
				// Increase history count
				if m.historyCount < 50 {
					m.historyCount++
					return m, m.loadHistory()
				}
				return m, nil

			case "-", "_":
				// Decrease history count
				if m.historyCount > 0 {
					m.historyCount--
					return m, m.loadHistory()
				}
				return m, nil

			case "d", "ctrl+d":
				// Scroll down (vim-like)
				shouldUpdateViewport = true
				m.viewport.HalfPageDown()

			case "u", "ctrl+u":
				// Scroll up (vim-like)
				shouldUpdateViewport = true
				m.viewport.HalfPageUp()

			case "f", "ctrl+f", "pgdown":
				// Page down
				shouldUpdateViewport = true
				m.viewport.PageDown()

			case "b", "ctrl+b", "pgup":
				// Page up
				shouldUpdateViewport = true
				m.viewport.PageUp()

			case "g":
				// Go to top
				shouldUpdateViewport = true
				m.viewport.GotoTop()

			case "G":
				// Go to bottom
				shouldUpdateViewport = true
				m.viewport.GotoBottom()

			case "j", "down":
				// Scroll down one line
				shouldUpdateViewport = true
				m.viewport.ScrollDown(1)

			case "k", "up":
				// Scroll up one line
				shouldUpdateViewport = true
				m.viewport.ScrollUp(1)

			case "ctrl+a":
				// Open history detail view
				historyView := newChatHistoryViewModel(m.shared)
				return historyView, tea.Sequence(
					historyView.Init(),
					func() tea.Msg {
						return tea.WindowSizeMsg{
							Width:  m.shared.width,
							Height: m.shared.height,
						}
					},
				)

			case "ctrl+g":
				// Open model selector
				m.mode = chatModelSelectMode
				m.modelCursor = m.selectedModelIdx
				return m, nil

			case "ctrl+s":
				// Copy command to clipboard
				cmd := fmt.Sprintf("owl --context_name %s --prompt \"\"",
					m.shared.selectedCtx.Name)
				clipboard.WriteAll(cmd)
				return m, nil
			}

			// In normal mode, don't update textarea
			if shouldUpdateViewport {
				m.viewport, vpCmd = m.viewport.Update(msg)
			}
			return m, vpCmd
		}

		// Handle MODEL SELECT mode
		if m.mode == chatModelSelectMode {
			switch msg.String() {
			case "esc", "q":
				m.mode = chatInputMode
				m.modelCursor = m.selectedModelIdx
				return m, nil

			case "up", "k":
				if m.modelCursor > 0 {
					m.modelCursor--
				}

			case "down", "j":
				if m.modelCursor < len(m.availableModels)-1 {
					m.modelCursor++
				}

			case "enter":
				m.selectedModelIdx = m.modelCursor
				m.mode = chatInputMode
				return m, nil
			}
			return m, nil
		}

		// Handle INPUT mode (default)
		switch msg.String() {
		case "ctrl+n":
			// Enter normal mode
			m.mode = chatNormalMode
			m.textarea.Blur()
			return m, nil

		case "ctrl+a":
			// Open history detail view
			historyView := newChatHistoryViewModel(m.shared)
			return historyView, tea.Sequence(
				historyView.Init(),
				func() tea.Msg {
					return tea.WindowSizeMsg{
						Width:  m.shared.width,
						Height: m.shared.height,
					}
				},
			)

		case "ctrl+c":
			return m, tea.Quit

		case "esc":
			// Return to list view from input mode
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

		case "ctrl+g":
			// Open model selector
			m.mode = chatModelSelectMode
			m.modelCursor = m.selectedModelIdx
			return m, nil

		case "ctrl+w":
			// Send message
			if !m.sending && m.textarea.Value() != "" {
				m.sending = true
				prompt := m.textarea.Value()
				m.currentPrompt = prompt
				m.currentResponse = ""
				m.textarea.Reset()
				return m, m.sendMessage(prompt)
			}

		// Allow viewport scrolling ONLY with ctrl+ in input mode
		case "ctrl+u":
			shouldUpdateViewport = true
			m.viewport.HalfPageUp()

		case "ctrl+d":
			shouldUpdateViewport = true
			m.viewport.HalfPageDown()

		case "ctrl+b", "pgup":
			shouldUpdateViewport = true
			m.viewport.PageUp()

		case "ctrl+f", "pgdown":
			shouldUpdateViewport = true
			m.viewport.PageDown()

		// All other keys in input mode go to textarea
		default:
			// Update textarea for normal typing
			m.textarea, tiCmd = m.textarea.Update(msg)
			return m, tiCmd
		}

	case historyLoadedMsg:
		logger.Debug.Printf("historyLoaded msg handled")
		m.history = []data.History(msg)
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

	// Only update viewport if we explicitly want to scroll
	if shouldUpdateViewport {
		m.viewport, vpCmd = m.viewport.Update(msg)
	}

	return m, tea.Batch(tiCmd, vpCmd)
}

func (m *chatViewModel) View() string {
	if m.loading || !m.historyLoaded {
		return loadingStyle.Render("Loading conversation...")
	}

	if m.err != nil {
		return errorStyle.Render(fmt.Sprintf("Error: %v\n\nPress ESC to go back", m.err))
	}

	// Show model selector
	if m.mode == chatModelSelectMode {
		return m.renderModelSelector()
	}

	// Normal chat view
	status := ""
	if m.sending {
		status = sendingStyle.Render(" Sending...")
	}

	// Show mode indicator
	modeIndicator := ""
	switch m.mode {
	case chatNormalMode:
		modeIndicator = dimStyle.Render(" [NORMAL]")
	case chatInputMode:
		modeIndicator = dimStyle.Render(" [INPUT]")
	}

	currentModel := m.availableModels[m.selectedModelIdx]
	modelInfo := dimStyle.Render(fmt.Sprintf(" [%s] [history: %d]", currentModel, m.historyCount))

	// Different help text based on mode
	helpText := ""
	if m.mode == chatNormalMode {
		helpText = "i: input • d/u: scroll • g/G: top/bottom • +/-: history • ctrl+g: model • ctrl+h: history • esc: back"
	} else {
		helpText = "ctrl+n: normal • ctrl+w: send • ctrl+g: model • ctrl+u/d: scroll • ctrl+h: history • esc: back"
	}

	return fmt.Sprintf(
		"%s%s%s\n\n%s\n\n%s\n%s%s",
		headerStyle.Render(fmt.Sprintf("💬 %s", m.shared.selectedCtx.Name)),
		modelInfo,
		modeIndicator,
		m.viewport.View(),
		m.textarea.View(),
		helpStyle.Render(helpText),
		status,
	)
}

func (m *chatViewModel) updateViewportContent() {
	if !m.historyLoaded {
		return
	}
	logger.Debug.Printf("passed guard")

	var b strings.Builder

	logger.Debug.Printf("history count %d", len(m.history))
	for i, h := range m.history {
		logger.Debug.Printf("add history %d", i)
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

	// Show current response if streaming
	if m.sending && m.currentResponse != "" {
		b.WriteString(userPromptStyle.Render(fmt.Sprintf("You: %s", m.currentPrompt)))
		b.WriteString("\n\n")

		rendered, err := glamour.Render(m.currentResponse, "dark")
		if err != nil {
			rendered = m.currentResponse
		}
		b.WriteString(aiResponseStyle.Render(rendered))
		b.WriteString("\n")
	}

	m.viewport.SetContent(b.String())
	if m.viewport.Height > 0 && m.viewport.Width > 0 {
		m.viewport.GotoBottom()
	}
}

func (m *chatViewModel) renderModelSelector() string {
	var b strings.Builder

	b.WriteString(headerStyle.Render("Select Model"))
	b.WriteString("\n\n")

	for i, model := range m.availableModels {
		cursor := " "
		style := itemStyle

		if i == m.modelCursor {
			cursor = ">"
			style = selectedItemStyle
		}

		// Highlight the currently selected model
		modelName := model
		if i == m.selectedModelIdx {
			modelName += " ✓"
		}

		line := fmt.Sprintf("%s %s", cursor, modelName)
		b.WriteString(style.Render(line))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(helpStyle.Render("↑/k up • ↓/j down • enter select • esc/q cancel"))

	return b.String()
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

func (h *tuiResponseHandler) FinalText(contextId int64, prompt string, response string) {
	h.fullResponse = response

	// Signal done BEFORE closing responseChan
	logger.Debug.Println("Final text in tui response channel")
	logger.Debug.Println("closing doneChan and responseChan")
	close(h.doneChan)
	close(h.responseChan)

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

	// Copy to clipboard
	err = clipboard.WriteAll(allCode)
	if err != nil {
		fmt.Printf("Error copying to clipboard: %v\n", err)
	}
}
