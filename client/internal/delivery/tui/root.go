package tui

import (
	"context"
	"fmt"
	"gophKeeper/client/internal/config"
	"gophKeeper/client/internal/domain"
	"gophKeeper/client/internal/services"
	"gophKeeper/client/internal/transport"

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
	program *tea.Program
	cfg     *config.Config
	width   int
	height  int
	log     *zerolog.Logger
	syncer  *services.SyncService
}

// NewRootModel создает корневую модель.
func NewRootModel(storage domain.LocalStorage, cfg *config.Config, syncer *services.SyncService, log *zerolog.Logger) RootModel {
	mainViewModel := NewMainViewModel(cfg, storage, log)

	return RootModel{
		state:   unauthorizedState,
		storage: storage,
		cfg:     cfg,
		login:   NewLoginModel(storage, cfg, log),
		main:    mainViewModel,
		log:     log,
		syncer:  syncer,
	}
}

func (m RootModel) Init() tea.Cmd {
	switch m.state {
	case authorizedState:
		return m.main.Init()
	default:
		return m.login.Init()
	}
}

func (m *RootModel) SetProgram(p *tea.Program) {
	m.program = p
	if mainVM, ok := m.main.(*MainViewModel); ok {
		mainVM.SetProgram(p)
	}
}

type initialSyncDone struct{}

func (m *RootModel) performInitialSync() tea.Cmd {
	return func() tea.Msg {
		m.log.Info().Msg("Performing initial synchronization after login.")

		_, token, deviceID, _, err := m.storage.GetUserCredentials()
		if err != nil {
			m.log.Error().Err(err).Msg("Failed to get user credentials for sync")
			return initialSyncDone{}
		}

		if token == "" {
			m.log.Error().Msg("Token is empty, cannot perform sync")
			return initialSyncDone{}
		}
		if deviceID == "" {
			m.log.Error().Msg("DeviceID is empty, cannot perform sync")
			return initialSyncDone{}
		}

		m.log.Debug().Str("token_len", fmt.Sprintf("%d", len(token))).Str("deviceID", deviceID).Msg("Credentials retrieved for sync")

		ctx := transport.WithAuthCredentials(context.Background(), token, deviceID)

		if err := m.syncer.Sync(ctx); err != nil {
			m.log.Error().Err(err).Msg("Initial synchronization failed after login.")
		} else {
			m.log.Info().Msg("Initial synchronization completed successfully.")
		}
		return initialSyncDone{}
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
	case loginOk:
		m.state = authorizedState
		return m, tea.Batch(m.main.Init(), m.performInitialSync())

	case initialSyncDone:
		return m, nil
	}

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
