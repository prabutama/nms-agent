package tui

import "github.com/charmbracelet/lipgloss"

var (
	styleTitle = lipgloss.NewStyle().Bold(true)

	styleDim    = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	styleOK     = lipgloss.NewStyle().Foreground(lipgloss.Color("82"))
	styleWarn   = lipgloss.NewStyle().Foreground(lipgloss.Color("220"))
	styleCrit   = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	styleHeader = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15")).Background(lipgloss.Color("33")).Padding(0, 1)

	styleTableHeader = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39"))

	styleFooter = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))

	styleSelected = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212")).Background(lipgloss.Color("235"))

	styleDivider = lipgloss.NewStyle().Foreground(lipgloss.Color("238"))
)
