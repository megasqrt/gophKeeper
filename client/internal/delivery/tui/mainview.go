package tui

import (
	"context"
	"fmt"
	"gophKeeper/client/internal/config"
	"io"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
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
	lastSync     time.Time
	serverOnline bool
	cfg          *config.Config
	storage      LocalStorage
	log          *zerolog.Logger
	width        int
	height       int
}

type serverStatusMsg struct{ online bool }

func NewMainViewModel(login string, lastSync time.Time, cfg *config.Config, storage LocalStorage, log *zerolog.Logger) *MainViewModel {
	items := []list.Item{
		item("Credit Cards"),
		item("Passwords"),
		item("Secure Notes"),
		item("Binary Data"),
		item("Settings"),
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
		login:        login,
		lastSync:     lastSync,
		serverOnline: true, // Изначально считаем, что онлайн
		cfg:          cfg,
		storage:      storage,
		cardModel:    NewCardListModel(storage),
		log:          log,
	}
}

func (m *MainViewModel) Init() tea.Cmd {
	return tea.Batch(m.cardModel.Init(), checkServer(m.cfg), tea.Every(10*time.Minute, func(t time.Time) tea.Msg {
		return checkServer(m.cfg)()
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
		return m, nil // Никакой новой команды не нужно

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
					// При переходе загружаем данные
					if cardVM, ok := m.cardModel.(*CardListModel); ok {
						cardVM.Load()
					}
					return m, nil
					// TODO: Добавить обработку других пунктов меню
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
		return docStyle.Render(m.menu.View())
	}
}