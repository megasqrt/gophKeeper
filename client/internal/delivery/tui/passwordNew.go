package tui

import (
	"fmt"
	"gophKeeper/client/internal/domain/model"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type passFormSavedMsg struct {
	data map[string]string
}

type passFormBackMsg struct{}

type PassFormModel struct {
	focusIndex int
	inputs     []textinput.Model
	storage    LocalStorage
	passID     string // ID для редактируемой карты
}

func NewPassForm(storage LocalStorage, pass *model.Password) PassFormModel {
	m := PassFormModel{
		inputs:  make([]textinput.Model, 3),
		storage: storage,
	}

	var t textinput.Model
	for i := range m.inputs {
		t = textinput.New()
		t.Cursor.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
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

		m.inputs[i] = t
	}

	if pass != nil {
		m.passID = pass.ID
		m.inputs[0].SetValue(pass.Login)
		m.inputs[1].SetValue(pass.Password)
		m.inputs[2].SetValue(pass.Description)
	}

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
			if s == "enter" && m.focusIndex == len(m.inputs) {
				passData := map[string]string{
					"login": m.inputs[0].Value(),
					"password": m.inputs[1].Value(),
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

	// Обновляем только то поле, которое в фокусе
	var cmd tea.Cmd
	if m.focusIndex < len(m.inputs) {
		m.inputs[m.focusIndex], cmd = m.inputs[m.focusIndex].Update(msg)
	}

	return m, cmd
}

func (m PassFormModel) View() string {
	return fmt.Sprintf(
		` Total: $21.50:

 %s
 %s

 %s
 %s

 %s
 %s

 %s
`,
		inputStyle.Width(30).Render("Login"),
		m.inputs[0].View(),
		inputStyle.Width(30).Render("Password"),
		m.inputs[1].View(),
		inputStyle.Width(64).Render("Description"),
		m.inputs[2].View(),
		continueStyle.Render("Continue ->"),
	) + "\n"
}