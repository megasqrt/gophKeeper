package tui

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type fileInfoItem struct {
	info fs.FileInfo
	path string
}

type fileBrowserModel struct {
	table        table.Model
	items        []fileInfoItem // Храним полную информацию, т.к. таблица хранит только строки
	currentPath  string
	selectedFile string // Поле для отображения выбранного файла
	err          error
}

func newFileBrowserModel() fileBrowserModel {
	columns := []table.Column{
		{Title: "Name", Width: 40},
		{Title: "Size", Width: 10},
		{Title: "Modified", Width: 20},
	}

	tbl := table.New(
		table.WithColumns(columns),
		table.WithFocused(true),
	)

	startPath, err := os.UserHomeDir()
	if err != nil {
		startPath = "/"
	}

	return fileBrowserModel{
		table:       tbl,
		currentPath: startPath,
	}
}

func (m *fileBrowserModel) loadDirItems(path string) tea.Cmd {
	return func() tea.Msg {
		files, err := os.ReadDir(path)
		if err != nil {
			return err
		}

		items := []fileInfoItem{}
		// Добавляем ".." для навигации вверх
		if path != "/" {
			// Получаем info для родительской директории
			parentInfo, statErr := os.Stat(filepath.Dir(path))
			if statErr == nil {
				// Используем специальное имя ".." для отображения
				parentInfo = dirInfo{"..", parentInfo}
				items = append(items, fileInfoItem{info: parentInfo, path: filepath.Dir(path)})
			}
		}

		// Сортируем: сначала папки, потом файлы
		sort.Slice(files, func(i, j int) bool {
			infoI, _ := files[i].Info()
			infoJ, _ := files[j].Info()
			if infoI.IsDir() != infoJ.IsDir() {
				return infoI.IsDir()
			}
			return files[i].Name() < files[j].Name()
		})

		for _, f := range files {
			info, err := f.Info()
			if err != nil {
				continue
			}
			items = append(items, fileInfoItem{info: info, path: filepath.Join(path, info.Name())})
		}
		return items
	}
}

func (m fileBrowserModel) Init() tea.Cmd {
	return m.loadDirItems(m.currentPath)
}

func (m fileBrowserModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.table.SetHeight(msg.Height - 5)
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "q":
			// Если мы не в корне, "esc" поднимает на уровень выше
			if m.currentPath != "/" {
				m.currentPath = filepath.Dir(m.currentPath)
				return m, m.loadDirItems(m.currentPath)
			}
			// Если в корне, то "esc" закрывает браузер (отправляем FileSelectMsg без файлов)
			return m, func() tea.Msg { return FileSelectMsg{Files: nil} }

		case "enter":

			if len(m.items) == 0 {
				return m, nil
			}
			selectedItem := m.items[m.table.Cursor()]

			if selectedItem.info.IsDir() {
				m.currentPath = selectedItem.path
				//		m.err = fmt.Errorf("items dir wtf")
				return m, m.loadDirItems(m.currentPath)
			}
			// Файл выбран, отправляем сообщение
				//	m.err = fmt.Errorf("items send")
			return m, func() tea.Msg {
				return FileSelectMsg{Files: []string{selectedItem.path}}
			}
		}

	case []fileInfoItem:
		m.items = msg
		rows := make([]table.Row, len(m.items))
		for i, item := range m.items {
			icon := "📄"
			if item.info.IsDir() {
				icon = "📁"
			}
			rows[i] = table.Row{
				fmt.Sprintf("%s %s", icon, item.info.Name()),
				formatFileSize(item.info.Size()),
				item.info.ModTime().Format("2006-01-02 15:04"),
			}
		}
		m.table.SetRows(rows)

	case error:
		m.err = msg
	}

	// Обновляем таблицу, чтобы она обработала навигацию (вверх/вниз)
	m.table, cmd = m.table.Update(msg)

	// Обновляем поле selectedFile после всех перемещений
	if len(m.items) > 0 && m.table.Cursor() < len(m.items) {
		selectedItem := m.items[m.table.Cursor()]
		if !selectedItem.info.IsDir() {
			m.selectedFile = selectedItem.path
		} else {
			m.selectedFile = "" // Это директория, поле пустое
		}
	}
	return m, cmd
}

func (m fileBrowserModel) View() string {
	if m.err != nil {
		return fmt.Sprintf("Error: %v", m.err)
	}
	title := lipgloss.NewStyle().Bold(true).Render("File Browser: " + m.currentPath)
	selected := lipgloss.NewStyle().Foreground(lipgloss.Color("213")).Render("Selected: " + m.selectedFile)
	help := helpStyle.Render("(↑/↓) navigate | (enter) select | (esc/q) up/back")

	view := lipgloss.JoinVertical(lipgloss.Left, title, baseStyle.Render(m.table.View()), selected, help)
	return view
}

// dirInfo - это обертка для fs.FileInfo, позволяющая подменить имя (например, для "..")
type dirInfo struct {
	name string
	fs.FileInfo
}

func (d dirInfo) Name() string {
	return d.name
}

func (d dirInfo) Size() int64 {
	return 0 // У родительской директории не показываем размер
}

func (d dirInfo) ModTime() time.Time {
	return d.FileInfo.ModTime()
}
