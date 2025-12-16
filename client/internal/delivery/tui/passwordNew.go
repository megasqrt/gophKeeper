package tui

import (
	"gophKeeper/client/internal/domain"
	model "gophKeeper/pkg/grpchelper"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type passFormSavedMsg struct{}

type passFormBackMsg struct{}

type PassFormModel struct {
	formModel
	storage domain.LocalStorage
	passID  int64 // ID для редактируемой карты
	err     error  // Ошибка при сохранении
}

func NewPassForm(storage domain.LocalStorage, pass *model.Password) PassFormModel {
	m := PassFormModel{
		storage:   storage,
		formModel: newFormModel(),
	}

	inputs := make([]textinput.Model, 3)
	for i := range inputs {
		t := textinput.New()
		t.Cursor.Style = focusedStyle
		t.CharLimit = 32
		t.Prompt = ""

		switch i {
		case 0:
			t.Placeholder = "Login"
			t.Focus()
			t.CharLimit = 20
			t.Width = 30
		case 1:
			t.Placeholder = "Password"
			t.CharLimit = 20
			t.Width = 20
		case 2:
			t.Placeholder = "Description"
			t.CharLimit = 64
			t.Width = 64
		}

		inputs[i] = t
	}
	m.setInputs(inputs)

	if pass != nil {
		m.passID = pass.LocalID
		m.inputs[0].SetValue(pass.Login)
		m.inputs[1].SetValue(pass.Password)
		m.inputs[2].SetValue(pass.Description)
	}

	m.updateFocus()
	return m
}

func (m PassFormModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m PassFormModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case errMsg:
		m.err = msg
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			// Отправляем сообщение о возврате к списку
			return m, func() tea.Msg { return passFormBackMsg{} }

		// Переключение фокуса
		case "tab", "shift+tab", "enter", "up", "down":
			s := msg.String()

			// Нажатие Enter на последнем поле или на кнопке "Submit"
			if s == "enter" && m.submitFocused() {
				passData := &model.Password{
					LocalID:     m.passID,
					Login:       m.inputs[0].Value(),
					Password:    m.inputs[1].Value(),
					Description: m.inputs[2].Value(),
				}
				var err error
				
				if m.passID != 0 { // Если есть ID, обновляем
					err = m.storage.UpdatePass(passData)
				} else { // Иначе создаем новую
					err = m.storage.SavePass(passData)
				}

				if err != nil {
					m.err = err
					return m, nil
				}
				// Отправляем сообщение об успешном сохранении
				return m, func() tea.Msg { return passFormSavedMsg{} }
			}

			// Переключение фокуса
			if s == "up" || s == "shift+tab" {
				m.prevInput()
			} else {
				m.nextInput()
			}

			return m, m.updateFocus()
		}
	}

	// Обновляем только то поле, которое в фокусе
	var cmd tea.Cmd
	cmd = m.updateInputs(msg)
	return m, cmd
}

func (m PassFormModel) View() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Password Details"))
	b.WriteString("\n\n")

	b.WriteString(focusedStyle.Render("Login") + "\n")
	b.WriteString(m.inputs[0].View() + "\n\n")

	b.WriteString(focusedStyle.Render("Password") + "\n")
	b.WriteString(m.inputs[1].View() + "\n\n")

	b.WriteString(focusedStyle.Render("Description") + "\n")
	b.WriteString(m.inputs[2].View() + "\n\n")

	// Кнопка
	submitButton := blurredStyle.Render("[ Submit ]")
	if m.submitFocused() {
		submitButton = focusedStyle.Render("> [ Submit ]")
	}
	b.WriteString(submitButton)

	// Показываем ошибку, если есть
	if m.err != nil {
		errorStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("9")).
			Padding(0, 1)
		b.WriteString("\n\n" + errorStyle.Render("❌ Error: "+m.err.Error()))
	}

	return docStyle.Render(b.String())
}
