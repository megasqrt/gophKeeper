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

// viewState определяет, какой вид сейчас активен в главном окне.
type mainViewState int

const (
	mainMenu mainViewState = iota
	cardView
	passView
	textView
	fileView
	settingsView
	// Здесь будут другие состояния: passwordView, noteView и т.д.
)

// item реализует интерфейс list.Item
type item string

func (i item) FilterValue() string { return "" }

type mainMenuDelegate struct{}

func (d mainMenuDelegate) Height() int                               { return 1 }
func (d mainMenuDelegate) Spacing() int                              { return 0 }
func (d mainMenuDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }
func (d mainMenuDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
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
	passModel tea.Model
	textModel tea.Model
	fileModel tea.Model
	settingsModel tea.Model
	// Другие модели для паролей, заметок и т.д.

	login        string
	serverOnline bool
	cfg          *config.Config
	program      *tea.Program
	storage      LocalStorage
	width        int
	height       int
}

type serverStatusMsg struct{ online bool }
type checkNowMsg struct{}

func NewMainViewModel(cfg *config.Config, storage LocalStorage) *MainViewModel {
	items := []list.Item{
		item("Credit Cards"),
		item("Passwords"),
		item("Text Notes"),
		item("Binary Data"),
		item("Settings"), // This will now trigger the check
	}

	l := list.New(items, mainMenuDelegate{}, DefaulListtWidth, DefaultlistHeight)
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
		passModel:    NewPassListModel(storage),
		textModel:    NewTextEditModel(storage),
		fileModel:    NewFileUploadModel(storage),
		settingsModel: NewSettingsModel(),
	}
}

func (m *MainViewModel) SetProgram(p *tea.Program) {
	m.program = p
	// Передаем программу в дочерние модели, которым она нужна
	if fileVM, ok := m.fileModel.(*FileUploadModel); ok {
		fileVM.SetProgram(p)
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
				case "Passwords":
					m.state = passView
					if passVM, ok := m.passModel.(*PassListModel); ok {
						passVM.Load()
					}
					return m, nil
				case "Text Notes":
					m.state = textView
					if textVM, ok := m.textModel.(*TextEditModel); ok {
						textVM.Load()
					}
					return m, nil
				case "Binary Data":
					m.state = fileView
					if fileVM, ok := m.fileModel.(*FileUploadModel); ok {
						return m, fileVM.Load()
					}
					return m, nil
				case "Settings":
					m.state = settingsView
					return m, nil
				}
			}
		}
	}

	// Передаем сообщения в активную модель
	switch m.state {
	case cardView:
		m.cardModel, cmd = m.cardModel.Update(msg)
	case passView:
		m.passModel, cmd = m.passModel.Update(msg)
	case textView:
		m.textModel, cmd = m.textModel.Update(msg)
	case fileView:
		m.fileModel, cmd = m.fileModel.Update(msg)
	case settingsView:
		m.settingsModel, cmd = m.settingsModel.Update(msg)
	default: // mainMenu
		m.menu, cmd = m.menu.Update(msg)
	}

	return m, cmd
}

func (m *MainViewModel) View() string {
	switch m.state {
	case cardView:
		return m.cardModel.View()
	case passView:
		return m.passModel.View()
	case textView:
		return m.textModel.View()
	case fileView:
		return m.fileModel.View()
	case settingsView:
		return m.settingsModel.View()
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
