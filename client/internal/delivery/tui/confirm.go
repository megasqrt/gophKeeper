package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	dialogBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#874BFD")).
			Padding(1, 2)

	buttonStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFF7DB")).
			Background(lipgloss.Color("#5A5A5A")).
			Padding(0, 3).
			MarginTop(1)

	activeButtonStyle = buttonStyle.Copy().
				Background(lipgloss.Color("#874BFD"))
)

type confirmMsg struct {
	Confirmed bool
}

type ConfirmModel struct {
	prompt     string
	focusIndex int
	width      int
	height     int
	callback   func(bool) tea.Cmd
}

func NewConfirmModel(prompt string, callback func(bool) tea.Cmd) ConfirmModel {
	return ConfirmModel{
		prompt:     prompt,
		callback:   callback,
		focusIndex: 0, // "Да" по умолчанию
	}
}

func (m *ConfirmModel) setSize(w, h int) {
	m.width = w
	m.height = h
}

func (m ConfirmModel) Init() tea.Cmd {
	return nil
}

func (m ConfirmModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "right", "l", "tab":
			m.focusIndex = (m.focusIndex + 1) % 2
		case "left", "h", "shift+tab":
			m.focusIndex--
			if m.focusIndex < 0 {
				m.focusIndex = 1
			}
		case "enter":
			return m, m.callback(m.focusIndex == 0) // 0 is "Да"
		case "esc", "q", "ctrl+c":
			return m, m.callback(false)
		}
	}
	return m, nil
}

func (m ConfirmModel) View() string {
	var yesButton, noButton string

	if m.focusIndex == 0 {
		yesButton = activeButtonStyle.Render("Да")
		noButton = buttonStyle.Render("Нет")
	} else {
		yesButton = buttonStyle.Render("Да")
		noButton = activeButtonStyle.Render("Нет")
	}

	question := lipgloss.NewStyle().Width(50).Align(lipgloss.Center).Render(m.prompt)
	buttons := lipgloss.JoinHorizontal(lipgloss.Center, yesButton, lipgloss.NewStyle().Width(2).Render(""), noButton)

	ui := lipgloss.JoinVertical(lipgloss.Center, question, buttons)

	dialog := dialogBoxStyle.Render(ui)

	// Рассчитываем позицию для центрирования
	if m.width > 0 {
		return lipgloss.Place(m.width, m.height,
			lipgloss.Center, lipgloss.Center,
			dialog,
		)
	}
	return dialog
}

func (m ConfirmModel) GetPrompt() string {
	return m.prompt
}

func (m *ConfirmModel) SetPrompt(prompt string) {
	m.prompt = fmt.Sprintf("Вы точно хотите удалить %s?", prompt)
}
