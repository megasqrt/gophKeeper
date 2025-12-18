package tui

import (
	"github.com/charmbracelet/lipgloss"
)

const (
	hotPink  = lipgloss.Color("#FF06B7")
	darkGray = lipgloss.Color("#767676")
	green    = lipgloss.Color("10")
	red      = lipgloss.Color("9")
	blue     = lipgloss.Color("63")
	purple   = lipgloss.Color("170")
	accent   = lipgloss.Color("205")
)

var (
	// General styles
	docStyle     = lipgloss.NewStyle().Padding(1, 2, 1, 2)
	helpStyle    = lipgloss.NewStyle().Foreground(darkGray).Margin(1, 0, 0, 2)
	baseStyle    = lipgloss.NewStyle().BorderStyle(lipgloss.NormalBorder()).BorderForeground(darkGray)
	titleStyle   = lipgloss.NewStyle().MarginLeft(2).Bold(true).Foreground(blue)
	errorStyle   = lipgloss.NewStyle().Foreground(red)
	infoStyle    = lipgloss.NewStyle().Foreground(green)
	focusedStyle = lipgloss.NewStyle().Foreground(hotPink)
	blurredStyle = lipgloss.NewStyle().Foreground(darkGray)
	noStyle      = lipgloss.NewStyle()

	// List styles
	listTitleStyle       = titleStyle.Copy()
	itemStyle            = lipgloss.NewStyle().PaddingLeft(4)
	selectedItemStyle    = lipgloss.NewStyle().PaddingLeft(2).Foreground(purple)
	itemDescriptionStyle = lipgloss.NewStyle().Faint(true).PaddingLeft(2).Foreground(darkGray)
)

// Устаревшие константы, которые больше не используются
const DefaulListtWidth = 20
const DefaultlistHeight = 14
