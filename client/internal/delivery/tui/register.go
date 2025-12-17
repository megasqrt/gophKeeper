package tui

import (
	"context"
	"fmt"
	"gophKeeper/client/internal/config"
	"gophKeeper/client/internal/domain"
	"gophKeeper/client/internal/services"
	"gophKeeper/client/internal/transport"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"

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
	storage       domain.LocalStorage
	loginInput    textinput.Model
	passwordInput textinput.Model
	emailInput    textinput.Model
	focusIndex    int
	err           error
	spinner       spinner.Model
	loading       bool
	registered    bool
	token         string
	attemptsLeft  int
	width         int
}

func InitialModel(storage domain.LocalStorage, cfg *config.Config) *regmodel {
	m := &regmodel{
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

	m.emailInput = textinput.New()
	m.emailInput.Placeholder = "Email"
	m.emailInput.CharLimit = 64
	m.emailInput.Width = 20

	m.spinner = spinner.New()
	m.spinner.Spinner = spinner.Dot
	m.spinner.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return m
}

func (m *regmodel) Init() tea.Cmd {
	return textinput.Blink
}

func (m *regmodel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, func() tea.Msg { return backToMenuMsg{} }

		case "tab", "shift+tab", "enter", "up", "down":
			s := msg.String()

			if s == "enter" && m.focusIndex == 3 {
				m.loading = true
				return m, tea.Batch(m.spinner.Tick, performRegistration(m.cfg, m.loginInput.Value(), m.passwordInput.Value(), m.emailInput.Value(), m.storage))
			}

			if s == "up" || s == "shift+tab" {
				m.focusIndex--
			} else {
				m.focusIndex++
			}

			if m.focusIndex > 3 {
				m.focusIndex = 0
			} else if m.focusIndex < 0 {
				m.focusIndex = 3
			}

			cmds := make([]tea.Cmd, 2)
			if m.focusIndex == 0 {
				cmds[0] = m.loginInput.Focus()
				m.passwordInput.Blur()
			} else if m.focusIndex == 1 {
				m.loginInput.Blur()
				m.emailInput.Blur()
				cmds[0] = m.passwordInput.Focus()
			} else if m.focusIndex == 2 {
				m.loginInput.Blur()
				m.passwordInput.Blur()
				cmds[0] = m.emailInput.Focus()
			} else {
				m.loginInput.Blur()
				m.passwordInput.Blur()
				m.emailInput.Blur()
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
			m.err = fmt.Errorf("user with this login already exists, please choose another one")
			return m, tea.Quit
		}

		m.attemptsLeft--
		if m.attemptsLeft != 0 {
			m.err = fmt.Errorf("too many failed attempts: %w", m.err)
			return m, tea.Quit
		}
		return m, textinput.Blink

	case registrationOk:
		m.loading = false
		m.registered = true
		m.token = msg.token
		return m, func() tea.Msg { return loginOk{} }

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
	m.emailInput, cmds[1] = m.emailInput.Update(msg) // reuse index, batch will handle it
	return tea.Batch(cmds...)
}

func (m *regmodel) View() string {
	if m.loading {
		return fmt.Sprintf("%s Registering...", m.spinner.View())
	}

	var b strings.Builder

	b.WriteString("Register a new user on GophKeeper\n\n")

	b.WriteString(m.loginInput.View())
	b.WriteString("\n")
	b.WriteString(m.passwordInput.View())
	b.WriteString("\n")
	b.WriteString(m.emailInput.View())
	b.WriteString("\n\n")

	button := "[ Register ]"
	if m.focusIndex == 3 {
		button = "> [ Register ]"
	}
	b.WriteString(button)

	if m.err != nil {
		errorStyle := lipgloss.NewStyle().
			Width(m.width-4).
			Padding(0, 2).
			Foreground(lipgloss.Color("9"))
		errorText := fmt.Sprintf("❌ Error: %s", m.err.Error())
		if m.attemptsLeft > 0 && m.attemptsLeft < 3 {
			errorText += fmt.Sprintf(" (%d attempts left)", m.attemptsLeft)
		}
		b.WriteString("\n\n" + errorStyle.Render(errorText))
	}

	b.WriteString("\n\n(press esc to quit)")

	return b.String()
}

func performRegistration(cfg *config.Config, login, password, email string, storage domain.LocalStorage) tea.Cmd {
	return func() tea.Msg {
		if login == "" || password == "" {
			return errMsg(fmt.Errorf("login and password cannot be empty"))
		}

		res, err := transport.Register(context.Background(), login, password, email)
		if err != nil {
			return errMsg(err)
		}

		deviceID, err := getDeviceIDFromToken(res.GetToken())
		if err != nil {
			return errMsg(fmt.Errorf("failed to parse token: %w", err))
		}

		// Сохраняем токен и зашифрованный мастер-ключ
		encryptedMasterKey := res.GetEncryptedMasterKey()
		if err := storage.SaveUserCredentials(login, res.GetToken(), deviceID, encryptedMasterKey); err != nil {
			return errMsg(fmt.Errorf("failed to save credentials: %w", err))
		}

		encryptionService := services.NewEncryptionService()
		if len(encryptedMasterKey) > 0 {
			if err := encryptionService.DecryptMasterKey(encryptedMasterKey, password, []byte(login)); err == nil {
				services.SetGlobalEncryptionService(encryptionService)
			}
		}

		if err := storage.SaveLastSyncTime(time.Now().Unix()); err != nil {
		}

		return registrationOk{token: res.GetToken()}
	}
}

// getDeviceIDFromToken парсит JWT и извлекает из него device id.
func getDeviceIDFromToken(tokenString string) (string, error) {
	// JWT состоит из 3 частей, разделенных точками. Нам нужна вторая часть (payload).
	parts := strings.Split(tokenString, ".")
	if len(parts) != 3 {
		return "", fmt.Errorf("invalid token format")
	}

	token, _, err := new(jwt.Parser).ParseUnverified(tokenString, jwt.MapClaims{})
	if err != nil {
		return "", fmt.Errorf("failed to parse token: %w", err)
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok {
		deviceID, ok := claims["device_id"].(string)
		if !ok {
			return "", fmt.Errorf("device_id not found or not a string in token claims")
		}
		return deviceID, nil
	}

	return "", fmt.Errorf("invalid token claims")
}
