package tui

import (
	"fmt"
	"strings"
	"gophKeeper/client/internal/domain/model"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"strconv"
)

const (
	hotPink  = lipgloss.Color("#FF06B7")
	darkGray = lipgloss.Color("#767676")
)

var (
	inputStyle    = lipgloss.NewStyle().Foreground(hotPink)
	continueStyle = lipgloss.NewStyle().Foreground(darkGray)
)

type cardFormSavedMsg struct {
	data map[string]string
}

type cardFormBackMsg struct{}

type CardFormModel struct {
	focusIndex int
	inputs     []textinput.Model
	storage    LocalStorage
	cardID     string // ID для редактируемой карты
}

func NewCardForm(storage LocalStorage, card *model.Card) CardFormModel {
	m := CardFormModel{
		inputs:  make([]textinput.Model, 4),
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
			t.Placeholder = "4505 **** **** 1234"
			t.Focus()
			t.CharLimit = 20
			t.Validate = ccnValidator
			t.Width = 30
			//t.PromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
			//t.TextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
		case 1:
			t.Placeholder = "MM/YY "
			t.CharLimit = 5
			t.Width = 5
			t.Validate = expValidator
		case 2:
			t.Placeholder = "CVV"
			t.CharLimit = 5
			t.CharLimit = 3
			t.Width = 5
			t.Validate = cvvValidator
		case 3:
			t.Placeholder = "Владелец карты"
			t.CharLimit = 64
		}

		m.inputs[i] = t
	}

	if card != nil {
		m.cardID = card.ID
		m.inputs[0].SetValue(card.Number)
		m.inputs[1].SetValue(card.Expiry)
		m.inputs[2].SetValue(card.CVV)
		m.inputs[3].SetValue(card.Holder)
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
					"holder": m.inputs[3].Value(),
					"expiry": m.inputs[1].Value(),
					"cvv":    m.inputs[2].Value(),
				}
				var err error
				if m.cardID != "" { // Если есть ID, обновляем
					cardData["id"] = m.cardID
					err = m.storage.UpdateCard(cardData)
				} else { // Иначе создаем новую
					err = m.storage.SaveCard(cardData)
				}

				if err != nil {
					// TODO: обработать ошибку
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

	// Обновляем только то поле, которое в фокусе
	var cmd tea.Cmd
	if m.focusIndex < len(m.inputs) {
		m.inputs[m.focusIndex], cmd = m.inputs[m.focusIndex].Update(msg)
	}

	return m, cmd
}

// func (m CardFormModel) View() string {
// 	var b strings.Builder

// 	b.WriteString("Enter New Credit Card Details\n\n")

// 	for i := range m.inputs {
// 		b.WriteString(m.inputs[i].View())
// 		if i < len(m.inputs)-1 {
// 			b.WriteRune('\n')
// 		}
// 	}

// 	button := "\n\n[ Submit ]"
// 	if m.focusIndex == len(m.inputs) {
// 		button = "\n\n> [ Submit ]"
// 	}

// 	b.WriteString(button)

// 	b.WriteString(fmt.Sprintf("\n\n%s", helpStyle.Render("(esc) back to list | (s) save")))

// 	return b.String()
// }

func (m CardFormModel) View() string {
	return fmt.Sprintf(
		` Total: $21.50:

 %s
 %s

 %s  %s
 %s  %s

 %s
 %s

 %s
`,
		inputStyle.Width(30).Render("Card Number"),
		m.inputs[0].View(),
		inputStyle.Width(6).Render("EXP"),
		inputStyle.Width(6).Render("CVV"),
		m.inputs[1].View(),
		m.inputs[2].View(),
		inputStyle.Width(64).Render("Card Holder"),
		m.inputs[3].View(),
		continueStyle.Render("Continue ->"),
	) + "\n"
}
func ccnValidator(s string) error {
	// Credit Card Number should a string less than 20 digits
	// It should include 16 integers and 3 spaces
	if len(s) > 16+3 {
		return fmt.Errorf("CCN is too long")
	}

	if len(s) == 0 || len(s)%5 != 0 && (s[len(s)-1] < '0' || s[len(s)-1] > '9') {
		return fmt.Errorf("CCN is invalid")
	}

	// The last digit should be a number unless it is a multiple of 4 in which
	// case it should be a space
	if len(s)%5 == 0 && s[len(s)-1] != ' ' {
		return fmt.Errorf("CCN must separate groups with spaces")
	}

	// The remaining digits should be integers
	c := strings.ReplaceAll(s, " ", "")
	_, err := strconv.ParseInt(c, 10, 64)

	return err
}

func expValidator(s string) error {
	// The 3 character should be a slash (/)
	// The rest should be numbers
	e := strings.ReplaceAll(s, "/", "")
	_, err := strconv.ParseInt(e, 10, 64)
	if err != nil {
		return fmt.Errorf("EXP is invalid")
	}

	// There should be only one slash and it should be in the 2nd index (3rd character)
	if len(s) >= 3 && (strings.Index(s, "/") != 2 || strings.LastIndex(s, "/") != 2) {
		return fmt.Errorf("EXP is invalid")
	}

	return nil
}

func cvvValidator(s string) error {
	// The CVV should be a number of 3 digits
	// Since the input will already ensure that the CVV is a string of length 3,
	// All we need to do is check that it is a number
	_, err := strconv.ParseInt(s, 10, 64)
	return err
}