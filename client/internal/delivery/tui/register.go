package tui

import (
	"context"
	"fmt"
	"gophKeeper/client/internal/config"
	"gophKeeper/client/internal/transport"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type (
	registrationOk struct {
		token string
	}
)

type regmodel struct {
	cfg           *config.Config
	storage       LocalStorage
	loginInput    textinput.Model
	passwordInput textinput.Model
	focusIndex    int
	err           error
	spinner       spinner.Model
	loading       bool
	registered    bool
	token         string
	attemptsLeft  int
	width         int
}

func InitialModel(storage LocalStorage, cfg *config.Config) regmodel {
	m := regmodel{
		cfg:          cfg,
		storage:      storage,
		attemptsLeft: 3, // Устанавливаем 3 попытки
	}

	m.loginInput = textinput.New()
	m.loginInput.Placeholder = "Login"
	m.loginInput.Focus()
	m.loginInput.CharLimit = 32
	m.loginInput.Width = 20

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

func (m regmodel) Init() tea.Cmd {
	return textinput.Blink
}

func (m regmodel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
				return m, tea.Batch(m.spinner.Tick, performRegistration(m.cfg, m.loginInput.Value(), m.passwordInput.Value(), m.storage))
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
		if m.err == nil { // Игнорируем nil-ошибки от saveCredentials
			return m, nil
		}

		m.loading = false

		// Проверяем, является ли ошибка статусом gRPC
		st, ok := status.FromError(m.err)
		if ok && st.Code() == codes.AlreadyExists {
			// Если пользователь уже существует, нет смысла продолжать. Выходим.
			m.err = fmt.Errorf("user with this login already exists, please choose another one")
			return m, tea.Quit
		}

		m.attemptsLeft--
		if m.attemptsLeft <= 0 {
			m.err = fmt.Errorf("too many failed attempts: %w", m.err)
			return m, tea.Quit
		}
		return m, textinput.Blink

	case registrationOk:
		m.loading = false
		m.registered = true
		m.token = msg.token
		// После успешной регистрации сохраняем учетные данные и выходим.
		// Сохранение токена теперь происходит внутри performRegistration
		return m, tea.Quit

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

func (m *regmodel) updateInputs(msg tea.Msg) tea.Cmd {
	cmds := make([]tea.Cmd, 2)
	m.loginInput, cmds[0] = m.loginInput.Update(msg)
	m.passwordInput, cmds[1] = m.passwordInput.Update(msg)
	return tea.Batch(cmds...)
}

func (m regmodel) View() string {
	if m.loading {
		return fmt.Sprintf("%s Registering...", m.spinner.View())
	}

	var b strings.Builder

	b.WriteString("Register a new user on GophKeeper\n\n")

	b.WriteString(m.loginInput.View())
	b.WriteString("\n")
	b.WriteString(m.passwordInput.View())
	b.WriteString("\n\n")

	button := "[ Register ]"
	if m.focusIndex == 2 {
		button = "> [ Register ]"
	}
	b.WriteString(button)

	if m.err != nil {
		// Создаем стиль для текста ошибки с переносом строк
		// Отступы слева и справа, чтобы текст не прилипал к краям
		errorStyle := lipgloss.NewStyle().
			Width(m.width-4).
			Padding(0, 2).
			Foreground(lipgloss.Color("9")) // Красный цвет для ошибки

		// Добавляем значок к тексту ошибки
		errorText := fmt.Sprintf("❌ Error: %s", m.err.Error())
		if m.attemptsLeft > 0 && m.attemptsLeft < 3 {
			errorText += fmt.Sprintf(" (%d attempts left)", m.attemptsLeft)
		}
		b.WriteString("\n\n" + errorStyle.Render(errorText))
	}

	b.WriteString("\n\n(press esc to quit)")

	return b.String()
}

func performRegistration(cfg *config.Config, login, password string, storage LocalStorage) tea.Cmd {
	return func() tea.Msg {
		if login == "" || password == "" {
			return errMsg(fmt.Errorf("login and password cannot be empty"))
		}

		res, err := transport.Register(context.Background(), login, password)
		if err != nil {
			return errMsg(err)
		}

		// Сохраняем токен
		if err := storage.SaveUserCredentials(login, res.GetToken()); err != nil {
			return errMsg(fmt.Errorf("failed to save credentials: %w", err))
		}

		// "Открываем" хранилище с новым паролем, чтобы зашифровать время
		if err := storage.Unlock(login, password); err != nil {
			return errMsg(fmt.Errorf("failed to unlock storage after registration: %w", err))
		}

		// Сохраняем время успешной операции
		if err := storage.SaveLastSyncTime(time.Now()); err != nil {
			// Не критичная ошибка, просто логируем или игнорируем
		}

		return registrationOk{token: res.GetToken()}
	}
}
