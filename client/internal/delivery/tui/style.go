package tui

import "github.com/charmbracelet/lipgloss"

var (
	// General styles
	docStyle   = lipgloss.NewStyle().Padding(1, 2, 1, 2)
	helpStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Margin(1, 0, 0, 2)
	baseStyle  = lipgloss.NewStyle().BorderStyle(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("240"))
	titleStyle = lipgloss.NewStyle().MarginLeft(2)

	// List styles
	listTitleStyle       = lipgloss.NewStyle().MarginLeft(2).Bold(true).Foreground(lipgloss.Color("63"))
	itemStyle            = lipgloss.NewStyle().PaddingLeft(4)
	selectedItemStyle    = lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("170"))
	itemDescriptionStyle = lipgloss.NewStyle().Faint(true).PaddingLeft(2)
)

// Устаревшие константы, которые больше не используются
const DefaulListtWidth = 20
const DefaultlistHeight = 14
