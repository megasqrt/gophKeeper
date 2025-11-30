package tui

import (
	"fmt"
	"gophKeeper/client/internal/domain/model"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type textItemDelegate struct{}

func (d textItemDelegate) Height() int                               { return 1 }
func (d textItemDelegate) Spacing() int                              { return 0 }
func (d textItemDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }
func (d textItemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(textItem)
	if !ok {
		return
	}

	str := i.Title()

	if index == m.Index() {
		title := selectedItemStyle.Render("> " + str)
		fmt.Fprint(w, title)
	} else {
		title := itemStyle.Render(str)
		fmt.Fprint(w, title)
	}
}

type textItem struct {
	model.TextData
}

func (i textItem) Title() string       { return i.TextData.Title }
func (i textItem) FilterValue() string { return i.TextData.Title }

type keyMap struct {
	SwitchFocus key.Binding
	Save        key.Binding
	Back        key.Binding
	NewItem     key.Binding
	DeleteItem  key.Binding
}

type TextEditModel struct {
	list          list.Model
	editor        textarea.Model
	titleInput    textinput.Model
	storage       LocalStorage
	state         viewState
	width, height int
	err           error
	keys          keyMap
}

func NewTextEditModel(storage LocalStorage) *TextEditModel {
	// 1. Создаем список (list)
	l := list.New([]list.Item{}, textItemDelegate{}, 0, 15)
	l.Title = "Your Secure Notes"
	l.Styles.Title = listTitleStyle
	l.SetShowStatusBar(true)
	l.SetShowPagination(true)
	l.SetStatusBarItemName("note", "notes")

	// Убираем стандартную справку
	l.SetShowHelp(false)

	// 2. Создаем текстовый редактор (textarea)
	t := textarea.New()
	t.Placeholder = "Select a note to view its content..."
	t.ShowLineNumbers = true

	// Поле для ввода заголовка
	ti := textinput.New()
	ti.Placeholder = "Note title..."
	ti.CharLimit = 100
	ti.Width = 30

	// 3. Определяем горячие клавиши
	keys := keyMap{
		SwitchFocus: key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "switch focus")),
		Save:        key.NewBinding(key.WithKeys("ctrl+s"), key.WithHelp("ctrl+s", "save")),
		Back:        key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back to menu")),
		NewItem:     key.NewBinding(key.WithKeys("ctrl+n"), key.WithHelp("ctrl+n", "new note")),
		DeleteItem:  key.NewBinding(key.WithKeys("delete"), key.WithHelp("del", "delete note")),
	}

	// 4. Собираем модель
	m := &TextEditModel{
		titleInput: ti,
		list:       l,
		editor:     t,
		storage:    storage,
		state:      tableView, // По умолчанию фокус на списке
		keys:       keys,
	}

	return m
}

// Load данные из хранилища и обновляет список.
func (m *TextEditModel) Load() {
	textsData, err := m.storage.GetTexts()
	if err != nil {
		m.err = fmt.Errorf("could not load notes: %w", err)
		m.list.SetItems(nil)
		return
	}

	items := make([]list.Item, len(textsData))
	for i, data := range textsData {
		items[i] = textItem{
			model.TextData{
				ID:    data["id"],
				Title: data["title"],
				Text:  data["text"],
			},
		}
	}
	m.list.SetItems(items)
	m.syncEditor()
}

// syncEditor обновляет содержимое редактора в соответствии с выбранным элементом списка.
func (m *TextEditModel) syncEditor() {
	selectedItem, ok := m.list.SelectedItem().(textItem)
	if !ok {
		m.titleInput.SetValue("")
		m.editor.Reset()
		return
	}

	m.titleInput.SetValue(selectedItem.Title())
	m.editor.SetValue(selectedItem.Text)
}

func (m *TextEditModel) saveNote() {
	selectedItem, ok := m.list.SelectedItem().(textItem)
	if !ok {
		return
	}

	data := map[string]string{
		"id":    selectedItem.ID,
		"title": m.titleInput.Value(),
		"text":  m.editor.Value(),
	}
	var err error
	if selectedItem.ID == "" { // Новый элемент без ID
		err = m.storage.SaveText(data)
	} else {
		err = m.storage.UpdateText(data)
	}
	if err == nil {
		m.err = nil // Сбрасываем ошибку при успехе
		m.Load()    // Перезагружаем, чтобы обновить данные
	} else {
		m.err = fmt.Errorf("could not save note: %w", err)
	}
}

func (m *TextEditModel) Init() tea.Cmd {
	return m.editor.Focus()
}

func (m *TextEditModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.height = msg.Height
		m.width = msg.Width
		listWidth := int(float64(msg.Width) * 0.4) // 40% ширины для списка
		editorWidth := msg.Width - listWidth
		m.list.SetHeight(msg.Height - 2) // -2 для рамки и строки помощи
		m.list.SetWidth(listWidth)
		m.titleInput.Width = editorWidth - 4 // отступы
		m.editor.SetWidth(editorWidth)
		m.editor.SetHeight(msg.Height - 6) // Оставляем место для поля заголовка и отступов
		return m, nil

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Back) || msg.String() == "q":
			return m, func() tea.Msg { return backToMenuMsg{} }

		case key.Matches(msg, m.keys.SwitchFocus):
			// Простое переключение между списком и формой (редактором)
			if m.state == tableView {
				m.state = formView
				cmd = m.titleInput.Focus() // Фокус на заголовок при переходе в форму
			} else { // formView
				m.state = tableView
				m.titleInput.Blur()
				m.editor.Blur()
				m.saveNote() // Автосохранение при выходе из формы
			}
			cmds = append(cmds, cmd)

		case key.Matches(msg, m.keys.Save):
			m.saveNote()

		case key.Matches(msg, m.keys.NewItem):
			newItem := textItem{model.TextData{ID: "", Title: "Новая заметка", Text: ""}}
			m.list.InsertItem(0, newItem)
			m.list.Select(0)
			m.syncEditor()
			m.state = formView
			return m, m.titleInput.Focus()

		case key.Matches(msg, m.keys.DeleteItem):
			if m.state == tableView {
				selectedItem, ok := m.list.SelectedItem().(textItem)
				if ok && selectedItem.ID != "" {
					err := m.storage.DeleteText(selectedItem.ID)
					if err == nil {
						m.err = nil
						m.Load()
					} else {
						m.err = fmt.Errorf("could not delete note: %w", err)
					}
				}
			}
		}
	}

	// Обновляем компоненты в зависимости от состояния
	if m.state == tableView {
		m.list, cmd = m.list.Update(msg)
		cmds = append(cmds, cmd)
		m.syncEditor() // Обновляем редактор при навигации по списку
	} else {
		// В режиме формы обновляем оба поля ввода
		m.titleInput, cmd = m.titleInput.Update(msg)
		cmds = append(cmds, cmd)
		m.editor, cmd = m.editor.Update(msg)
		cmds = append(cmds, cmd) // cmd будет перезаписан, но это нормально
	}

	return m, tea.Batch(cmds...)
}

func (m *TextEditModel) View() string {
	listView := m.list.View()

	rightPane := lipgloss.JoinVertical(lipgloss.Left,
		m.titleInput.View(),
		m.editor.View(),
	)

	// Добавляем рамку к компоненту в фокусе
	if m.state == formView {
		rightPane = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("205")).Render(rightPane)
	} else {
		listView = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("205")).Render(listView)
	}

	help := m.helpView()

	// Соединяем все части вместе
	mainView := lipgloss.JoinVertical(lipgloss.Left,
		lipgloss.JoinHorizontal(lipgloss.Top, listView, rightPane),
		help,
	)

	if m.err != nil {
		errorText := lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Render("Error: " + m.err.Error())
		return lipgloss.JoinVertical(lipgloss.Left, mainView, errorText)
	}
	return mainView
}

func (m *TextEditModel) helpView() string {
	var parts []string
	parts = append(parts, m.keys.NewItem.Help().Key+" "+m.keys.NewItem.Help().Desc)
	parts = append(parts, m.keys.DeleteItem.Help().Key+" "+m.keys.DeleteItem.Help().Desc)
	parts = append(parts, m.keys.SwitchFocus.Help().Key+" "+m.keys.SwitchFocus.Help().Desc)
	parts = append(parts, m.keys.Save.Help().Key+" "+m.keys.Save.Help().Desc)
	parts = append(parts, m.keys.Back.Help().Key+" "+m.keys.Back.Help().Desc)

	return helpStyle.Render(strings.Join(parts, " | "))
}
