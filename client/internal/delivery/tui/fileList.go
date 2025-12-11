package tui

import (
	"encoding/base64"
	"fmt"
	"gophKeeper/client/internal/domain"
	model "gophKeeper/pkg/grpchelper"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Модель для элемента файла
type FileItem struct {
	model.FileData
	Name       string
	Path       string
	Size       int64
	IsUploaded bool
}

func (f FileItem) Title() string {
	status := "📄"
	if f.IsUploaded {
		status = "✅"
	}
	return fmt.Sprintf("%s %s", status, f.Name)
}

func (f FileItem) FilterValue() string { return f.Name }

type fileItemDelegate struct{}

func (d fileItemDelegate) Height() int                               { return 2 }
func (d fileItemDelegate) Spacing() int                              { return 1 }
func (d fileItemDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }
func (d fileItemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(FileItem)
	if !ok {
		return
	}

	titleStr := i.Title()

	if index == m.Index() {
		title := selectedItemStyle.Render("> " + titleStr)
		fmt.Fprint(w, title)
	} else {
		title := itemStyle.Render(titleStr)
		fmt.Fprint(w, title)
	}
}

// Сообщения для загрузки файлов
type FileSelectMsg struct {
	Files []string
}

type UploadProgressMsg struct {
	FileName string
	Progress float64
}

type UploadCompleteMsg struct {
	FileName string
	Error    error
}

type DownloadCompleteMsg struct {
	FileName string
	SavePath string
	Error    error
}

type FilePathInputMsg struct {
	Path string
}

// Хоткеи
type fileKeyMap struct {
	AddFiles key.Binding
	Upload   key.Binding
	Download key.Binding
	Delete   key.Binding
	Back     key.Binding
	Refresh  key.Binding
}

type fileListViewState int

const (
	mainListState fileListViewState = iota
	browsingState
	confirmDeleteState
)

type deleteFileMsg struct{ confirmed bool }

// Модель загрузки файлов
type FileUploadModel struct {
	list           list.Model
	viewport       viewport.Model
	progress       progress.Model
	storage        domain.LocalStorage
	width, height  int
	err            error
	infoMsg        string
	uploading      bool
	uploadProgress float64
	currentFile    *FileItem
	keys           fileKeyMap
	p              *tea.Program // Ссылка на программу для отправки сообщений из горутин
	state          fileListViewState
	browser        fileBrowserModel
	confirmModel   ConfirmModel
}

func NewFileUploadModel(storage domain.LocalStorage) *FileUploadModel {
	// Настройка списка файлов
	l := list.New([]list.Item{}, fileItemDelegate{}, 0, 15)
	l.Title = "📁 File Manager"
	l.SetShowStatusBar(true)
	l.SetStatusBarItemName("file", "files")
	l.SetShowHelp(false)
	l.SetFilteringEnabled(true)

	// Настройка прогресс-бара
	prog := progress.New(progress.WithDefaultGradient())
	prog.Width = 60

	// Настройка вьюпорта для информации о файле
	vp := viewport.New(60, 10)

	keys := fileKeyMap{
		AddFiles: key.NewBinding(
			key.WithKeys("a"),
			key.WithHelp("a", "add files"),
		),
		Upload: key.NewBinding(
			key.WithKeys("u"),
			key.WithHelp("u", "upload selected"),
		),
		Download: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "save to disk"),
		),
		Delete: key.NewBinding(
			key.WithKeys("ctrl+d"),
			key.WithHelp("ctrl+d", "delete"),
		),
		Back: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "back"),
		),
		Refresh: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "refresh"),
		),
	}

	m := &FileUploadModel{
		list:     l,
		viewport: vp,
		progress: prog,
		storage:  storage,
		keys:     keys,
		state:    mainListState,
		browser:  newFileBrowserModel(),
	}

	m.confirmModel = NewConfirmModel("Default prompt", func(confirmed bool) tea.Cmd {
		return func() tea.Msg {
			return deleteFileMsg{confirmed: confirmed}
		}
	})

	return m
}

func (m *FileUploadModel) SetProgram(p *tea.Program) {
	m.p = p
}

func (m *FileUploadModel) Init() tea.Cmd {
	return m.loadFiles
}

func (m *FileUploadModel) Load() tea.Cmd {
	// Загружаем список файлов из хранилища
	files, err := m.storage.GetFiles()
	if err != nil {
		return func() tea.Msg { return err }
	}

	var items []list.Item
	for _, file := range files {
		if !file.Deleted {
			items = append(items, FileItem{
				FileData:   file,
				Name:       file.Name,
				Path:       file.Metadata, // Используем Metadata для хранения локального пути
				Size:       file.Size,
				IsUploaded: true,
			})
		}
	}

	return func() tea.Msg { return items }
}

func (m *FileUploadModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	if m.state == confirmDeleteState {
		newConfirmModel, newCmd := m.confirmModel.Update(msg)
		if _, ok := newConfirmModel.(ConfirmModel); ok {
			m.confirmModel = newConfirmModel.(ConfirmModel)
		}
		return m, newCmd
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.updateLayout()
		m.confirmModel.setSize(msg.Width, msg.Height)

	case tea.KeyMsg:
		if m.uploading {
			break // Игнорируем клавиши во время загрузки
		}

		switch {
		case key.Matches(msg, m.keys.Back):
			if m.state == browsingState {
				m.state = mainListState
				return m, nil
			}
			return m, func() tea.Msg { return backToMenuMsg{} }

		case key.Matches(msg, m.keys.AddFiles):
			m.state = browsingState
			m.browser = newFileBrowserModel()
			return m, m.browser.Init()

		case key.Matches(msg, m.keys.Upload):
			if item := m.list.SelectedItem(); item != nil {
				if fileItem, ok := item.(FileItem); ok && !fileItem.IsUploaded {
					return m, m.startUpload(fileItem)
				}
			}

		case key.Matches(msg, m.keys.Download):
			if item := m.list.SelectedItem(); item != nil {
				if fileItem, ok := item.(FileItem); ok && fileItem.IsUploaded {
					return m, m.downloadSelectedFile(fileItem)
				}
			}

		case key.Matches(msg, m.keys.Delete):
			if m.state == mainListState {
				if item, ok := m.list.SelectedItem().(FileItem); ok {
					m.state = confirmDeleteState
					m.confirmModel.SetPrompt(fmt.Sprintf("файл '%s'", item.Name))
					return m, nil
				}
			}
		case key.Matches(msg, m.keys.Refresh):
			return m, m.loadFiles
		}

	case FileSelectMsg:
		m.state = mainListState
		if len(msg.Files) > 0 {
			// Файл был выбран, добавляем его и сразу начинаем загрузку.
			path := msg.Files[0]
			info, err := os.Stat(path)
			if err != nil {
				m.err = err
				return m, nil
			}
			fileItem := FileItem{
				Name:       filepath.Base(path),
				Path:       path,
				Size:       info.Size(),
				IsUploaded: false,
			}
			m.list.InsertItem(len(m.list.Items()), fileItem)
			return m, m.startUpload(fileItem)
		}
		// Если msg.Files пуст (нажали esc в браузере), просто возвращаемся.
		return m, nil

	case UploadProgressMsg:
		if m.uploading {
			m.uploadProgress = msg.Progress
			// Обновляем прогресс в модели
			cmd = m.progress.SetPercent(m.uploadProgress)
			cmds = append(cmds, cmd)
			return m, tea.Batch(cmds...) //m,nil
		}

	case UploadCompleteMsg:
		m.uploading = false
		m.uploadProgress = 0
		m.currentFile = nil
		// Сбрасываем прогресс
		cmd = m.progress.SetPercent(0)
		cmds = append(cmds, cmd)
		if msg.Error != nil {
			m.err = msg.Error
		}
		return m, tea.Batch(append(cmds, m.loadFiles)...) // Перезагружаем список

	case deleteFileMsg:
		m.state = mainListState
		if msg.confirmed {
			if item, ok := m.list.SelectedItem().(FileItem); ok {
				// Используем DeleteFileByID для soft delete (помечает как удаленный)
				err := m.storage.DeleteFileByID(item.GetLocalID())
				if err != nil {
					m.err = err
				} else {
					return m, m.loadFiles
				}
			}
		}
		return m, nil

	case DownloadCompleteMsg:
		m.err = nil    // Сбрасываем предыдущую ошибку
		m.infoMsg = "" // Сбрасываем предыдущее инфо-сообщение
		if msg.Error != nil {
			m.err = msg.Error
		} else {
			m.infoMsg = fmt.Sprintf("✅ File '%s' saved to %s", msg.FileName, msg.SavePath)
		}

	case []list.Item:
		m.list.SetItems(msg)

	case error:
		m.err = msg
	}

	// После обработки глобальных сообщений, передаем управление дочерней модели
	if m.state == browsingState {
		newBrowser, browserCmd := m.browser.Update(msg)
		m.browser = newBrowser.(fileBrowserModel)
		return m, browserCmd
	}

	// Обновляем вложенные модели
	if !m.uploading {
		m.list, cmd = m.list.Update(msg)
		cmds = append(cmds, cmd)
	}

	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	newProgress, cmd := m.progress.Update(msg)
	m.progress = newProgress.(progress.Model)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m *FileUploadModel) View() string {
	var mainView string
	if m.state == browsingState {
		return m.browser.View()
	}

	var sections []string

	// Заголовок
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("63")).
		Padding(0, 1).
		Render("📁 File Upload Manager")
	sections = append(sections, title)

	// Прогресс загрузки (если есть)
	if m.uploading && m.currentFile != nil {
		progressSection := lipgloss.JoinVertical(lipgloss.Left,
			lipgloss.NewStyle().Foreground(lipgloss.Color("213")).Render(
				fmt.Sprintf("Uploading: %s", m.currentFile.Name)),
			m.progress.View(), // ← Исправлено: без аргументов
		)
		sections = append(sections, progressSection)
	}

	// Основной контент
	mainContent := lipgloss.JoinHorizontal(lipgloss.Top,
		lipgloss.NewStyle().Width(50).Render(m.list.View()),
		lipgloss.NewStyle().Width(m.width-55).Render(m.fileInfoView()),
	)
	sections = append(sections, mainContent)

	// Статус бар
	sections = append(sections, m.helpView())

	// Ошибки
	if m.err != nil {
		errorSection := lipgloss.NewStyle().
			Foreground(lipgloss.Color("9")).
			Render("Error: " + m.err.Error())
		sections = append(sections, errorSection)
	}
	// Информационные сообщения
	if m.infoMsg != "" {
		infoSection := lipgloss.NewStyle().
			Foreground(lipgloss.Color("10")). // Зеленый цвет для успеха
			Render(m.infoMsg)
		sections = append(sections, infoSection)
	}

	mainView = lipgloss.JoinVertical(lipgloss.Left, sections...)

	if m.state == confirmDeleteState {
		dialog := m.confirmModel.View()
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, dialog)
	}

	return mainView
}

func (m *FileUploadModel) fileInfoView() string {
	if item := m.list.SelectedItem(); item != nil {
		if file, ok := item.(FileItem); ok {
			info := []string{
				lipgloss.NewStyle().Bold(true).Render("File Info:"),
				fmt.Sprintf("Name: %s", file.Name),
				fmt.Sprintf("Size: %s", model.FormatFileSize(file.Size)),
				fmt.Sprintf("Path: %s", file.Path),
				fmt.Sprintf("Status: %s", map[bool]string{true: "✅ Uploaded", false: "📄 Local"}[file.IsUploaded]),
			}
			return strings.Join(info, "\n")
		}
	}
	return "Select a file to view details"
}

func (m *FileUploadModel) helpView() string {
	var helps []string
	helps = append(helps, m.keys.AddFiles.Help().Key+" "+m.keys.AddFiles.Help().Desc)
	helps = append(helps, m.keys.Upload.Help().Key+" "+m.keys.Upload.Help().Desc)
	helps = append(helps, m.keys.Download.Help().Key+" "+m.keys.Download.Help().Desc)
	helps = append(helps, m.keys.Delete.Help().Key+" "+m.keys.Delete.Help().Desc)
	helps = append(helps, m.keys.Refresh.Help().Key+" "+m.keys.Refresh.Help().Desc)
	helps = append(helps, m.keys.Back.Help().Key+" "+m.keys.Back.Help().Desc)

	return helpStyle.Render(strings.Join(helps, " | "))
}

func (m *FileUploadModel) updateLayout() {
	listHeight := m.height - 6 // Заголовок + хелп + отступы
	m.list.SetSize(50, listHeight)
	m.viewport.Width = m.width - 55
	m.viewport.Height = listHeight
	m.progress.Width = m.width - 20
}

func (m *FileUploadModel) loadFiles() tea.Msg {
	return m.Load()()
}

func (m *FileUploadModel) startUpload(file FileItem) tea.Cmd {
	m.uploading = true
	m.currentFile = &file
	m.uploadProgress = 0

	// Запускаем загрузку в горутине, не блокируя UI
	return func() tea.Msg {
		go m.uploadFile(file)
		return nil // Не возвращаем сообщение, чтобы не блокировать
	}
}

func (m *FileUploadModel) uploadFile(file FileItem) {
	// Реальная логика сохранения файла
	f, err := os.Open(file.Path)
	if err != nil {
		m.p.Send(UploadCompleteMsg{FileName: file.Name, Error: err})
		return
	}
	defer f.Close()

	// Создаем обертку для отслеживания прогресса чтения
	reader := &progressReader{
		Reader:   f,
		Total:    file.Size,
		fileName: file.Name,
		program:  m.p,
	}

	// Читаем файл в память, прогресс будет отправляться автоматически
	data, err := io.ReadAll(reader)
	if err != nil {
		m.p.Send(UploadCompleteMsg{FileName: file.Name, Error: err})
		return
	}

	// Сохраняем в хранилище
	fileData := &model.FileData{
		Name:     file.Name,
		Size:     file.Size,
		Metadata: file.Path, // Сохраняем локальный путь в метаданные
		// содержимое файла (data) передается вторым аргументом
	}
	saveErr := m.storage.SaveFile(fileData, data)
	m.p.Send(UploadCompleteMsg{FileName: file.Name, Error: saveErr})
}

// progressReader - это обертка для io.Reader, которая отслеживает прогресс чтения.
type progressReader struct {
	io.Reader
	Total     int64
	bytesRead int64 // Поле для отслеживания прочитанных байт
	fileName  string
	program   *tea.Program
}

func (r *progressReader) Read(p []byte) (n int, err error) {
	n, err = r.Reader.Read(p)
	r.bytesRead += int64(n)
	progress := float64(r.bytesRead) / float64(r.Total)
	r.program.Send(UploadProgressMsg{FileName: r.fileName, Progress: progress})
	return
}

func (m *FileUploadModel) downloadSelectedFile(file FileItem) tea.Cmd {
	return func() tea.Msg {
		fileDataMap, err := m.storage.GetFileByID(file.LocalID)
		if err != nil {
			return DownloadCompleteMsg{FileName: file.Name, Error: fmt.Errorf("failed to get file from storage: %w", err)}
		}

		// Проверяем наличие данных
		dataRaw, exists := fileDataMap["data"]
		if !exists || dataRaw == nil {
			return DownloadCompleteMsg{FileName: file.Name, Error: fmt.Errorf("file content not found in storage")}
		}

		var fileContent []byte
		var decodeErr error

		// Пытаемся получить данные как []byte (наиболее вероятный случай)
		switch v := dataRaw.(type) {
		case []byte:
			fileContent = v
		case string:
			// Если данные в виде строки, пытаемся декодировать из base64
			fileContent, decodeErr = base64.StdEncoding.DecodeString(v)
			if decodeErr != nil {
				// Если не base64, используем строку как есть
				fileContent = []byte(v)
			}
		case []interface{}:
			// Если это slice интерфейсов (может быть при JSON unmarshal)
			fileContent = make([]byte, len(v))
			for i, val := range v {
				if b, ok := val.(byte); ok {
					fileContent[i] = b
				} else if n, ok := val.(float64); ok {
					// JSON числа могут быть float64
					fileContent[i] = byte(n)
				} else {
					return DownloadCompleteMsg{
						FileName: file.Name,
						Error:    fmt.Errorf("invalid data format in storage: cannot convert []interface{} element type %T to byte", val),
					}
				}
			}
		default:
			return DownloadCompleteMsg{
				FileName: file.Name,
				Error:    fmt.Errorf("invalid data format in storage: expected []byte or string, got %T (value: %v)", dataRaw, dataRaw),
			}
		}

		// Сохраняем в /tmp/
		savePath := filepath.Join(os.TempDir(), file.Name)
		err = os.WriteFile(savePath, fileContent, 0644)
		if err != nil {
			return DownloadCompleteMsg{FileName: file.Name, Error: fmt.Errorf("could not write file to disk: %w", err)}
		}

		return DownloadCompleteMsg{FileName: file.Name, SavePath: savePath}
	}
}
