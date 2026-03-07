package tui

import (
	"fmt"
	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
	commontypes "owl/common_types"
	"owl/data"
	"owl/logger"
	picker "owl/picker"
	"owl/services"
	"strings"
	"time"
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

	historyCount     int
	statusMessage    string
	showUsagePanel   bool
	usagePanelPinned bool
	contextUsage     commontypes.TokenUsage
	lastUsage        *commontypes.TokenUsage
}

const (
	usagePanelMinWidth = 120
	usagePanelWidth    = 32
	minContentWidth    = 40
	minTextareaWidth   = 20
)

type historyLoadedMsg []data.History
type messageReceivedMsg string
type messageDoneMsg struct {
	prompt   string
	response string
}

type statusMsg string

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

func newChatViewModel(shared *sharedState) *chatViewModel {
	ta := textarea.New()
	ta.Placeholder = "Type your message..."
	ta.Focus()
	ta.CharLimit = 5000
	ta.SetWidth(shared.width - 4)
	ta.SetHeight(3)

	vp := viewport.New(shared.width, shared.height-10)
	vp.YPosition = 0

	availableModels := []string{
		"sonnet",
		"codex",
		"grok",
		"opus",
		"gpt",
		"haiku",
		"ollama",
		"claude",
	}

	m := &chatViewModel{
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
		statusMessage:    "",
		showUsagePanel:   shared.width >= usagePanelMinWidth,
	}
	m.applyLayout()
	return m
}

func (m *chatViewModel) Init() tea.Cmd {
	logger.Debug.Printf("init")

	return tea.Batch(
		textarea.Blink,
		m.loadHistory(),
		m.listenForStatus(),
	)
}

func (m *chatViewModel) listenForStatus() tea.Cmd {
	return func() tea.Msg {
		if logger.StatusChan == nil {
			return nil
		}

		msg, ok := <-logger.StatusChan
		if !ok {
			return nil
		}

		return statusMsg(msg)
	}
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
		logger.Debug.Println("returning historyLoadedMsg")
		return historyLoadedMsg(history)
	}
}

func (m *chatViewModel) sendMessage(prompt string) tea.Cmd {
	return func() tea.Msg {
		responseChan := make(chan string, 100)
		doneChan := make(chan struct{})

		handler := &tuiResponseHandler{
			responseChan: responseChan,
			doneChan:     doneChan,
			fullResponse: "",
			Repository:   m.shared.config.Repository,
		}

		modelName := m.availableModels[m.selectedModelIdx]

		model, actualModelName := picker.GetModelForQuery(
			modelName,
			m.shared.selectedCtx,
			handler,
			m.shared.config.Repository,
			true,
			true,
			false,
			false,
		)

		m.shared.config.Model = model

		logger.Debug.Printf("MODEL SELECTION: %s (actual: %s)", modelName, actualModelName)

		go func() {
			services.StreamedQuery(
				prompt,
				model,
				m.shared.config.Repository,
				m.historyCount,
				m.shared.selectedCtx,
				&commontypes.PayloadModifiers{
					Pdf:   "",
					Web:   false,
					Image: false,
				},
				actualModelName,
			)
		}()

		waitCmd := waitForChatActivity(responseChan, doneChan, prompt)
		return waitCmd()
	}
}

func waitForChatActivity(responseChan chan string, doneChan chan struct{},
	prompt string) tea.Cmd {

	logger.Debug.Println("waitForChatActivity started")

	return func() tea.Msg {
		select {
		case text, ok := <-responseChan:
			// logger.Debug.Printf("responseChan: %v: %s", ok, text)
			if !ok {
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
	customViewportHandled := false

	switch msg := msg.(type) {
	case statusMsg:
		m.statusMessage = string(msg)
		if msg != "" {
			logger.Debug.Printf("Received status: %s", msg)
		}

		return m, tea.Batch(
			m.listenForStatus(),
			m.clearStatusAfterDelay(),
		)

	case chatChunkMsg:
		m.currentResponse += msg.text
		m.updateViewportContent()
		return m, waitForChatActivity(msg.responseChan, msg.doneChan, msg.prompt)

	case chatCompleteMsg:
		logger.Debug.Println("got chatCompleteMsg")
		m.sending = false
		m.loading = false
		m.statusMessage = ""
		return m, m.loadHistory()

	case chatErrorMsg:
		m.err = msg.err
		m.loading = false
		m.sending = false
		m.statusMessage = ""
		return m, nil

	case tea.KeyMsg:
		if m.sending {
			return m, nil
		}

		if m.mode == chatNormalMode {
			switch msg.String() {
			case "i":
				m.mode = chatInputMode
				m.textarea.Focus()
				return m, nil

			case "esc", "q":
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
				if m.historyCount < services.DefaultHistoryCount {
					m.historyCount++
					return m, m.loadHistory()
				}
				return m, nil

			case "-", "_":
				if m.historyCount > 1 {
					m.historyCount--
					return m, m.loadHistory()
				}
				return m, nil

			case "d", "ctrl+d":
				m.scrollHalfPage(true)
				customViewportHandled = true

			case "u", "ctrl+u":
				m.scrollHalfPage(false)
				customViewportHandled = true

			case "f", "ctrl+f", "pgdown":
				shouldUpdateViewport = true
				m.viewport.PageDown()

			case "b", "ctrl+b", "pgup":
				shouldUpdateViewport = true
				m.viewport.PageUp()

			case "g":
				shouldUpdateViewport = true
				m.viewport.GotoTop()

			case "G":
				shouldUpdateViewport = true
				m.viewport.GotoBottom()

			case "j", "down":
				shouldUpdateViewport = true
				m.viewport.ScrollDown(1)

			case "k", "up":
				shouldUpdateViewport = true
				m.viewport.ScrollUp(1)

			case "ctrl+a":
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
				m.mode = chatModelSelectMode
				m.modelCursor = m.selectedModelIdx
				return m, nil

			case "ctrl+s":
				cmd := fmt.Sprintf("owl --context_name %s --prompt \"\"",
					m.shared.selectedCtx.Name)
				clipboard.WriteAll(cmd)
				return m, nil

			case "ctrl+t":
				m.usagePanelPinned = true
				m.showUsagePanel = !m.showUsagePanel
				m.applyLayout()
				m.updateViewportContent()
				return m, nil
			}

			if shouldUpdateViewport {
				m.viewport, vpCmd = m.viewport.Update(msg)
			}
			return m, vpCmd
		}

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

		switch msg.String() {
		case "ctrl+n":
			m.mode = chatNormalMode
			m.textarea.Blur()
			return m, nil

		case "ctrl+a":
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
			cmd := fmt.Sprintf("owl --context_name %s --prompt \"\"",
				m.shared.selectedCtx.Name)
			clipboard.WriteAll(cmd)

		case "ctrl+g":
			m.mode = chatModelSelectMode
			m.modelCursor = m.selectedModelIdx
			return m, nil

		case "ctrl+t":
			m.usagePanelPinned = true
			m.showUsagePanel = !m.showUsagePanel
			m.applyLayout()
			m.updateViewportContent()
			return m, nil

		case "ctrl+w":
			if !m.sending && m.textarea.Value() != "" {
				m.sending = true
				prompt := m.textarea.Value()
				m.currentPrompt = prompt
				m.currentResponse = ""
				m.textarea.Reset()
				return m, m.sendMessage(prompt)
			}

		case "ctrl+u":
			m.scrollHalfPage(false)
			customViewportHandled = true

		case "ctrl+d":
			m.scrollHalfPage(true)
			customViewportHandled = true

		case "ctrl+b", "pgup":
			shouldUpdateViewport = true
			m.viewport.PageUp()

		case "ctrl+f", "pgdown":
			shouldUpdateViewport = true
			m.viewport.PageDown()

		default:
			m.textarea, tiCmd = m.textarea.Update(msg)
			return m, tiCmd
		}

	case historyLoadedMsg:
		logger.Debug.Printf("historyLoaded msg handled")
		m.history = []data.History(msg)
		m.loading = false
		m.historyLoaded = true
		m.recalculateUsage()
		m.updateViewportContent()

	case messageDoneMsg:
		m.sending = false
		return m, m.loadHistory()

	case errorMsg:
		m.err = msg.err
		m.loading = false
		m.sending = false

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		logger.Debug.Printf("windowSizeMsg")
		if !m.usagePanelPinned {
			m.showUsagePanel = msg.Width >= usagePanelMinWidth
		}

		if !m.ready {
			logger.Debug.Printf("windowSizeMsg not ready")
			m.viewport = viewport.New(m.contentWidth(), msg.Height-10)
			m.viewport.YPosition = 0
			m.ready = true
		}

		m.applyLayout()
		m.updateViewportContent()
	}

	if shouldUpdateViewport && !customViewportHandled {
		m.viewport, vpCmd = m.viewport.Update(msg)
	}

	return m, tea.Batch(tiCmd, vpCmd)
}

func (m *chatViewModel) clearStatusAfterDelay() tea.Cmd {
	return tea.Tick(3*time.Second, func(t time.Time) tea.Msg {
		return statusMsg("")
	})
}

func (m *chatViewModel) View() string {
	if m.loading || !m.historyLoaded {
		return loadingStyle.Render("Loading conversation...")
	}

	if m.err != nil {
		return errorStyle.Render(fmt.Sprintf("Error: %v\n\nPress ESC to go back", m.err))
	}

	if m.mode == chatModelSelectMode {
		return m.renderModelSelector()
	}

	status := ""
	if m.sending {
		status = sendingStyle.Render(" Sending...")
	}

	if m.statusMessage != "" {
		status = dimStyle.Render(fmt.Sprintf("%s %s", status, m.statusMessage))
	}

	modeIndicator := ""
	switch m.mode {
	case chatNormalMode:
		modeIndicator = dimStyle.Render(" [NORMAL]")
	case chatInputMode:
		modeIndicator = dimStyle.Render(" [INPUT]")
	}

	currentModel := m.availableModels[m.selectedModelIdx]
	modelInfo := dimStyle.Render(fmt.Sprintf(" [%s] [history: %d]", currentModel, m.historyCount))

	helpText := ""
	if m.mode == chatNormalMode {
		helpText = "i: input • d/u: scroll • g/G: top/bottom • +/-: history • ctrl+g: model • ctrl+a: history • ctrl+t: usage • esc: back"
	} else {
		helpText = "ctrl+n: normal • ctrl+w: send • ctrl+g: model • ctrl+u/d: scroll • ctrl+a: history • ctrl+t: usage • esc: back"
	}

	mainContent := fmt.Sprintf(
		"%s%s%s\n\n%s\n\n%s\n%s%s",
		headerStyle.Render(fmt.Sprintf("💬 %s", m.shared.selectedCtx.Name)),
		modelInfo,
		modeIndicator,
		m.viewport.View(),
		m.textarea.View(),
		helpStyle.Render(helpText),
		status,
	)
	contentWidth := m.contentWidth()
	mainRendered := lipgloss.NewStyle().Width(contentWidth).Render(mainContent)

	if m.showUsagePanel {
		panel := m.renderUsagePanel()
		if panel != "" {
			return lipgloss.JoinHorizontal(
				lipgloss.Top,
				mainRendered,
				usagePanelStyle.Width(usagePanelWidth).Render(panel),
			)
		}
	}

	return mainRendered
}

func (m *chatViewModel) updateViewportContent() {
	if !m.historyLoaded {
		return
	}

	var b strings.Builder

	for _, h := range m.history {
		pStyle := userPromptStyle
		rStyle := aiResponseStyle
		archivedPrefix := ""

		if h.Archived {
			pStyle = dimStyle
			rStyle = dimStyle
			archivedPrefix = "[ARCHIVED] "
		}

		b.WriteString(pStyle.Render(fmt.Sprintf("%sYou: %s", archivedPrefix, h.Prompt)))
		b.WriteString("\n\n")

		rendered := renderMarkdown(h.Response, m.viewport.Width-4)
		b.WriteString(rStyle.Render(rendered))
		b.WriteString("\n")
		b.WriteString(dimStyle.Render(strings.Repeat("─", m.width)))
		b.WriteString("\n\n")
	}

	if m.sending && m.currentResponse != "" {
		b.WriteString(userPromptStyle.Render(fmt.Sprintf("You: %s", m.currentPrompt)))
		b.WriteString("\n\n")

		rendered := renderMarkdown(m.currentResponse, m.viewport.Width-4)
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

func (m *chatViewModel) recalculateUsage() {
	var total commontypes.TokenUsage
	var last *commontypes.TokenUsage
	for _, h := range m.history {
		total.PromptTokens += h.PromptTokens
		total.CompletionTokens += h.CompletionTokens
		total.CacheReadTokens += h.CacheReadTokens
		total.CacheWriteTokens += h.CacheWriteTokens
	}
	if len(m.history) > 0 {
		last = historyToUsage(m.history[len(m.history)-1])
	}
	m.contextUsage = total
	m.lastUsage = last
}

func historyToUsage(h data.History) *commontypes.TokenUsage {
	if h.PromptTokens == 0 && h.CompletionTokens == 0 && h.CacheReadTokens == 0 && h.CacheWriteTokens == 0 {
		return nil
	}
	return &commontypes.TokenUsage{
		PromptTokens:     h.PromptTokens,
		CompletionTokens: h.CompletionTokens,
		CacheReadTokens:  h.CacheReadTokens,
		CacheWriteTokens: h.CacheWriteTokens,
	}
}

func usageLines(u *commontypes.TokenUsage) []string {
	if u == nil {
		return []string{"Prompt: —", "Completion: —", "Cache read: —", "Cache write: —"}
	}
	return []string{
		fmt.Sprintf("Prompt: %d", u.PromptTokens),
		fmt.Sprintf("Completion: %d", u.CompletionTokens),
		fmt.Sprintf("Cache read: %d", u.CacheReadTokens),
		fmt.Sprintf("Cache write: %d", u.CacheWriteTokens),
	}
}

func (m *chatViewModel) renderUsagePanel() string {
	var b strings.Builder
	b.WriteString(usagePanelTitleStyle.Render("Token Usage"))
	b.WriteString("\n\n")
	b.WriteString(usageMetricLabelStyle.Render("Context Total"))
	b.WriteString("\n")
	for _, line := range usageLines(&m.contextUsage) {
		b.WriteString(usageMetricValueStyle.Render(line))
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(usageMetricLabelStyle.Render("Last Message"))
	b.WriteString("\n")
	for _, line := range usageLines(m.lastUsage) {
		b.WriteString(usageMetricValueStyle.Render(line))
		b.WriteString("\n")
	}
	b.WriteString("\n" + usageMetricLabelStyle.Render("Toggle: ctrl+t"))
	return strings.TrimSuffix(b.String(), "\n")
}

func (m *chatViewModel) scrollHalfPage(down bool) {
	amount := m.viewport.Height / 2
	if amount < 1 {
		amount = 1
	}
	yOffset := m.viewport.YOffset
	if down {
		yOffset += amount
	} else {
		yOffset -= amount
		if yOffset < 0 {
			yOffset = 0
		}
	}
	m.viewport.SetYOffset(yOffset)
}

func (m *chatViewModel) contentWidth() int {
	width := m.width
	if m.showUsagePanel {
		width -= usagePanelWidth + 2
	}
	if width < minContentWidth {
		width = minContentWidth
	}
	return width
}

func (m *chatViewModel) applyLayout() {
	contentWidth := m.contentWidth()
	m.viewport.Width = contentWidth
	viewportHeight := m.height - 10
	if viewportHeight < 0 {
		viewportHeight = 0
	}
	m.viewport.Height = viewportHeight
	textareaWidth := contentWidth - 4
	if textareaWidth < minTextareaWidth {
		textareaWidth = minTextareaWidth
	}
	m.textarea.SetWidth(textareaWidth)
}

func renderMarkdown(content string, width int) string {
	if width <= 0 {
		width = 80
	}
	renderer, err := glamour.NewTermRenderer(
		glamour.WithStandardStyle("dark"),
		glamour.WithColorProfile(termenv.TrueColor),
		glamour.WithWordWrap(width),
	)
	if err != nil {
		logger.Debug.Printf("failed to create markdown renderer: %v", err)
		return content
	}
	rendered, err := renderer.Render(content)
	if err != nil {
		logger.Debug.Printf("failed to render markdown: %v", err)
		return content
	}
	return strings.TrimSuffix(rendered, "\n")
}

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

func (h *tuiResponseHandler) FinalText(contextId int64, prompt string, response string, responseContent string, toolResults string, modelName string, usage *commontypes.TokenUsage) {
	h.fullResponse = response

	history := data.History{
		ContextId:       contextId,
		Prompt:          prompt,
		Response:        response,
		Abbreviation:    "",
		TokenCount:      0,
		ResponseContent: responseContent,
		ToolResults:     toolResults,
		Model:           modelName,
	}

	if usage != nil {
		history.PromptTokens = usage.PromptTokens
		history.CompletionTokens = usage.CompletionTokens
		history.CacheReadTokens = usage.CacheReadTokens
		history.CacheWriteTokens = usage.CacheWriteTokens
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

	logger.Debug.Println("Final text in tui response channel")
	if toolResults == "" {
		logger.Debug.Println("closing doneChan and responseChan")
		close(h.doneChan)
		close(h.responseChan)
	} else {
		logger.Debug.Println("not closing doneChan and responseChan because of expected response to tool call answers.")
		logger.Debug.Printf("contents of results: %v", toolResults)
	}
}
