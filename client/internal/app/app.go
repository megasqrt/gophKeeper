// План реализации:
// 1. Определить структуру приложения.
// 2. Реализовать конструктор для создания экземпляра приложения.
// 3. Реализовать методы для запуска и остановки приложения.
package app

import (
	"context"
	"fmt"
	"gophKeeper/client/internal/config"
	"gophKeeper/client/internal/delivery/tui"
	"gophKeeper/client/internal/services"
	"gophKeeper/client/internal/storage"
	"gophKeeper/client/internal/transport"
	logger "gophKeeper/pkg/logger"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/rs/zerolog"
)

type App struct {
	Logger  *zerolog.Logger
	Storage *storage.BboltStorage
	Tui     *tui.RootModel
}

func NewApp(ctx context.Context) *App {
	// Инициализируем конфигурацию.
	// Это создаст ~/.gophkeeper/gpk.conf, если его нет.

	cfg, err := config.Init()
	if err != nil {
		// Если конфиг не загрузился, логгер еще не создан. Паникуем.
		panic(fmt.Sprintf("Failed to initialize config: %v", err))
	}

	log := logger.NewFileLoger(cfg.LogPath)
	log.Info().Msg("Client application starting")
	log.Info().Str("log_path", cfg.LogPath).Msg("Logging to file")

	// Инициализируем хранилище.
	store, err := storage.NewBboltStorage(cfg.DBPath, *log)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize storage")
	}

	// Initialize the transport layer.
	if err := transport.Init(ctx, cfg, log); err != nil {
		log.Warn().Err(err).Msg("Failed to initialize transport layer; starting in offline mode")
	}

	// Инициализируем сервис синхронизации
	syncService := services.NewSyncService(store, log)

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
		os.Exit(1)
	}

	return app
}

func (a *App) Stop() {
	a.Logger.Info().Msg("До свидания!")
	a.Storage.Close()
}
