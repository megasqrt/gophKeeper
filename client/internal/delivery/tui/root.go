package tui

import (
	"gophKeeper/client/internal/config"

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

	// Всегда начинаем с экрана входа, чтобы получить пароль для ключа.
	//login, _, _ := storage.GetUserCredentials() // Можем получить логин, чтобы предзаполнить поле

	// lastSync и deviceName будут получены после успешного входа.
	// Поэтому передаем пустые значения в NewMainViewModel.
	mainViewModel := NewMainViewModel(cfg, storage)

	return RootModel{
		state:   unauthorizedState, // Всегда начинаем с этого состояния
		storage: storage,
		cfg:     cfg,
		login:   NewLoginModel(storage, cfg),
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
		switch msg.Type {
		case tea.KeyCtrlC:
			return m, tea.Quit
		}
	// Проверяем, не был ли вход успешным. Эта проверка должна быть здесь,
	// чтобы перехватить сообщение от дочерней модели login.
	case loginOk:
		m.state = authorizedState
		// После успешного входа нам нужно обновить main view актуальными данными
		//login, _, _ := m.storage.GetUserCredentials()
		//lastSync, _ := m.storage.GetLastSyncTime()
		newMainModel := NewMainViewModel(m.cfg, m.storage)

		// Теперь, когда хранилище открыто, загружаем данные для вкладок.
		if cardVM, ok := newMainModel.cardModel.(*CardListModel); ok {
			cardVM.Load()
		}
		m.main = newMainModel
		//return m, tea.Batch(m.main.Init(), func() tea.Msg { return loginOk{} }) // Инициализируем main view (запускаем пингер)
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
