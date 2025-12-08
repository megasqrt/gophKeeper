package tui

import (
	"fmt"
	"gophKeeper/client/internal/config"
	"gophKeeper/client/internal/domain"

	//"gophKeeper/client/internal/transport"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type loginOk struct{}

type LoginModel struct {
	cfg           *config.Config
	storage       domain.LocalStorage
	passwordInput textinput.Model
	focusIndex    int
	err           error
	spinner       spinner.Model
	loading       bool
	width         int
	firstRun      bool
}

func NewLoginModel(storage domain.LocalStorage, cfg *config.Config) tea.Model {
	m := LoginModel{
		cfg:     cfg,
		storage: storage,
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

					return m, tea.Batch(m.spinner.Tick, performRegister(m.cfg, m.storage, m.passwordInput.Value()))
				} else {
					return m, tea.Batch(m.spinner.Tick, performLogin(m.cfg, m.storage, m.passwordInput.Value()))
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
		// Ошибка уже установлена, пользователь увидит её в View()
		return m, nil

	case loginOk:
		// Это сообщение означает успешный вход.
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

func performLogin(cfg *config.Config, storage domain.LocalStorage, password string) tea.Cmd {
	return func() tea.Msg {
		if password == "" {
			return errMsg(fmt.Errorf("password cannot be empty"))
		}

		// Шаг 1: Локальная аутентификация. Пытаемся разблокировать хранилище.
		if err := storage.Unlock(cfg.User, password); err != nil {
			return errMsg(fmt.Errorf("failed to unlock local storage: %w", err))
		}

		// Локальная аутентификация прошла успешно, возвращаем loginOk.
		return loginOk{}
	}
}

func performRegister(cfg *config.Config, storage domain.LocalStorage, password string) tea.Cmd {
	return func() tea.Msg {
		if password == "" {
			return errMsg(fmt.Errorf("login and password cannot be empty"))
		}

		if err := storage.LocalRegister(cfg.User, password); err != nil {
			return errMsg(fmt.Errorf("login and password cannot be save local storage: %w", err))
		}

		return loginOk{}
	}
}
