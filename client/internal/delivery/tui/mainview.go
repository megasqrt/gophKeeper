package tui

import (
	"fmt"
	"gophKeeper/client/internal/config"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type MainViewModel struct {
	tabs      []string
	activeTab int
	login     string
	// TODO: deviceName нужно будет получать через отдельный gRPC вызов
	deviceName   string
	lastSync     time.Time
	serverOnline bool
	cfg          *config.Config
	storage      LocalStorage
}

type serverStatusMsg struct{ online bool }

func NewMainViewModel(login, deviceName string, lastSync time.Time, cfg *config.Config, storage LocalStorage) tea.Model {
	return MainViewModel{
		tabs:         []string{"Info", "Password", "Card", "Text", "Settings"},
		activeTab:    0,
		login:        login,
		deviceName:   deviceName,
		lastSync:     lastSync,
		serverOnline: true, // Изначально считаем, что онлайн
		cfg:          cfg,
		storage:      storage,
	}
}

func (m MainViewModel) Init() tea.Cmd {
	// Запускаем пингер сервера при инициализации
	return tea.Batch(checkServer(m.cfg), tea.Every(10*time.Minute, func(t time.Time) tea.Msg {
		return checkServer(m.cfg)()
	}))
}

func (m MainViewModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case serverStatusMsg:
		m.serverOnline = msg.online
		return m, nil // Никакой новой команды не нужно
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "right", "l":
			m.activeTab = (m.activeTab + 1) % len(m.tabs)
		case "left", "h":
			m.activeTab = (m.activeTab - 1 + len(m.tabs)) % len(m.tabs)
		}
	}
	return m, nil
}

func (m MainViewModel) View() string {
	doc := strings.Builder{}

	var renderedTabs []string
	for i, t := range m.tabs {
		var style lipgloss.Style
		if i == m.activeTab {
			style = lipgloss.NewStyle().
				Background(lipgloss.Color("63")).
				Foreground(lipgloss.Color("230")).
				Padding(1, 2)
		} else {
			style = lipgloss.NewStyle().Padding(1, 2)
		}
		renderedTabs = append(renderedTabs, style.Render(t))
	}

	// Добавляем иконку статуса сервера
	serverStatusIcon := "❌" // Офлайн
	if m.serverOnline {
		serverStatusIcon = "✔️" // Онлайн
	}
	statusLine := lipgloss.NewStyle().Align(lipgloss.Right).SetString(fmt.Sprintf("Server: %s", serverStatusIcon))
	tabsAndStatus := lipgloss.JoinHorizontal(lipgloss.Top, lipgloss.JoinHorizontal(lipgloss.Top, renderedTabs...), statusLine.Render())

	doc.WriteString(tabsAndStatus)
	doc.WriteString("\n\n")

	switch m.tabs[m.activeTab] {
	case "Info":
		infoStyle := lipgloss.NewStyle().PaddingLeft(2)
		doc.WriteString(infoStyle.Render(fmt.Sprintf("User: %s\n", m.login)))
		doc.WriteString(infoStyle.Render(fmt.Sprintf("Device: %s\n", m.deviceName)))
		if m.lastSync.IsZero() {
			doc.WriteString(infoStyle.Render("Last sync: never\n"))
		} else {
			doc.WriteString(infoStyle.Render(fmt.Sprintf("Last sync: %s\n", m.lastSync.Format(time.RFC1123))))
		}
	default:
		doc.WriteString(fmt.Sprintf("Content for %s tab.", m.tabs[m.activeTab]))
	}

	return doc.String()
}

// checkServer возвращает команду для проверки доступности сервера.
func checkServer(cfg *config.Config) tea.Cmd {
	return func() tea.Msg {
		serverHost := strings.Split(cfg.ServerAddress, ":")[0]
		creds, err := credentials.NewClientTLSFromFile(cfg.CACertPath, serverHost)
		if err != nil {
			return serverStatusMsg{online: false}
		}

		// Используем короткий таймаут для проверки соединения
		// ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		// defer cancel()

		conn, err := grpc.NewClient(cfg.ServerAddress, grpc.WithTransportCredentials(creds))
		if err != nil {
			return serverStatusMsg{online: false}
		}
		conn.Close() // Просто проверяем возможность подключения

		// Можно добавить реальный health check эндпоинт в будущем
		// Например, client.HealthCheck(ctx, &pb.HealthCheckRequest{})

		return serverStatusMsg{online: true}
	}
}
