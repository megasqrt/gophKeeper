package tui

import (
	"context"
	"fmt"
	"gophKeeper/client/internal/config"
	"gophKeeper/client/internal/domain"
	"gophKeeper/client/internal/services"
	"gophKeeper/client/internal/transport"

	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/rs/zerolog"
)

type loginOk struct{}

type LoginModel struct {
	cfg           *config.Config
	storage       domain.LocalStorage
	log           *zerolog.Logger
	passwordInput textinput.Model
	focusIndex    int
	err           error
	spinner       spinner.Model
	loading       bool
	width         int
	firstRun      bool
}

func NewLoginModel(storage domain.LocalStorage, cfg *config.Config, log *zerolog.Logger) tea.Model {
	m := LoginModel{
		cfg:     cfg,
		storage: storage,
		log:     log,
	}

	m.firstRun = storage.IsFirstRun()
	m.passwordInput = textinput.New()
	m.passwordInput.Placeholder = "Password"
	m.passwordInput.EchoMode = textinput.EchoPassword
	m.passwordInput.EchoCharacter = '•'
	m.passwordInput.CharLimit = 32
	m.passwordInput.Width = 20
	m.passwordInput.Focus()

	m.spinner = spinner.New()
	m.spinner.Spinner = spinner.Dot
	m.spinner.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return m
}

func (m LoginModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m LoginModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit

		case "tab", "shift+tab", "enter", "up", "down":
			s := msg.String()

			if s == "enter" {
				m.loading = true

				if m.firstRun {

					return m, tea.Batch(m.spinner.Tick, performRegister(m.cfg, m.storage, m.passwordInput.Value(), m.log))
				} else {
					return m, tea.Batch(m.spinner.Tick, performLogin(m.cfg, m.storage, m.passwordInput.Value(), m.log))
				}
			}

			if s == "up" || s == "shift+tab" {
				m.focusIndex--
			} else {
				m.focusIndex++
			}

			if m.focusIndex > 1 {
				m.focusIndex = 0
			} else if m.focusIndex < 0 {
				m.focusIndex = 1
			}

			cmds := make([]tea.Cmd, 1)
			if m.focusIndex == 0 {
				cmds[0] = m.passwordInput.Focus()
			} else {
				m.passwordInput.Blur()
			}
			return m, tea.Batch(cmds...)
		}

	case errMsg:
		m.err = msg
		m.loading = false
		return m, nil

	case loginOk:
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		if m.loading {
			m.spinner, cmd = m.spinner.Update(msg)
		}
		return m, cmd
	}

	cmd := m.updateInputs(msg)
	return m, cmd
}

func (m *LoginModel) updateInputs(msg tea.Msg) tea.Cmd {
	cmds := make([]tea.Cmd, 1)
	m.passwordInput, cmds[0] = m.passwordInput.Update(msg)
	return tea.Batch(cmds...)
}

func (m LoginModel) View() string {
	if m.loading {
		return fmt.Sprintf("%s Logging in...", m.spinner.View())
	}

	var b strings.Builder
	if m.firstRun {
		b.WriteString("Придумайте пароль для локального хранилища.\n\n")
	} else {
		b.WriteString("Добро пожаловать в GophKeeper. Введите пароль для входа.\n\n")
	}
	b.WriteString(m.passwordInput.View())
	b.WriteString("\n\n")

	button := "[ Login ]"
	if m.focusIndex == 1 {
		button = "> [ Login ]"
	}
	b.WriteString(button)

	if m.err != nil {
		errorStyle := lipgloss.NewStyle().
			Width(m.width-4).
			Padding(0, 2).
			Foreground(lipgloss.Color("9"))

		errorText := fmt.Sprintf("❌ Error: %s", m.err.Error())
		b.WriteString("\n\n" + errorStyle.Render(errorText))
	}

	b.WriteString("\n\n(press esc to quit)")

	return b.String()
}

func performLogin(cfg *config.Config, storage domain.LocalStorage, password string, log *zerolog.Logger) tea.Cmd {
	return func() tea.Msg {
		if password == "" {
			return errMsg(fmt.Errorf("password cannot be empty"))
		}

		// Шаг 1: Локальная аутентификация. Пытаемся разблокировать хранилище.
		if err := storage.Unlock(cfg.User, password); err != nil {
			log.Error().Err(err).Msg("Failed to unlock local storage")
			return errMsg(fmt.Errorf("failed to unlock local storage: %w", err))
		}
		log.Info().Msg("Local storage unlocked successfully")

		login, token, deviceID, encryptedMasterKey, err := storage.GetUserCredentials()
		if err != nil {
			errStr := err.Error()
			// Если ошибка связана с отсутствием данных или блокировкой хранилища, продолжаем с локальной сессией
			if errStr == "user not registered or logged in" || errStr == "storage is locked, call Unlock first" || errStr == "token is not set in database" || errStr == "device ID is not set in database" {
				log.Info().Err(err).Msg("No remote credentials found, continuing with local-only session")
				return loginOk{}
			}
			log.Error().Err(err).Msg("Failed to get user credentials - decryption error")
			return errMsg(fmt.Errorf("failed to decrypt stored credentials: %w", err))
		}
		if token == "" || deviceID == "" {
			log.Info().Msg("No remote credentials found (empty token or deviceID), continuing with local-only session")
			return loginOk{}
		}
		log.Info().Msg("Remote credentials found, proceeding with remote login/sync logic")

		encryptionService := services.NewEncryptionService()
		if len(encryptedMasterKey) > 0 {
			log.Info().Msg("Found local encrypted master key, attempting to decrypt")
			if err := encryptionService.DecryptMasterKey(encryptedMasterKey, password, []byte(login)); err != nil {
				log.Error().Err(err).Msg("Failed to decrypt master key from local storage")
			} else {
				log.Info().Msg("Successfully decrypted master key from local storage")
				services.SetGlobalEncryptionService(encryptionService)
			}
		}

		// Шаг 4: Если мастер-ключ не был расшифрован, пытаемся получить его с сервера
		if !encryptionService.HasMasterKey() {
			log.Info().Msg("No local master key available, attempting remote login to fetch it")
			ctx := transport.WithAuthCredentials(context.Background(), "", deviceID)
			res, err := transport.Login(ctx, login, password)
			if err != nil {
				log.Error().Err(err).Msg("Remote login transport error")
			} else if res != nil && len(res.GetEncryptedMasterKey()) > 0 {
				log.Info().Msg("Successfully fetched encrypted master key from server")
				if err := encryptionService.DecryptMasterKey(res.GetEncryptedMasterKey(), password, []byte(login)); err == nil {
					log.Info().Msg("Successfully decrypted master key from server response")
					services.SetGlobalEncryptionService(encryptionService)
					newDeviceID := res.GetDeviceId()
					if newDeviceID == "" {
						newDeviceID = deviceID // Fallback to old deviceID if server didn't return a new one
					}
					log.Info().Str("token_len", fmt.Sprintf("%d", len(res.GetToken()))).Str("deviceID", newDeviceID).Msg("Saving new credentials to local storage")
					if err := storage.SaveUserCredentials(login, res.GetToken(), newDeviceID, res.GetEncryptedMasterKey()); err != nil {
						log.Error().Err(err).Msg("Failed to save credentials to local storage")
						return errMsg(fmt.Errorf("failed to save credentials: %w", err))
					}
				} else {
					log.Error().Err(err).Msg("Failed to decrypt master key from server response")
				}
			} else {
				log.Warn().Msg("Remote login successful but no encrypted master key was returned from server")
			}
		}

		return loginOk{}
	}
}

func performRegister(cfg *config.Config, storage domain.LocalStorage, password string, log *zerolog.Logger) tea.Cmd {
	return func() tea.Msg {
		if password == "" {
			return errMsg(fmt.Errorf("login and password cannot be empty"))
		}

		if err := storage.LocalRegister(cfg.User, password); err != nil {
			log.Error().Err(err).Msg("Failed to register user locally")
			return errMsg(fmt.Errorf("login and password cannot be save local storage: %w", err))
		}
		log.Info().Str("user", cfg.User).Msg("Local registration successful")
		return loginOk{}
	}
}
