package tui

import "github.com/charmbracelet/lipgloss"

var (
	primaryColor   = lipgloss.Color("63")
	secondaryColor = lipgloss.Color("240")
	panelColor     = lipgloss.Color("#242424")
	accentColor    = lipgloss.Color("205")
	errorColor     = lipgloss.Color("196")
	panelBorder    = lipgloss.RoundedBorder()

	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(primaryColor).
			MarginBottom(1)

	itemStyle = lipgloss.NewStyle().
			PaddingLeft(2)

	selectedItemStyle = lipgloss.NewStyle().
				PaddingLeft(1).
				Foreground(accentColor).
				Bold(true)

	dimStyle = lipgloss.NewStyle().
			Foreground(secondaryColor)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#a0a0a0")).
			Italic(true).
			MarginTop(1)

	loadingStyle = lipgloss.NewStyle().
			Foreground(primaryColor).
			Bold(true)

	errorStyle = lipgloss.NewStyle().
			Foreground(errorColor).
			Bold(true)

	userPromptStyle = lipgloss.NewStyle().
			PaddingLeft(2).
			Foreground(accentColor).
			Bold(true)

	aiResponseStyle = lipgloss.NewStyle().
		// Background(panelColor).
		PaddingLeft(2)

	sendingStyle = lipgloss.NewStyle().
			Foreground(primaryColor).
			Bold(true).
			Blink(true)

	usagePanelStyle = lipgloss.NewStyle().
			Border(panelBorder, true).
			BorderForeground(secondaryColor).
			Padding(1, 2).
			MarginLeft(2)

	usagePanelTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(primaryColor)

	usageMetricLabelStyle = lipgloss.NewStyle().
				Foreground(secondaryColor)

	usageMetricValueStyle = lipgloss.NewStyle().
				Bold(true)
)
