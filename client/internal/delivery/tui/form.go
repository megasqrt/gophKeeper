package tui

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// formModel — это базовая модель для форм ввода данных.
// Она управляет фокусом, навигацией и обновлением полей ввода.
type formModel struct {
	focusIndex int
	inputs     []textinput.Model
}

func newFormModel(inputs ...textinput.Model) formModel {
	return formModel{
		inputs: inputs,
	}
}

// updateFocus устанавливает фокус на input с текущим focusIndex.
func (m *formModel) updateFocus() tea.Cmd {
	cmds := make([]tea.Cmd, len(m.inputs))
	for i := 0; i < len(m.inputs); i++ {
		if i == m.focusIndex {
			cmds[i] = m.inputs[i].Focus()
			m.inputs[i].PromptStyle = focusedStyle
			m.inputs[i].TextStyle = focusedStyle
			continue
		}
		m.inputs[i].Blur()
		m.inputs[i].PromptStyle = noStyle
		m.inputs[i].TextStyle = noStyle
	}
	return tea.Batch(cmds...)
}

// updateInputs передает сообщение msg активному полю ввода.
func (m *formModel) updateInputs(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	if m.focusIndex < len(m.inputs) {
		m.inputs[m.focusIndex], cmd = m.inputs[m.focusIndex].Update(msg)
	}
	return cmd
}

// nextInput переключает фокус на следующее поле ввода.
func (m *formModel) nextInput() {
	m.focusIndex = (m.focusIndex + 1) % (len(m.inputs) + 1) // +1 для кнопки "Submit"
}

// prevInput переключает фокус на предыдущее поле ввода.
func (m *formModel) prevInput() {
	m.focusIndex--
	if m.focusIndex < 0 {
		m.focusIndex = len(m.inputs) // фокус на кнопке "Submit"
	}
}

// focused возвращает true, если фокус находится на полях ввода.
func (m *formModel) focused() bool {
	return m.focusIndex < len(m.inputs)
}

// submitFocused возвращает true, если фокус на воображаемой кнопке "Submit".
func (m *formModel) submitFocused() bool {
	return m.focusIndex == len(m.inputs)
}

// setInputs устанавливает поля ввода для модели.
func (m *formModel) setInputs(inputs []textinput.Model) {
	m.inputs = inputs
	if len(m.inputs) > 0 {
		m.focusIndex = 0
	}
}
