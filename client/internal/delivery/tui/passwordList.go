package tui

import (
	"fmt"
	"gophKeeper/client/internal/domain"
	model "gophKeeper/pkg/grpchelper"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/rs/zerolog"
)

// pass list has its own viewstate because it has different views from cards
// type passViewState int

// const (
// 	passTableView passViewState = iota
// 	passFormView
// 	passConfirmDeleteView
// )

type deletePassMsg struct{ confirmed bool }

type PassListModel struct {
	state        viewState
	table        table.Model
	form         PassFormModel
	confirmModel ConfirmModel
	storage      domain.LocalStorage
	passs        []model.Password // Добавляем поле для хранения полных данных
	width        int
	height       int
	log          *zerolog.Logger
}

func NewPassListModel(storage domain.LocalStorage, log *zerolog.Logger) *PassListModel {
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
		log:     log,
	}
	m.confirmModel = NewConfirmModel("Default prompt", func(confirmed bool) tea.Cmd {
		return func() tea.Msg {
			return deletePassMsg{confirmed: confirmed}
		}
	})

	return m
}

// Load загружает пароли из хранилища и обновляет строки таблицы.
func (m *PassListModel) Load() {
	rows, passs := m.loadPasss()
	m.passs = passs
	m.table.SetRows(rows)
}

func (m *PassListModel) loadPasss() ([]table.Row, []model.Password) {
	allPasswords, err := m.storage.GetPasss()
	if err != nil {
		//	m.log.Error().Err(err).Msg("get passs error")
		return []table.Row{}, []model.Password{}
	}

	var rows []table.Row
	var displayedPasswords []model.Password
	for _, pass := range allPasswords {
		if !pass.Deleted {
			maskedPassword := strings.Repeat("*", len(pass.Password))
			rows = append(rows, table.Row{pass.Login, maskedPassword, pass.Description})
			displayedPasswords = append(displayedPasswords, pass)
		}
	}
	return rows, displayedPasswords
}

func (m *PassListModel) Init() tea.Cmd {
	return nil
}

func (m *PassListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	// First, handle messages that change state or come from commands
	case deletePassMsg:
		m.state = tableView
		if msg.confirmed {
			if len(m.passs) > 0 {
				selectedPass := m.passs[m.table.Cursor()]
				err := m.storage.DeletePass(selectedPass.GetLocalID())
				if err != nil {
					m.log.Error().Err(err).Msg("error deleting password")
				}
				m.Load() // Reload to reflect deletion
			}
		}
		return m, nil

	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.confirmModel.setSize(msg.Width, msg.Height)

	case passFormBackMsg:
		m.state = tableView
		return m, nil
	case passFormSavedMsg:
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
		m.form = newForm.(PassFormModel)
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
			m.form = NewPassForm(m.storage, nil) // Create a new clean form
			return m, m.form.Init()
		case "e", "enter":
			if len(m.passs) == 0 {
				return m, nil
			}
			selectedPass := m.passs[m.table.Cursor()]
			m.state = formView
			m.form = NewPassForm(m.storage, &selectedPass) // Pass the selected password to the form
			return m, m.form.Init()

		case "ctrl+d":
			if m.state == tableView && len(m.passs) > 0 {
				selectedPass := m.passs[m.table.Cursor()]
				m.confirmModel.SetPrompt(fmt.Sprintf("пароль для '%s'", selectedPass))
				m.state = confirmDeleteView
				return m, nil
			}
		}
	}

	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m *PassListModel) View() string {
	if m.state == confirmDeleteView {
		return m.confirmModel.View()
	}
	if m.state == formView {
		return m.form.View()
	}

	help := helpStyle.Render("(↑/↓) navigate | (a) add | (e) edit | (ctrl+d) delete | (esc) back")
	return baseStyle.Render(m.table.View()) + "\n" + help
}
