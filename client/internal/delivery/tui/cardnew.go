package tui

import (
	"fmt"
	"gophKeeper/client/internal/domain"
	model "gophKeeper/pkg/grpchelper"
	"strings"

	"strconv"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type cardFormSavedMsg struct{}

type cardFormBackMsg struct{}

type CardFormModel struct {
	formModel
	storage domain.LocalStorage
	cardID  string // ID для редактируемой карты
	err     error  // Ошибка при сохранении
}

func NewCardForm(storage domain.LocalStorage, card *model.Card) CardFormModel {
	m := CardFormModel{
		storage:   storage,
		formModel: newFormModel(),
	}

	inputs := make([]textinput.Model, 4)
	for i := range inputs {
		t := textinput.New()
		t.Cursor.Style = focusedStyle
		t.CharLimit = 32
		t.Prompt = ""

		switch i {
		case 0:
			t.Placeholder = "4505 **** **** 1234"
			t.Focus()
			t.CharLimit = 20
			t.Validate = ccnValidator
			t.Width = 30
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

		inputs[i] = t
	}
	m.setInputs(inputs)

	if card != nil {
		m.cardID = card.LocalID
		m.inputs[0].SetValue(card.Number)
		m.inputs[1].SetValue(card.Expiry)
		m.inputs[2].SetValue(card.CVV)
		m.inputs[3].SetValue(card.Holder)
	}

	m.updateFocus()
	return m
}

func (m CardFormModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m CardFormModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case errMsg:
		m.err = msg
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			// Отправляем сообщение о возврате к списку
			return m, func() tea.Msg { return cardFormBackMsg{} }

		// Переключение фокуса
		case "tab", "shift+tab", "enter", "up", "down":
			s := msg.String()

			// Нажатие Enter на последнем поле или на кнопке "Submit"
			if s == "enter" && m.submitFocused() {
				cardData := &model.Card{
					LocalID: m.cardID,
					Number:  m.inputs[0].Value(),
					Expiry:  m.inputs[1].Value(),
					CVV:     m.inputs[2].Value(),
					Holder:  m.inputs[3].Value(),
				}

				var err error
				if m.cardID != "" { // Если есть ID, обновляем
					err = m.storage.UpdateCard(cardData)
				} else { // Иначе создаем новую
					err = m.storage.SaveCard(cardData)
				}

				if err != nil {
					m.err = err
					return m, nil
				}
				// Отправляем сообщение об успешном сохранении
				return m, func() tea.Msg { return cardFormSavedMsg{} }
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

func (m CardFormModel) View() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Credit Card Details"))
	b.WriteString("\n\n")

	// Поля ввода
	b.WriteString(focusedStyle.Render("Card Number") + "\n")
	b.WriteString(m.inputs[0].View() + "\n\n")

	expAndCvv := lipgloss.JoinHorizontal(lipgloss.Top,
		lipgloss.JoinVertical(lipgloss.Left,
			focusedStyle.Render("Expiry"),
			m.inputs[1].View(),
		),
		lipgloss.JoinVertical(lipgloss.Left,
			focusedStyle.Render("CVV"),
			m.inputs[2].View(),
		),
	)
	b.WriteString(expAndCvv + "\n\n")

	b.WriteString(focusedStyle.Render("Card Holder") + "\n")
	b.WriteString(m.inputs[3].View() + "\n\n")

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
