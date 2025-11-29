package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type cardFormSavedMsg struct {
	data map[string]string
}

type cardFormBackMsg struct{}

type CardFormModel struct {
	focusIndex int
	inputs     []textinput.Model
	storage    LocalStorage
}

func NewCardForm(storage LocalStorage) CardFormModel {
	m := CardFormModel{
		inputs:  make([]textinput.Model, 4),
		storage: storage,
	}

	var t textinput.Model
	for i := range m.inputs {
		t = textinput.New()
		t.Cursor.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
		t.CharLimit = 32

		switch i {
		case 0:
			t.Placeholder = "Card Number"
			t.Focus()
			t.PromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
			t.TextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
		case 1:
			t.Placeholder = "Holder Name"
			t.CharLimit = 64
		case 2:
			t.Placeholder = "MM/YY"
			t.CharLimit = 5
		case 3:
			t.Placeholder = "CVV"
			t.EchoMode = textinput.EchoPassword
			t.EchoCharacter = '•'
			t.CharLimit = 4
		}

		m.inputs[i] = t
	}

	return m
}

func (m CardFormModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m CardFormModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			// Отправляем сообщение о возврате к списку
			return m, func() tea.Msg { return cardFormBackMsg{} }

		// Переключение фокуса
		case "tab", "shift+tab", "enter", "up", "down":
			s := msg.String()

			// Нажатие Enter на последнем поле или на кнопке "Submit"
			if s == "enter" && m.focusIndex == len(m.inputs) {
				cardData := map[string]string{
					"number": m.inputs[0].Value(),
					"holder": m.inputs[1].Value(),
					"expiry": m.inputs[2].Value(),
					"cvv":    m.inputs[3].Value(),
				}
				// Сохраняем карту
				if err := m.storage.SaveCard(cardData); err != nil {
					// TODO: обработать ошибку сохранения
					return m, nil
				}
				// Отправляем сообщение об успешном сохранении
				return m, func() tea.Msg { return cardFormSavedMsg{data: cardData} }
			}

			// Переключение фокуса
			if s == "up" || s == "shift+tab" {
				m.focusIndex--
			} else {
				m.focusIndex++
			}

			if m.focusIndex > len(m.inputs) {
				m.focusIndex = 0
			} else if m.focusIndex < 0 {
				m.focusIndex = len(m.inputs)
			}

			cmds := make([]tea.Cmd, len(m.inputs))
			for i := 0; i <= len(m.inputs)-1; i++ {
				if i == m.focusIndex {
					cmds[i] = m.inputs[i].Focus()
					m.inputs[i].PromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
					m.inputs[i].TextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
					continue
				}
				m.inputs[i].Blur()
				m.inputs[i].PromptStyle = lipgloss.NewStyle()
				m.inputs[i].TextStyle = lipgloss.NewStyle()
			}

			return m, tea.Batch(cmds...)
		}
	}

	// Обновляем поле ввода, которое в фокусе
	cmd := m.updateInputs(msg)

	return m, cmd
}

func (m *CardFormModel) updateInputs(msg tea.Msg) tea.Cmd {
	cmds := make([]tea.Cmd, len(m.inputs))

	for i := range m.inputs {
		m.inputs[i], cmds[i] = m.inputs[i].Update(msg)
	}

	return tea.Batch(cmds...)
}

func (m CardFormModel) View() string {
	var b strings.Builder

	b.WriteString("Enter New Credit Card Details\n\n")

	for i := range m.inputs {
		b.WriteString(m.inputs[i].View())
		if i < len(m.inputs)-1 {
			b.WriteRune('\n')
		}
	}

	button := "\n\n[ Submit ]"
	if m.focusIndex == len(m.inputs) {
		button = "\n\n> [ Submit ]"
	}

	b.WriteString(button)

	b.WriteString(fmt.Sprintf("\n\n%s", helpStyle.Render("(esc) back to list | (s) save")))

	return b.String()
}
