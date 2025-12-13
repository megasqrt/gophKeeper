package tui

import (
	"gophKeeper/client/internal/config"
	"gophKeeper/client/internal/domain"
	"gophKeeper/client/internal/services"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/rs/zerolog"
)

// sessionState определяет текущее состояние сессии пользователя.
type sessionState int

const (
	unauthorizedState sessionState = iota // Пользователь не авторизован, показываем экран входа.
	authorizedState                       // Пользователь авторизован, показываем главный экран.
)

// RootModel - это корневая модель, которая управляет другими моделями (login, mainview).
type RootModel struct {
	state   sessionState
	login   tea.Model
	main    tea.Model
	storage domain.LocalStorage
	program *tea.Program // Ссылка на программу для отправки сообщений
	cfg     *config.Config
	width   int
	height  int
	log     *zerolog.Logger
}

// NewRootModel создает корневую модель.
func NewRootModel(storage domain.LocalStorage, cfg *config.Config, syncer *services.SyncService, log *zerolog.Logger) RootModel {
	// Всегда начинаем с экрана входа, чтобы получить пароль для ключа.
	mainViewModel := NewMainViewModel(cfg, storage, syncer, log)

	return RootModel{
		state:   unauthorizedState, // Всегда начинаем с этого состояния
		storage: storage,
		cfg:     cfg,
		login:   NewLoginModel(storage, cfg),
		main:    mainViewModel, // Создаем модель главного вида
		log:     log,
	}
}

func (m RootModel) Init() tea.Cmd {
	switch m.state {
	case authorizedState:
		return m.main.Init()
	default: // unauthorizedState
		return m.login.Init()
	}
}

func (m *RootModel) SetProgram(p *tea.Program) {
	m.program = p
	// Также передаем программу в дочерние модели, которым она нужна
	if mainVM, ok := m.main.(*MainViewModel); ok {
		mainVM.SetProgram(p)
	}
}

func (m RootModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			return m, tea.Quit
		}
	// Проверяем, не был ли вход успешным. Эта проверка должна быть здесь,
	// чтобы перехватить сообщение от дочерней модели login.
	case loginOk:
		m.state = authorizedState

		// Не создаем новую модель, а передаем программу в уже существующую.
		// Указатель на программу был установлен в app.go.
		// Теперь мы просто "активируем" main модель.
		return m, m.main.Init()
	}

	// Передаем сообщения в активную дочернюю модель
	var cmd tea.Cmd
	if m.state == authorizedState {
		var newMainModel tea.Model
		newMainModel, cmd = m.main.Update(msg)
		m.main = newMainModel
	} else {
		var newLoginModel tea.Model
		newLoginModel, cmd = m.login.Update(msg)
		m.login = newLoginModel
	}
	return m, cmd
}

func (m RootModel) View() string {
	if m.state == authorizedState {
		return m.main.View()
	}
	return m.login.View()
}
