package tui

import (
	"gophKeeper/client/internal/domain/model"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)


type PassListModel struct {
	state   viewState
	table   table.Model
	form    PassFormModel
	storage LocalStorage
	passs   []model.Password // Добавляем поле для хранения полных данных карт
}

func NewPassListModel(storage LocalStorage) *PassListModel {
	columns := []table.Column{
		{Title: "Login", Width: 20},
		{Title: "Password", Width: 25},
		{Title: "Description", Width: 10},
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

	m := &PassListModel{
		state:   tableView,
		storage: storage,
		form:    NewPassForm(storage, nil),
		table:   tbl,
	}

	return m
}

// Load загружает пароли из хранилища и обновляет строки таблицы.
func (m *PassListModel) Load() {
	rows, passs := m.loadPasss()
	m.passs = passs
	m.table.SetRows(rows)
}

func (m *PassListModel) loadPasss() ([]table.Row, []model.Password) {
	passsData, err := m.storage.GetPasss()
	if err != nil {
	//	m.log.Error().Err(err).Msg("get passs error")
		return []table.Row{}, []model.Password{}
	}

	rows := make([]table.Row, len(passsData))
	passs := make([]model.Password, len(passsData))
	for i, data := range passsData {
		pass := model.Password{
			ID:     data["id"],
			Login: data["login"],
			Password: data["password"],
			Description: data["description"],
		}
		// Используем Title() и Description() из модели пароли для консистентности
		passs[i] = pass
		rows[i] = table.Row{pass.Login, pass.Password, pass.Description}
	}
	return rows, passs
}

func (m *PassListModel) Init() tea.Cmd {
	return nil
}

func (m *PassListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg.(type) {
	case passFormBackMsg:
		m.state = tableView
		return m, nil
	case passFormSavedMsg:
		m.Load() // Просто перезагружаем данные в таблицу
		m.state = tableView
		return m, nil
	}

	if m.state == formView {
		var newForm tea.Model
		newForm, cmd = m.form.Update(msg)
		m.form = newForm.(PassFormModel)
		return m, cmd
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc":
			return m, func() tea.Msg { return backToMenuMsg{} }
		case "a": // 'a' for "add"
			m.state = formView
			m.form = NewPassForm(m.storage,nil) // Создаем новую чистую форму
			return m, m.form.Init()
		case "e", "enter":
			if len(m.passs) == 0 {
				return m, nil
			}
			selectedPass := m.passs[m.table.Cursor()]
			m.state = formView
			m.form = NewPassForm(m.storage, &selectedPass) // Передаем выбранный пароль в форму
			return m, m.form.Init()
		}
	}

	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m *PassListModel) View() string {
	if m.state == formView {
		return m.form.View()
	}

	help := helpStyle.Render("(↑/↓) navigate | (a) add new pass | (q) back to menu")
	return baseStyle.Render(m.table.View()) + "\n" + help
}