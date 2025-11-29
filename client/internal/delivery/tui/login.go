package tui

import (
	"errors"
	"fmt"
	"gophKeeper/client/internal/config"
	"gophKeeper/client/internal/transport"
	
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	)

type loginOk struct{}

type LoginModel struct {
	cfg           *config.Config
	storage       LocalStorage
	loginInput    textinput.Model
	passwordInput textinput.Model
	focusIndex    int
	err           error
	spinner       spinner.Model
	loading       bool
	width         int
}

func NewLoginModel(storage LocalStorage, login string, cfg *config.Config) tea.Model {
	m := LoginModel{
		cfg:     cfg,
		storage: storage,
	}

	m.loginInput = textinput.New()
	m.loginInput.Placeholder = "Login"
	m.loginInput.Focus()
	m.loginInput.CharLimit = 32
	m.loginInput.Width = 20
	if login != "" {
		m.loginInput.SetValue(login)

	}

	m.passwordInput = textinput.New()
	m.passwordInput.Placeholder = "Password"
	m.passwordInput.EchoMode = textinput.EchoPassword
	m.passwordInput.EchoCharacter = '•'
	m.passwordInput.CharLimit = 32
	m.passwordInput.Width = 20

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

			if s == "enter" && m.focusIndex == 2 {
				m.loading = true
				return m, tea.Batch(m.spinner.Tick, performLogin(m.cfg, m.storage, m.loginInput.Value(), m.passwordInput.Value()))
			}

			if s == "up" || s == "shift+tab" {
				m.focusIndex--
			} else {
				m.focusIndex++
			}

			if m.focusIndex > 2 {
				m.focusIndex = 0
			} else if m.focusIndex < 0 {
				m.focusIndex = 2
			}

			cmds := make([]tea.Cmd, 2)
			if m.focusIndex == 0 {
				cmds[0] = m.loginInput.Focus()
				m.passwordInput.Blur()
			} else if m.focusIndex == 1 {
				m.loginInput.Blur()
				cmds[0] = m.passwordInput.Focus()
			} else {
				m.loginInput.Blur()
				m.passwordInput.Blur()
			}
			return m, tea.Batch(cmds...)
		}

	case errMsg:
		m.err = msg
		m.loading = false
		//TODO parse error
		// if ok && st.Code() == codes.Unauthenticated {
		// 	m.err = errors.New("invalid login or password")
		// } else if ok && st.Code() == codes.Unavailable {
		// 	m.err = errors.New("server is unavailable, please try again later")
		// }

		return m, nil

	case loginOk:
		// Это сообщение означает успешный вход. Мы должны выйти из программы,
		// Корневая модель перехватит это сообщение и переключит вид.
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
	cmds := make([]tea.Cmd, 2)
	m.loginInput, cmds[0] = m.loginInput.Update(msg)
	m.passwordInput, cmds[1] = m.passwordInput.Update(msg)
	return tea.Batch(cmds...)
}

func (m LoginModel) View() string {
	if m.loading {
		return fmt.Sprintf("%s Logging in...", m.spinner.View())
	}

	var b strings.Builder
	// ... остальной код View() похож на register.go, я его восстановлю для полноты
	b.WriteString("Welcome to GophKeeper. Please log in.\n\n")
	b.WriteString(m.loginInput.View())
	b.WriteString("\n")
	b.WriteString(m.passwordInput.View())
	b.WriteString("\n\n")

	button := "[ Login ]"
	if m.focusIndex == 2 {
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

func performLogin(cfg *config.Config, storage LocalStorage, login, password string) tea.Cmd {
	return func() tea.Msg {
		if login == "" || password == "" {
			return errMsg(fmt.Errorf("login and password cannot be empty"))
		}

		// Шаг 1: Локальная аутентификация. Пытаемся разблокировать хранилище.
		if err := storage.Unlock(login, password); err != nil {
			return errMsg(fmt.Errorf("failed to unlock local storage: %w", err))
		}
		// Проверяем, что ключ подходит, пытаясь что-нибудь расшифровать.
		if _, err := storage.GetLastSyncTime(); err != nil {
			// Если здесь ошибка, скорее всего, пароль неверный.
			return errMsg(errors.New("invalid login or password"))
		}

		err := transport.ping()
		if err != nil {
			// Сервер недоступен. Это НЕ ошибка для входа.
			// Просто логируем и продолжаем, приложение будет работать с локальными данными.
			fmt.Printf("Warning: server is unavailable, proceeding in offline mode: %v\n", err)
		} else {
			
			if res, err := transport.Login(login, password); err == nil {
				// Если сервер доступен и логин успешен, обновляем токен и время синхронизации.
				_ = storage.SaveUserCredentials(login, res.GetToken())
				_ = storage.SaveLastSyncTime(time.Now())
			}
		}
	
		// Локальная аутентификация прошла успешно, возвращаем loginOk.
		return loginOk{}
	}
}
