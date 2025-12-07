package tui

import (
	"gophKeeper/client/internal/domain"
	"gophKeeper/client/internal/domain/model"
	"time"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type viewState int

const (
	tableView viewState = iota
	formView
)

type backToMenuMsg struct{}

// CardListModel управляет состоянием вкладки "Card", теперь используя таблицу.
type CardListModel struct {
	state   viewState
	table   table.Model
	form    CardFormModel
	storage domain.LocalStorage
	cards   []model.Card // Добавляем поле для хранения полных данных карт
}

func NewCardListModel(storage domain.LocalStorage) *CardListModel {
	columns := []table.Column{
		{Title: "Card Number", Width: 20},
		{Title: "Holder", Width: 25},
		{Title: "Expiry", Width: 10},
	}

	tbl := table.New(
		table.WithColumns(columns),
		table.WithRows([]table.Row{}),
		table.WithFocused(true),
		table.WithHeight(10),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(false)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)
	tbl.SetStyles(s)

	m := &CardListModel{
		state:   tableView,
		storage: storage,
		form:    NewCardForm(storage, nil),
		table:   tbl,
	}

	return m
}

// Load загружает карты из хранилища и обновляет строки таблицы.
func (m *CardListModel) Load() {
	rows, cards := m.loadCards()
	m.cards = cards
	m.table.SetRows(rows)
}

func (m *CardListModel) loadCards() ([]table.Row, []model.Card) {
	cardsData, err := m.storage.GetCards()
	if err != nil {
		//	m.log.Error().Err(err).Msg("get cards error")
		return []table.Row{}, []model.Card{}
	}

	rows := make([]table.Row, len(cardsData))
	cards := make([]model.Card, len(cardsData))
	for i, data := range cardsData {
		card := model.Card{
			ID:     data["id"],
			Number: data["number"],
			Holder: data["holder"],
			Expiry: data["expiry"],
			CVV:    data["cvv"],
		}
		if changeTimeStr, ok := data["changeTime"]; ok {
			card.ChangeTime, _ = time.Parse(time.RFC3339Nano, changeTimeStr)
		}
		if syncTimeStr, ok := data["syncTime"]; ok {
			card.SyncTime, _ = time.Parse(time.RFC3339Nano, syncTimeStr)
		}

		// Используем Title() и Description() из модели карты для консистентности
		cards[i] = card
		rows[i] = table.Row{card.Title(), card.Description(), card.Expiry}
	}
	return rows, cards
}

func (m *CardListModel) Init() tea.Cmd {
	return nil
}

func (m *CardListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg.(type) {
	case cardFormBackMsg:
		m.state = tableView
		return m, nil
	case cardFormSavedMsg:
		m.Load() // Просто перезагружаем данные в таблицу
		m.state = tableView
		return m, nil
	}

	if m.state == formView {
		var newForm tea.Model
		newForm, cmd = m.form.Update(msg)
		m.form = newForm.(CardFormModel)
		return m, cmd
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return m, func() tea.Msg { return backToMenuMsg{} }
		case "a": // 'a' for "add"
			m.state = formView
			m.form = NewCardForm(m.storage, nil) // Создаем новую чистую форму
			return m, m.form.Init()
		case "e", "enter":
			if len(m.cards) == 0 {
				return m, nil
			}
			selectedCard := m.cards[m.table.Cursor()]
			m.state = formView
			m.form = NewCardForm(m.storage, &selectedCard) // Передаем выбранную карту в форму
			return m, m.form.Init()
		}
	}

	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m *CardListModel) View() string {
	if m.state == formView {
		return m.form.View()
	}

	help := helpStyle.Render("(↑/↓) navigate | (a) add new card | (esc) back to menu")
	return baseStyle.Render(m.table.View()) + "\n" + help
}
