package tui

import "github.com/charmbracelet/lipgloss"

var (
	primaryColor   = lipgloss.Color("63")
	secondaryColor = lipgloss.Color("240")
	accentColor    = lipgloss.Color("205")
	errorColor     = lipgloss.Color("196")

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
			Foreground(secondaryColor).
			Italic(true).
			MarginTop(1)

	loadingStyle = lipgloss.NewStyle().
			Foreground(primaryColor).
			Bold(true)

	errorStyle = lipgloss.NewStyle().
			Foreground(errorColor).
			Bold(true)

	userPromptStyle = lipgloss.NewStyle().
			Foreground(accentColor).
			Bold(true)

	aiResponseStyle = lipgloss.NewStyle().
			PaddingLeft(2)

	sendingStyle = lipgloss.NewStyle().
			Foreground(primaryColor).
			Bold(true).
			Blink(true)
)
