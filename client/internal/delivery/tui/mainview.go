package tui

import (
	"context"
	"fmt"
	"gophKeeper/client/internal/config"
	"gophKeeper/client/internal/domain"
	"gophKeeper/client/internal/services"
	"gophKeeper/client/internal/transport"
	"io"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/rs/zerolog"
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
	registerView
)

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

	str := string(i)

	fn := lipgloss.NewStyle().PaddingLeft(4).Render

	// Если пункт меню для регистрации и токен невалиден, окрашиваем в красный
	// if strings.Contains(str, "Register") && !d.tokenIsValid {
	// 	fn = lipgloss.NewStyle().PaddingLeft(4).Foreground(lipgloss.Color("#FF5B5B")).Render
	// }

	// Стиль для выбранного элемента (переопределяет предыдущий стиль)
	if index == m.Index() {
		fn = func(s ...string) string { // Active item
			return lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("170")).Render("> " + strings.Join(s, " "))
		}
	}

	fmt.Fprint(w, fn(str))
}

type MainViewModel struct {
	state         mainViewState
	menu          list.Model
	cardModel     tea.Model
	passModel     tea.Model
	textModel     tea.Model
	fileModel     tea.Model
	settingsModel tea.Model
	registerModel tea.Model

	login        string
	serverOnline bool
	tokenIsValid bool
	isSyncing    bool
	syncErr      error // Ошибка синхронизации
	cfg          *config.Config
	program      *tea.Program
	storage      domain.LocalStorage
	width        int
	height       int
	log          *zerolog.Logger
}

type serverStatusMsg struct{ tokenValid bool }
type syncStartMsg struct{}
type syncFinishMsg struct{ err error }
type checkNowMsg struct{}

// NewMainViewModel создает главную модель представления.
// syncer - это сервис для синхронизации данных.
func NewMainViewModel(cfg *config.Config, storage domain.LocalStorage, syncer *services.SyncService, log *zerolog.Logger) *MainViewModel {
	items := []list.Item{
		item("💳 Credit Cards"),
		item("🔑 Passwords"),
		item("📝 Text Notes"),
		item("📦 Binary Data"),
		item("⚙️ Settings"),
		item("🚀 Register to server"),
	}

	l := list.New(items, mainMenuDelegate{}, DefaulListtWidth, DefaultlistHeight)
	l.Title = "GophKeeper Main Menu"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.Styles.Title = titleStyle
	l.Styles.HelpStyle = helpStyle

	return &MainViewModel{
		state:         mainMenu,
		menu:          l,
		serverOnline:  transport.IsOnline(), // Initialize with status from transport layer
		tokenIsValid:  false,                // Assume token is invalid at start, will be checked
		isSyncing:     false,
		cfg:           cfg,
		storage:       storage,
		cardModel:     NewCardListModel(storage, log),
		passModel:     NewPassListModel(storage, log),
		textModel:     NewTextEditModel(storage, log),
		fileModel:     NewFileUploadModel(storage),
		settingsModel: NewSettingsModel(),
		registerModel: InitialModel(storage, cfg, syncer),
		log:           log,
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
	return tea.Batch(m.cardModel.Init(), checkServer(m.storage), tea.Every(5*time.Minute, func(t time.Time) tea.Msg {
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
		m.tokenIsValid = msg.tokenValid
		if m.tokenIsValid {
			// Если токен валиден, запускаем синхронизацию
			return m, func() tea.Msg { return syncStartMsg{} }
		}
		return m, nil

	case syncStartMsg:
		m.isSyncing = true
		return m, performSync(m.registerModel.(*regmodel).syncer)

	case syncFinishMsg:
		m.isSyncing = false
		m.syncErr = msg.err
		// Ошибка будет показана в View()
		return m, nil

	case checkNowMsg:
		return m, checkServer(m.storage)

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
				case "💳 Credit Cards":
					m.state = cardView
					if cardVM, ok := m.cardModel.(*CardListModel); ok {
						cardVM.Load()
					}
					return m, nil
				case "🔑 Passwords":
					m.state = passView
					if passVM, ok := m.passModel.(*PassListModel); ok {
						passVM.Load()
					}
					return m, nil
				case "📝 Text Notes":
					m.state = textView
					if textVM, ok := m.textModel.(*TextEditModel); ok {
						textVM.Load()
					}
					return m, nil
				case "📦 Binary Data":
					m.state = fileView
					if fileVM, ok := m.fileModel.(*FileUploadModel); ok {
						return m, fileVM.Load()
					}
					return m, nil
				case "⚙️ Settings":
					m.state = settingsView
					return m, nil
				case "🚀 Register to server", "☁️ Sync with server":
					m.state = registerView
					if m.tokenIsValid {
						// Если мы уже авторизованы, "Sync" просто запускает синхронизацию
						return m, func() tea.Msg { return syncStartMsg{} }
					}
					return m, m.registerModel.Init()
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
	case registerView:
		m.registerModel, cmd = m.registerModel.Update(msg)
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
	case registerView:
		return m.registerModel.View()
	// Другие case для других окон
	default: // mainMenu
		var status string

		// Динамически меняем текст пункта меню
		items := m.menu.Items()
		if m.tokenIsValid {
			items[5] = item("☁️ Sync with server")
		} else {
			items[5] = item("🚀 Register to server")
		}
		m.menu.SetItems(items)
		if m.tokenIsValid {
			// Используем AdaptiveColor для надежного отображения цвета
			green := lipgloss.AdaptiveColor{Light: "#43BF6D", Dark: "#73F59F"}
			statusText := "Authorized"
			status = lipgloss.NewStyle().Foreground(green).Render(statusText)
		} else {
			// Используем AdaptiveColor для надежного отображения цвета
			red := lipgloss.AdaptiveColor{Light: "#FF5B5B", Dark: "#FF6B6B"}
			statusText := "Offline / Not Registered"
			status = lipgloss.NewStyle().Foreground(red).Render(statusText)
		}

		// Добавляем статус синхронизации
		if m.isSyncing {
			syncStatus := lipgloss.NewStyle().
				Foreground(lipgloss.Color("213")). // Оранжевый
				Render("  Syncing...")
			status += syncStatus
		}

		// Показываем ошибку синхронизации, если есть
		var errorSection string
		if m.syncErr != nil {
			errorStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("9")).
				Padding(0, 1)
			errorSection = "\n" + errorStyle.Render("❌ Sync error: "+m.syncErr.Error())
		}

		return docStyle.Render(m.menu.View() + "\n" + status + errorSection)
	}
}

// checkServer returns a command that pings the server and returns a serverStatusMsg.
func checkServer(storage domain.LocalStorage) tea.Cmd {
	return func() tea.Msg {
		_, token, _, _, err := storage.GetUserCredentials()
		if err != nil {
			return serverStatusMsg{tokenValid: false}
		}
		return serverStatusMsg{tokenValid: transport.Ping(token)}
	}
}

//TODO 
// performSync запускает процесс синхронизации в фоновом режиме.
func performSync(syncer *services.SyncService) tea.Cmd {
	return func() tea.Msg {
		// Запускаем синхронизацию в горутине, чтобы не блокировать UI
		err := syncer.Sync(context.Background())
		// Возвращаем сообщение о завершении
		return syncFinishMsg{err: err}
	}
}
