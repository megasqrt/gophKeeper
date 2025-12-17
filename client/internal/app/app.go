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
	Logger  *zerolog.Logger
	Storage domain.LocalStorage
	Tui     *tui.RootModel
}

func NewApp(ctx context.Context) *App {

	cfg, err := config.Init()
	if err != nil {
		panic(fmt.Sprintf("Failed to initialize config: %v", err))
	}

	log := logger.NewFileLoger(cfg.LogPath)
	log.Info().Msg("Client application starting")
	log.Info().Str("log_path", cfg.LogPath).Msg("Logging to file")

	store, err := sqlite.NewSqliteStorage(cfg.DBPath, *log)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize storage")
	}

	// Получаем абсолютный путь к миграциям
	migrationsPath, err := filepath.Abs("client/internal/storage/sqlite/migrations")
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to get absolute path to migrations")
	}

	manager, err := migrations.NewManager(cfg.DBPath, migrationsPath)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to run migrations")
	}
	manager.Up()
	log.Info().Msg("Successfully run migrations.")

	if err := transport.Init(ctx, cfg, log); err != nil {
		log.Warn().Err(err).Msg("Failed to initialize transport layer; starting in offline mode")
	}

	// Инициализируем сервис синхронизации
	syncService := services.NewSyncService(store, log)

	appCtx, cancel := context.WithCancel(ctx)

	go func() {
		log.Info().Msg("Starting background synchronization process")
		ticker := time.NewTicker(10 * time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				log.Info().Msg("Running scheduled data synchronization")
				if err := syncService.Sync(appCtx); err != nil {
					log.Warn().Err(err).Msg("Scheduled data synchronization failed")
				}
			case <-appCtx.Done():
				log.Info().Msg("Stopping background synchronization process")
				return
			}
		}
	}()

	app := &App{
		Logger:  log,
		Storage: store,
	}

	// 1. Создаем модель без указателя на программу.
	rootModel := tui.NewRootModel(store, cfg, syncService, log)
	// 2. Создаем программу с этой моделью.
	p := tea.NewProgram(rootModel)
	// 3. Теперь, когда программа создана, устанавливаем указатель на нее в модели.
	rootModel.SetProgram(p)
	app.Tui = &rootModel

	if _, err := p.Run(); err != nil {
		log.Error().Err(err).Msg("Alas, there's been an error:")
		cancel()
		os.Exit(1)
	}
	cancel()

	return app
}

func (a *App) Stop() {
	a.Logger.Info().Msg("До свидания!")
	a.Storage.Close()
}
