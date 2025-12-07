package tui

import (
	"gophKeeper/client/internal/domain"
	"gophKeeper/client/internal/domain/model"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type passFormSavedMsg struct {
	data map[string]string
}

type passFormBackMsg struct{}

type PassFormModel struct {
	formModel
	storage domain.LocalStorage
	passID  string // ID для редактируемой карты
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
		m.passID = pass.ID
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
				passData := map[string]string{
					"login":       m.inputs[0].Value(),
					"password":    m.inputs[1].Value(),
					"description": m.inputs[2].Value(),
				}
				var err error
				if m.passID != "" { // Если есть ID, обновляем
					passData["id"] = m.passID
					err = m.storage.UpdatePass(passData)
				} else { // Иначе создаем новую
					err = m.storage.SavePass(passData)
				}

				if err != nil {
					// TODO: обработать ошибку
				}
				// Отправляем сообщение об успешном сохранении
				return m, func() tea.Msg { return passFormSavedMsg{data: passData} }
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

	return docStyle.Render(b.String())
}
