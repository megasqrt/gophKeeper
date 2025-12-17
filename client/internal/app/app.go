package app

import (
	"context"
	"fmt"
	"gophKeeper/client/internal/config"
	"gophKeeper/client/internal/delivery/tui"
	"gophKeeper/client/internal/domain"
	"gophKeeper/client/internal/services"
	"gophKeeper/client/internal/storage/sqlite"
	"gophKeeper/client/internal/transport"
	logger "gophKeeper/pkg/logger"
	migrations "gophKeeper/pkg/migrations/sqlite"
	"os"
	"path/filepath"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/rs/zerolog"
)

type App struct {
	Logger      *zerolog.Logger
	Storage     domain.LocalStorage
	Tui         *tui.RootModel
	program     *tea.Program
	syncService *services.SyncService
	cancel      context.CancelFunc
}

// findMigrationsPath ищет путь к миграциям, проверяя несколько вариантов
func findMigrationsPath() (string, error) {
	// Возможные пути относительно разных точек запуска
	possiblePaths := []string{
		"client/internal/storage/sqlite/migrations",
		"../internal/storage/sqlite/migrations",
		"../../internal/storage/sqlite/migrations",
	}

	// Получаем текущую рабочую директорию
	wd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get working directory: %w", err)
	}

	// Пробуем найти путь, проверяя существование директории
	for _, relPath := range possiblePaths {
		fullPath := filepath.Join(wd, relPath)
		if info, err := os.Stat(fullPath); err == nil && info.IsDir() {
			absPath, err := filepath.Abs(fullPath)
			if err != nil {
				continue
			}
			return absPath, nil
		}
	}

	// Если не найдено, пытаемся найти корень проекта (где находится go.mod)
	current := wd
	for {
		goModPath := filepath.Join(current, "go.mod")
		if _, err := os.Stat(goModPath); err == nil {
			// Нашли корень проекта
			migrationsPath := filepath.Join(current, "client", "internal", "storage", "sqlite", "migrations")
			if info, err := os.Stat(migrationsPath); err == nil && info.IsDir() {
				absPath, err := filepath.Abs(migrationsPath)
				if err != nil {
					return "", fmt.Errorf("failed to get absolute path: %w", err)
				}
				return absPath, nil
			}
		}

		parent := filepath.Dir(current)
		if parent == current {
			// Дошли до корня файловой системы
			break
		}
		current = parent
	}

	return "", fmt.Errorf("migrations directory not found. Tried paths: %v from %s", possiblePaths, wd)
}

func NewApp(ctx context.Context) (*App, error) {
	cfg, err := config.Init()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize config: %w", err)
	}

	log := logger.NewFileLoger(cfg.LogPath)
	log.Info().Msg("Client application initializing")
	log.Info().Str("log_path", cfg.LogPath).Msg("Logging to file")

	store, err := sqlite.NewSqliteStorage(cfg.DBPath, *log)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize storage: %w", err)
	}

	// Получаем абсолютный путь к миграциям
	migrationsPath, err := findMigrationsPath()
	if err != nil {
		return nil, fmt.Errorf("failed to find migrations path: %w", err)
	}

	manager, err := migrations.NewManager(cfg.DBPath, migrationsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create migration manager: %w", err)
	}

	if err := manager.Up(); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}
	log.Info().Msg("Successfully run migrations.")

	if err := transport.Init(ctx, cfg, log); err != nil {
		log.Warn().Err(err).Msg("Failed to initialize transport layer; starting in offline mode")
	}

	// Инициализируем сервис синхронизации
	syncService := services.NewSyncService(store, log)

	// 1. Создаем модель без указателя на программу.
	rootModel := tui.NewRootModel(store, cfg, syncService, log)
	// 2. Создаем программу с этой моделью.
	p := tea.NewProgram(rootModel)
	// 3. Теперь, когда программа создана, устанавливаем указатель на нее в модели.
	rootModel.SetProgram(p)

	app := &App{
		Logger:      log,
		Storage:     store,
		Tui:         &rootModel,
		program:     p,
		syncService: syncService,
	}

	return app, nil
}

// Run запускает приложение: TUI и фоновые процессы.
// Метод блокирует выполнение до завершения TUI.
func (a *App) Run(ctx context.Context) error {
	appCtx, cancel := context.WithCancel(ctx)
	a.cancel = cancel

	// Запускаем фоновую синхронизацию
	go func() {
		a.Logger.Info().Msg("Starting background synchronization process")
		ticker := time.NewTicker(10 * time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				a.Logger.Info().Msg("Running scheduled data synchronization")
				if err := a.syncService.Sync(appCtx); err != nil {
					a.Logger.Warn().Err(err).Msg("Scheduled data synchronization failed")
				}
			case <-appCtx.Done():
				a.Logger.Info().Msg("Stopping background synchronization process")
				return
			}
		}
	}()

	// Запускаем TUI (блокирующий вызов)
	if _, err := a.program.Run(); err != nil {
		a.Logger.Error().Err(err).Msg("Alas, there's been an error:")
		cancel()
		return fmt.Errorf("TUI error: %w", err)
	}

	cancel()
	return nil
}

func (a *App) Stop() {
	if a.cancel != nil {
		a.cancel()
	}
	a.Logger.Info().Msg("До свидания!")
	a.Storage.Close()
}
