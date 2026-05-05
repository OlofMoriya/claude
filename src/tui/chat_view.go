package tui

import (
	"fmt"
	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
	commontypes "owl/common_types"
	"owl/data"
	"owl/interaction"
	"owl/logger"
	picker "owl/picker"
	"owl/services"
	"owl/tools"
	"strings"
	"time"
)

type chatMode int

const (
	chatInputMode chatMode = iota
	chatNormalMode
	chatModelSelectMode
	chatQuestionMode
)

type questionAnswerState struct {
	selectedOption  int
	selectedOptions map[int]bool
	customAnswer    string
}

type questionPromptState struct {
	prompt      interaction.QuestionPrompt
	title       string
	questionIdx int
	optionIdx   int
	answers     []questionAnswerState
	customInput textinput.Model
	editingText bool
}

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
	statusVersion    int
	showUsagePanel   bool
	usagePanelPinned bool
	contextUsage     commontypes.TokenUsage
	lastUsage        *commontypes.TokenUsage
	questionPrompt   *questionPromptState
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
type clearStatusMsg struct {
	version int
}

type historyPersistedMsg int64

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

type questionPromptMsg struct {
	prompt interaction.QuestionPrompt
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
		m.listenForHistoryPersisted(),
		m.listenForQuestionPrompts(),
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

func (m *chatViewModel) listenForHistoryPersisted() tea.Cmd {
	return func() tea.Msg {
		if logger.HistoryPersistedChan == nil {
			return nil
		}

		contextId, ok := <-logger.HistoryPersistedChan
		if !ok {
			return nil
		}

		return historyPersistedMsg(contextId)
	}
}

func (m *chatViewModel) listenForQuestionPrompts() tea.Cmd {
	return func() tea.Msg {
		if interaction.QuestionPromptChan == nil {
			return nil
		}

		prompt, ok := <-interaction.QuestionPromptChan
		if !ok {
			return nil
		}

		return questionPromptMsg{prompt: prompt}
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
		cleaned := strings.TrimSpace(string(msg))
		cleaned = strings.ReplaceAll(cleaned, "\n", " ")
		cleaned = strings.Join(strings.Fields(cleaned), " ")

		if cleaned == "" {
			m.statusMessage = ""
			return m, m.listenForStatus()
		}

		m.statusMessage = cleaned
		m.statusVersion++
		currentVersion := m.statusVersion
		logger.Debug.Printf("Received status: %s", cleaned)

		return m, tea.Batch(
			m.listenForStatus(),
			m.clearStatusAfterDelay(currentVersion),
		)

	case clearStatusMsg:
		if msg.version == m.statusVersion {
			m.statusMessage = ""
		}
		return m, nil

	case historyPersistedMsg:
		if int64(msg) == m.shared.selectedCtx.Id {
			m.currentResponse = ""
			return m, tea.Batch(
				m.listenForHistoryPersisted(),
				m.loadHistory(),
			)
		}
		return m, m.listenForHistoryPersisted()

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

	case questionPromptMsg:
		state := newQuestionPromptState(msg.prompt)
		m.questionPrompt = &state
		m.mode = chatQuestionMode
		return m, m.listenForQuestionPrompts()

	case tea.KeyMsg:
		if m.mode == chatQuestionMode {
			return m, m.handleQuestionModeKey(msg)
		}

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

func (m *chatViewModel) clearStatusAfterDelay(version int) tea.Cmd {
	return tea.Tick(3*time.Second, func(t time.Time) tea.Msg {
		return clearStatusMsg{version: version}
	})
}

func newQuestionPromptState(prompt interaction.QuestionPrompt) questionPromptState {
	title := strings.TrimSpace(prompt.Request.Title)
	if title == "" {
		title = "Answer Questions"
	}

	answers := make([]questionAnswerState, len(prompt.Request.Questions))
	for i := range answers {
		answers[i] = questionAnswerState{selectedOption: -1, selectedOptions: map[int]bool{}}
	}

	ti := textinput.New()
	ti.Placeholder = "Type custom answer"
	ti.Prompt = "> "
	ti.CharLimit = 400

	return questionPromptState{
		prompt:      prompt,
		title:       title,
		questionIdx: 0,
		optionIdx:   0,
		answers:     answers,
		customInput: ti,
		editingText: false,
	}
}

func (m *chatViewModel) handleQuestionModeKey(msg tea.KeyMsg) tea.Cmd {
	if m.questionPrompt == nil {
		m.mode = chatInputMode
		return nil
	}

	qs := m.questionPrompt
	question := qs.prompt.Request.Questions[qs.questionIdx]

	if qs.editingText {
		switch msg.String() {
		case "esc":
			qs.editingText = false
			qs.customInput.Blur()
			return nil
		case "enter":
			qs.answers[qs.questionIdx].customAnswer = strings.TrimSpace(qs.customInput.Value())
			qs.editingText = false
			qs.customInput.Blur()
			return nil
		default:
			var cmd tea.Cmd
			qs.customInput, cmd = qs.customInput.Update(msg)
			return cmd
		}
	}

	switch msg.String() {
	case "j", "down":
		if qs.optionIdx < len(question.Options)-1 {
			qs.optionIdx++
		}
	case "k", "up":
		if qs.optionIdx > 0 {
			qs.optionIdx--
		}
	case "tab", "l", "right":
		if qs.questionIdx < len(qs.prompt.Request.Questions)-1 {
			qs.questionIdx++
			qs.optionIdx = 0
		}
	case "shift+tab", "h", "left":
		if qs.questionIdx > 0 {
			qs.questionIdx--
			qs.optionIdx = 0
		}
	case "enter":
		if len(question.Options) > 0 {
			if question.AllowMultiple {
				if qs.answers[qs.questionIdx].selectedOptions == nil {
					qs.answers[qs.questionIdx].selectedOptions = map[int]bool{}
				}
				qs.answers[qs.questionIdx].selectedOptions[qs.optionIdx] = !qs.answers[qs.questionIdx].selectedOptions[qs.optionIdx]
			} else {
				qs.answers[qs.questionIdx].selectedOption = qs.optionIdx
			}
		}
	case " ":
		if question.AllowMultiple && len(question.Options) > 0 {
			if qs.answers[qs.questionIdx].selectedOptions == nil {
				qs.answers[qs.questionIdx].selectedOptions = map[int]bool{}
			}
			qs.answers[qs.questionIdx].selectedOptions[qs.optionIdx] = !qs.answers[qs.questionIdx].selectedOptions[qs.optionIdx]
		}
	case "i":
		if question.AllowCustom {
			qs.editingText = true
			qs.customInput.SetValue(qs.answers[qs.questionIdx].customAnswer)
			qs.customInput.Focus()
		}
	case "x":
		qs.answers[qs.questionIdx] = questionAnswerState{selectedOption: -1, selectedOptions: map[int]bool{}}
	case "esc":
		qs.prompt.ResponseChan <- interaction.QuestionPromptResult{Err: fmt.Errorf("questionnaire canceled")}
		m.questionPrompt = nil
		m.mode = chatInputMode
	case "ctrl+w":
		response, err := buildQuestionBatchResponse(qs)
		if err != nil {
			m.statusMessage = err.Error()
			m.statusVersion++
			return m.clearStatusAfterDelay(m.statusVersion)
		}
		qs.prompt.ResponseChan <- interaction.QuestionPromptResult{Response: response}
		m.questionPrompt = nil
		m.mode = chatInputMode
	}

	return nil
}

func buildQuestionBatchResponse(state *questionPromptState) (*commontypes.QuestionBatchResponse, error) {
	answers := make([]commontypes.QuestionAnswer, 0, len(state.prompt.Request.Questions))

	for idx, q := range state.prompt.Request.Questions {
		a := state.answers[idx]
		custom := strings.TrimSpace(a.customAnswer)
		selectedIdx := a.selectedOption
		selectedLabel := ""
		selectedIndexes := []int{}
		selectedLabels := []string{}

		if q.AllowMultiple {
			for i := range q.Options {
				if a.selectedOptions != nil && a.selectedOptions[i] {
					selectedIndexes = append(selectedIndexes, i)
					selectedLabels = append(selectedLabels, q.Options[i].Label)
				}
			}
			if len(selectedLabels) > 0 {
				selectedLabel = strings.Join(selectedLabels, ", ")
			}
		} else if selectedIdx >= 0 && selectedIdx < len(q.Options) {
			selectedLabel = q.Options[selectedIdx].Label
			selectedIndexes = append(selectedIndexes, selectedIdx)
			selectedLabels = append(selectedLabels, selectedLabel)
		}

		if q.Required && selectedLabel == "" && custom == "" {
			return nil, fmt.Errorf("question %d is required", idx+1)
		}

		finalAnswer := selectedLabel
		if custom != "" {
			finalAnswer = custom
		}

		answer := commontypes.QuestionAnswer{
			ID:                    q.ID,
			SelectedOptionLabel:   selectedLabel,
			SelectedOptionIndexes: selectedIndexes,
			SelectedOptionLabels:  selectedLabels,
			CustomAnswer:          custom,
			FinalAnswer:           finalAnswer,
		}
		if selectedLabel != "" && selectedIdx >= 0 {
			answer.SelectedOptionIndex = &selectedIdx
		}
		answers = append(answers, answer)
	}

	return &commontypes.QuestionBatchResponse{Answers: answers}, nil
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

	if m.mode == chatQuestionMode {
		return m.renderQuestionPrompt()
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

func (m *chatViewModel) renderQuestionPrompt() string {
	if m.questionPrompt == nil {
		return loadingStyle.Render("Loading questions...")
	}

	qs := m.questionPrompt
	q := qs.prompt.Request.Questions[qs.questionIdx]

	var b strings.Builder
	b.WriteString(headerStyle.Render("? "+qs.title) + "\n\n")
	b.WriteString(dimStyle.Render(fmt.Sprintf("Question %d/%d", qs.questionIdx+1, len(qs.prompt.Request.Questions))) + "\n\n")
	b.WriteString(userPromptStyle.Render(q.Question) + "\n\n")

	for i, opt := range q.Options {
		cursor := " "
		if i == qs.optionIdx {
			cursor = ">"
		}
		selected := " "
		if q.AllowMultiple {
			if qs.answers[qs.questionIdx].selectedOptions != nil && qs.answers[qs.questionIdx].selectedOptions[i] {
				selected = "x"
			}
		} else if qs.answers[qs.questionIdx].selectedOption == i {
			selected = "x"
		}
		b.WriteString(fmt.Sprintf("%s [%s] %s\n", cursor, selected, opt.Label))
	}

	if q.AllowCustom {
		b.WriteString("\n")
		b.WriteString(dimStyle.Render("Custom answer") + "\n")
		if qs.editingText {
			b.WriteString(qs.customInput.View() + "\n")
		} else {
			custom := strings.TrimSpace(qs.answers[qs.questionIdx].customAnswer)
			if custom == "" {
				custom = "(none)"
			}
			b.WriteString(dimStyle.Render(custom) + "\n")
		}
	}

	b.WriteString("\n")
	selectHint := "enter: select"
	if q.AllowMultiple {
		selectHint = "enter/space: toggle"
	}
	b.WriteString(helpStyle.Render("j/k: option  tab/shift+tab: question  " + selectHint + "  i: custom  x: clear  ctrl+w: submit  esc: cancel"))

	return b.String()
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
		hasPrompt := strings.TrimSpace(h.Prompt) != ""

		if h.Archived {
			pStyle = dimStyle
			rStyle = dimStyle
			archivedPrefix = "[ARCHIVED] "
		}

		if hasPrompt {
			b.WriteString(pStyle.Render(fmt.Sprintf("%sYou: %s", archivedPrefix, h.Prompt)))
			b.WriteString("\n\n")
		}

		rendered := renderMarkdown(h.Response, m.viewport.Width-4)
		b.WriteString(rStyle.Render(rendered))
		if len(h.ToolUse) > 0 {
			b.WriteString("\n")
			b.WriteString(dimStyle.Render(renderToolUseSummary(h.ToolUse)))
		}
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

func renderToolUseSummary(toolUses []data.ToolUse) string {
	if len(toolUses) == 0 {
		return ""
	}

	failed := 0
	for _, tu := range toolUses {
		if !tu.Result.Success {
			failed++
		}
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("Tools: %d call(s)", len(toolUses)))
	if failed > 0 {
		b.WriteString(fmt.Sprintf(" (%d failed)", failed))
	}

	maxShown := 3
	for i, tu := range toolUses {
		if i >= maxShown {
			b.WriteString(fmt.Sprintf("\n- ... and %d more", len(toolUses)-maxShown))
			break
		}

		lines := tools.FormatToolUseForDisplay(tu)
		if len(lines) == 0 {
			continue
		}

		b.WriteString(fmt.Sprintf("\n- %s", lines[0]))
		for _, line := range lines[1:] {
			b.WriteString(fmt.Sprintf("\n  %s", line))
		}
	}

	return b.String()
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

func (h *tuiResponseHandler) FinalText(contextId int64, prompt string, response string, toolUse []data.ToolUse, modelName string, usage *commontypes.TokenUsage) {
	h.fullResponse = response

	history := data.History{
		ContextId:    contextId,
		Prompt:       prompt,
		Response:     response,
		Abbreviation: "",
		TokenCount:   0,
		Model:        modelName,
		ToolUse:      toolUse,
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
	} else {
		logger.HistoryPersisted(contextId)
	}

	code := services.ExtractCodeBlocks(response)
	allCode := strings.Join(code, "\n\n")

	err = clipboard.WriteAll(allCode)
	if err != nil {
		fmt.Printf("Error copying to clipboard: %v\n", err)
	}

	logger.Debug.Println("Final text in tui response channel")
	if len(toolUse) == 0 {
		logger.Debug.Println("closing doneChan and responseChan")
		close(h.doneChan)
		close(h.responseChan)
	} else {
		logger.Debug.Println("not closing doneChan and responseChan because of expected response to tool call answers.")
	}
}
