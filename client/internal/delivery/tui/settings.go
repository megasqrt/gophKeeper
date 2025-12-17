package tui

import (
	"fmt"
	"gophKeeper/client/internal/config"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type settingsKeyMap struct {
	Save key.Binding
	Up   key.Binding
	Down key.Binding
	Back key.Binding
}

type SettingsModel struct {
	inputs     []textinput.Model
	keys       []string // Для сохранения порядка ключей
	focusIndex int
	err        error
	infoMsg    string
	keysMap    settingsKeyMap
	width      int
}

func NewSettingsModel() *SettingsModel {
	m := &SettingsModel{
		keysMap: settingsKeyMap{
			Save: key.NewBinding(key.WithKeys("ctrl+s"), key.WithHelp("ctrl+s", "save")),
			Up:   key.NewBinding(key.WithKeys("up"), key.WithHelp("↑", "up")),
			Down: key.NewBinding(key.WithKeys("down"), key.WithHelp("↓", "down")),
			Back: key.NewBinding(key.WithKeys("esc", "q"), key.WithHelp("esc/q", "back")),
		},
	}
	m.Load()
	return m
}

func (m *SettingsModel) Load() {
	settings := config.GetSettings()
	keys := make([]string, 0, len(settings))
	for k := range settings {
		keys = append(keys, k)
	}
	sort.Strings(keys) // Сортируем ключи для консистентного порядка

	m.keys = keys
	m.inputs = make([]textinput.Model, len(settings))

	for i, k := range m.keys {
		ti := textinput.New()
		ti.SetValue(fmt.Sprintf("%v", settings[k]))
		ti.CharLimit = 256
		ti.Width = 50
		m.inputs[i] = ti
	}

	if len(m.inputs) > 0 {
		m.inputs[0].Focus()
	}
}

func (m *SettingsModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m *SettingsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keysMap.Back):
			return m, func() tea.Msg { return backToMenuMsg{} }

		case key.Matches(msg, m.keysMap.Save):
			m.err = nil
			m.infoMsg = ""
			newSettings := make(map[string]interface{})
			for i, k := range m.keys {
				newSettings[k] = m.inputs[i].Value()
			}
			if err := config.UpdateSettings(newSettings); err != nil {
				m.err = err
			} else {
				m.infoMsg = "✅ Settings saved successfully!"
			}
			return m, nil

		case key.Matches(msg, m.keysMap.Up):
			m.focusIndex--
		case key.Matches(msg, m.keysMap.Down):
			m.focusIndex++
		}

		// Циклическое переключение
		if m.focusIndex < 0 {
			m.focusIndex = len(m.inputs) - 1
		}
		if m.focusIndex >= len(m.inputs) {
			m.focusIndex = 0
		}

		// Устанавливаем фокус
		for i := range m.inputs {
			if i == m.focusIndex {
				cmds = append(cmds, m.inputs[i].Focus())
			} else {
				m.inputs[i].Blur()
			}
		}
	}

	// Обновляем только активное поле ввода
	if len(m.inputs) > 0 {
		var cmd tea.Cmd
		m.inputs[m.focusIndex], cmd = m.inputs[m.focusIndex].Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m *SettingsModel) View() string {
	var b strings.Builder

	b.WriteString(lipgloss.NewStyle().Bold(true).Render("Application Settings"))
	b.WriteString("\n\n")

	for i, k := range m.keys {
		keyStyle := lipgloss.NewStyle().Width(20).Align(lipgloss.Right).PaddingRight(2)
		if m.focusIndex == i {
			keyStyle = keyStyle.Foreground(lipgloss.Color("205"))
		}
		row := lipgloss.JoinHorizontal(lipgloss.Top,
			keyStyle.Render(k),
			m.inputs[i].View(),
		)
		b.WriteString(row)
		b.WriteString("\n")
	}

	// Help
	help := fmt.Sprintf("%s | %s | %s",
		m.keysMap.Up.Help().Key+" "+m.keysMap.Up.Help().Desc,
		m.keysMap.Down.Help().Key+" "+m.keysMap.Down.Help().Desc,
		m.keysMap.Save.Help().Key+" "+m.keysMap.Save.Help().Desc,
	)
	b.WriteString("\n" + helpStyle.Render(help))

	// Сообщения об ошибках и успехе
	if m.err != nil {
		errorMsg := lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Render("Error: " + m.err.Error())
		b.WriteString("\n" + errorMsg)
	}
	if m.infoMsg != "" {
		infoMsg := lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Render(m.infoMsg)
		b.WriteString("\n" + infoMsg)
	}

	return docStyle.Render(b.String())
}
