package tui

import (
	"context"
	"fmt"
	"gophKeeper/client/internal/config"
	"gophKeeper/client/internal/transport"
	"io"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	docStyle   = lipgloss.NewStyle().Padding(1, 2, 1, 2)
	titleStyle = lipgloss.NewStyle().MarginLeft(2)
	helpStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Margin(1, 0, 0, 2)
)

// viewState определяет, какой вид сейчас активен в главном окне.
type mainViewState int

const (
	mainMenu mainViewState = iota
	cardView
	// Здесь будут другие состояния: passwordView, noteView и т.д.
)

// item реализует интерфейс list.Item
type item string

func (i item) FilterValue() string { return "" }

type itemDelegate struct{}

func (d itemDelegate) Height() int                               { return 1 }
func (d itemDelegate) Spacing() int                              { return 0 }
func (d itemDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }
func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(item)
	if !ok {
		return
	}

	str := fmt.Sprintf("%d. %s", index+1, i)

	fn := lipgloss.NewStyle().PaddingLeft(4).Render
	if index == m.Index() {
		fn = func(s ...string) string {
			return lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("170")).Render("> " + strings.Join(s, " "))
		}
	}

	fmt.Fprint(w, fn(str))
}

type MainViewModel struct {
	state     mainViewState
	menu      list.Model
	cardModel tea.Model
	// Другие модели для паролей, заметок и т.д.

	login        string
	serverOnline bool
	cfg          *config.Config
	storage      LocalStorage
	width        int
	height       int
}

type serverStatusMsg struct{ online bool }
type checkNowMsg struct{}

func NewMainViewModel( cfg *config.Config, storage LocalStorage) *MainViewModel {
	items := []list.Item{
		item("Credit Cards"),
		item("Passwords"),
		item("Secure Notes"),
		item("Binary Data"),
		item("Settings"), // This will now trigger the check
	}

	const defaultWidth = 20
	const listHeight = 14

	l := list.New(items, itemDelegate{}, defaultWidth, listHeight)
	l.Title = "GophKeeper Main Menu"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.Styles.Title = titleStyle
	l.Styles.HelpStyle = helpStyle

	return &MainViewModel{
		state:        mainMenu,
		menu:         l,
		serverOnline: transport.IsOnline(), // Initialize with status from transport layer
		cfg:          cfg,
		storage:      storage,
		cardModel:    NewCardListModel(storage),
	}
}

func (m *MainViewModel) Init() tea.Cmd {
	// Periodically check the server status every 5 minutes.
	return tea.Batch(m.cardModel.Init(), checkServer(), tea.Every(5*time.Minute, func(t time.Time) tea.Msg {
		return checkNowMsg{}
	}))
}

func (m *MainViewModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.menu.SetWidth(msg.Width)
		return m, nil

	case serverStatusMsg:
		m.serverOnline = msg.online
		return m, nil

	case checkNowMsg:
		return m, checkServer()

	// Сообщение от дочерней модели о возврате в меню
	case backToMenuMsg:
		m.state = mainMenu
		return m, nil

	case tea.KeyMsg:
		// Если мы не в главном меню, передаем управление дочерней модели
		if m.state != mainMenu {
			break
		}

		// Обработка клавиш для главного меню
		switch keypress := msg.String(); keypress {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "enter":
			i, ok := m.menu.SelectedItem().(item)
			if ok {
				switch i {
				case "Credit Cards":
					m.state = cardView
					if cardVM, ok := m.cardModel.(*CardListModel); ok {
						cardVM.Load()
					}
					return m, nil
				case "Settings":
					// Manually trigger a server check
					return m, checkServer()
				}
			}
		}
	}

	// Передаем сообщения в активную модель
	switch m.state {
	case cardView:
		m.cardModel, cmd = m.cardModel.Update(msg)
	default: // mainMenu
		m.menu, cmd = m.menu.Update(msg)
	}

	return m, cmd
}

func (m *MainViewModel) View() string {
	switch m.state {
	case cardView:
		return m.cardModel.View()
	// Другие case для других окон
	default: // mainMenu
		var status string
		if m.serverOnline {
			status = lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Render("Status: Online")
		} else {
			status = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Render("Status: Offline")
		}
		return docStyle.Render(m.menu.View() + "\n" + status)
	}
}

// checkServer returns a command that pings the server and returns a serverStatusMsg.
func checkServer() tea.Cmd {
	return func() tea.Msg {
		err := transport.Ping(context.Background())
		return serverStatusMsg{online: err == nil}
	}
}