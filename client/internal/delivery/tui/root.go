package tui

import (
	"fmt"
	"gophKeeper/client/internal/config"
	"time"

	tea "github.com/charmbracelet/bubbletea"
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
	storage LocalStorage
	cfg     *config.Config
	width   int
	height  int
}

// NewRootModel создает корневую модель.
func NewRootModel(storage LocalStorage, cfg *config.Config) RootModel {
	// Проверяем, есть ли токен. Если да, считаем пользователя авторизованным.
	// Теперь мы всегда начинаем с экрана входа, чтобы получить пароль для ключа.
	login, _, _ := storage.GetUserCredentials() // Можем получить логин, чтобы предзаполнить поле

	// lastSync и deviceName будут получены после успешного входа.
	// Поэтому передаем пустые значения в NewMainViewModel.
	mainViewModel := NewMainViewModel("", "", time.Time{}, cfg, storage)

	//TODO нормальный логер и валидация токена на просрочку
	fmt.Printf("Login: %s, Token valid:\n", login)
	return RootModel{
		state:   unauthorizedState, // Всегда начинаем с этого состояния
		storage: storage,
		cfg:     cfg,
		login:   NewLoginModel(storage, login, cfg),
		main:    mainViewModel, // Создаем модель главного вида
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

func (m RootModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case tea.KeyMsg:
		if msg.Type == tea.KeyCtrlC {
			return m, tea.Quit
		}
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

		// Проверяем, не был ли вход успешным
		if _, ok := msg.(loginOk); ok {
			m.state = authorizedState
			// После успешного входа нам нужно обновить main view актуальными данными
			login, _, _ := m.storage.GetUserCredentials()
			lastSync, _ := m.storage.GetLastSyncTime()
			m.main = NewMainViewModel(login, "Initial Device", lastSync, m.cfg, m.storage)
			return m, m.main.Init() // Инициализируем main view (запускаем пингер)
		}
	}

	return m, cmd
}

func (m RootModel) View() string {
	if m.state == authorizedState {
		return m.main.View()
	}
	return m.login.View()
}
