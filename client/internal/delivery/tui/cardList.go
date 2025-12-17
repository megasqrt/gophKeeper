package tui

import (
	"fmt"
	model "gophKeeper/pkg/grpchelper"

	"gophKeeper/client/internal/domain"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/rs/zerolog"
)

type viewState int

const (
	tableView viewState = iota
	formView
	confirmDeleteView
)

type backToMenuMsg struct{}
type deleteCardMsg struct{ confirmed bool }

// CardListModel управляет состоянием вкладки "Card", теперь используя таблицу.
type CardListModel struct {
	state        viewState
	table        table.Model
	form         CardFormModel
	confirmModel ConfirmModel
	storage      domain.LocalStorage
	cards        []model.Card // Добавляем поле для хранения полных данных карт
	width        int
	height       int
	log          *zerolog.Logger
}

func NewCardListModel(storage domain.LocalStorage, log *zerolog.Logger) *CardListModel {
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
		log:     log,
	}
	m.confirmModel = NewConfirmModel("Default prompt", func(confirmed bool) tea.Cmd {
		return func() tea.Msg {
			return deleteCardMsg{confirmed: confirmed}
		}
	})

	return m
}

// Load загружает карты из хранилища и обновляет строки таблицы.
func (m *CardListModel) Load() {
	rows, cards := m.loadCards()
	m.cards = cards
	m.table.SetRows(rows)
}

func (m *CardListModel) loadCards() ([]table.Row, []model.Card) {
	allCards, err := m.storage.GetCards()
	if err != nil {
		//	m.log.Error().Err(err).Msg("get cards error")
		return []table.Row{}, []model.Card{}
	}

	var rows []table.Row
	var displayedCards []model.Card
	for _, card := range allCards {
		if !card.Deleted {
			rows = append(rows, table.Row{card.Title(), card.Description(), card.Expiry})
			displayedCards = append(displayedCards, card)
		}
	}
	return rows, displayedCards // Возвращаем только отображаемые карты
}

func (m *CardListModel) Init() tea.Cmd {
	return nil
}

func (m *CardListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	// First, handle messages that change state or come from commands
	case deleteCardMsg:
		m.state = tableView // Exit confirmation view
		if msg.confirmed {
			if len(m.cards) > 0 {
				selectedCard := m.cards[m.table.Cursor()]
				err := m.storage.DeleteCard(selectedCard.GetLocalID())
				if err != nil {
				}
				m.Load()
			}
		}
		return m, nil

	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.confirmModel.setSize(msg.Width, msg.Height)

	case cardFormBackMsg:
		m.state = tableView
		return m, nil

	case cardFormSavedMsg:
		m.Load() // Just reload the data in the table
		m.state = tableView
		return m, nil
	}

	// Then, delegate to sub-models based on the current state
	if m.state == confirmDeleteView {
		// confirmModel is updated in-place due to pointer receiver in its Update method.
		// We just need the command.
		_, cmd = m.confirmModel.Update(msg)
		return m, cmd
	}

	if m.state == formView {
		var newForm tea.Model
		newForm, cmd = m.form.Update(msg)
		m.form = newForm.(CardFormModel)
		return m, cmd
	}

	// Handle key presses for the main table view
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return m, func() tea.Msg { return backToMenuMsg{} }
		case "a": // 'a' for "add"
			m.state = formView
			m.form = NewCardForm(m.storage, nil) // Create a new clean form
			return m, m.form.Init()
		case "e", "enter":
			if len(m.cards) == 0 {
				return m, nil
			}
			selectedCard := m.cards[m.table.Cursor()]
			m.state = formView
			m.form = NewCardForm(m.storage, &selectedCard) // Pass the selected card to the form
			return m, m.form.Init()
		case "ctrl+d":
			if m.state == tableView && len(m.cards) > 0 {
				selectedCard := m.cards[m.table.Cursor()]
				m.confirmModel.SetPrompt(fmt.Sprintf("карту '%s'", selectedCard.Title()))
				m.state = confirmDeleteView
				return m, nil
			}
		}
	}

	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m *CardListModel) View() string {
	if m.state == confirmDeleteView {
		return m.confirmModel.View()
	}
	if m.state == formView {
		return m.form.View()
	}

	help := helpStyle.Render("(↑/↓) navigate | (a) add new card | (e) edit | (ctrl+d) delete | (esc) back to menu")
	return baseStyle.Render(m.table.View()) + "\n" + help
}
