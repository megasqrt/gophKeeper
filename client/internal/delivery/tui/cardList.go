package tui

import (
	"fmt"
	"gophKeeper/client/internal/domain/model"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/rs/zerolog/log"
)

type viewState int

const (
	listView viewState = iota
	formView
)

type backToMenuMsg struct{}

// CardListModel управляет состоянием вкладки "Card".
type CardListModel struct {
	state   viewState
	list    list.Model
	form    CardFormModel
	storage LocalStorage
}

func NewCardListModel(storage LocalStorage) *CardListModel {
	m := &CardListModel{
		state:   listView,
		storage: storage,
		form:    NewCardForm(storage),
	}

	// Настраиваем список
	cardList := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	cardList.Title = "Your Saved Cards"
	cardList.SetShowStatusBar(false)
	cardList.SetFilteringEnabled(false) // Пока отключаем фильтрацию

	m.list = cardList
	return m
}

// Load загружает карты из хранилища и обновляет список.
func (m *CardListModel) Load() {
	items := m.loadCards()
	m.list.SetItems(items)
}

func (m *CardListModel) loadCards() []list.Item {
	cardsData, err := m.storage.GetCards()
	if err != nil {
		log.Error().Err(err).Msg("get cards error")
		return []list.Item{}
	}

	items := make([]list.Item, len(cardsData))
	for i, data := range cardsData {
		items[i] = model.Card{
			ID:     data["number"], // Используем номер как ID
			Number: data["number"],
			Holder: data["holder"],
			Expiry: data["expiry"],
			CVV:    data["cvv"],
		}
	}
	return items
}

func (m CardListModel) Init() tea.Cmd {
	return nil
}

func (m *CardListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg.(type) {
	// Сообщения от дочерней модели (формы)
	case cardFormBackMsg:
		m.state = listView
		return m, nil
	case cardFormSavedMsg:
		// После сохранения обновляем список и возвращаемся к нему
		m.list.SetItems(m.loadCards())
		m.state = listView
		return m, nil
	}

	// Передаем сообщения в активную дочернюю модель
	if m.state == formView {
		var newForm tea.Model
		newForm, cmd = m.form.Update(msg)
		m.form = newForm.(CardFormModel) // CardFormModel не указатель, здесь все ок
		return m, cmd
	}

	// Обработка сообщений для списка
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		h, v := docStyle.GetFrameSize()
		m.list.SetSize(msg.Width-h, msg.Height-v)

	case tea.KeyMsg:
		// Не обрабатываем клавиши, если список пуст, кроме 'a' и 'q'
		if m.list.FilterState() == list.Filtering || (len(m.list.Items()) == 0 && msg.String() != "a" && msg.String() != "q") {
			// Исключение для выхода
			if msg.String() == "q" {
				return m, func() tea.Msg { return backToMenuMsg{} }
			}
		}

		switch msg.String() {
		case "a": // 'a' for "add"
			m.state = formView
			return m, m.form.Init()
		case "q":
			return m, func() tea.Msg { return backToMenuMsg{} }
		}
	}

	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m *CardListModel) View() string {
	if m.state == formView {
		return m.form.View()
	}

	help := "(a) add new card | (q) back to menu"
	if len(m.list.Items()) > 0 {
		help = "(↑/↓) navigate | " + help
	}
	return fmt.Sprintf("%s\n%s", m.list.View(), helpStyle.Render(help))
}
