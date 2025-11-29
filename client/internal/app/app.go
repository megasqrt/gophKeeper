// План реализации:
// 1. Определить структуру приложения.
// 2. Реализовать конструктор для создания экземпляра приложения.
// 3. Реализовать методы для запуска и остановки приложения.
package app

import (
	"context"
	"gophKeeper/client/internal/config"
	"gophKeeper/client/internal/delivery/tui"
	"gophKeeper/client/internal/storage"
	"gophKeeper/client/internal/transport/grpc"
	"gophKeeper/pkg/logger"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/docker/docker/client"
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

	log := logger.NewZerologLogger()

	cfg, err := config.Init()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize config")
	}

	// Инициализируем хранилище.
	store, err := storage.NewBboltStorage(cfg.DBPath)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize storage")
	}

	client, err:=grpc.NewClient(ctx,cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize grpc client")
	}

	

	// Используем новую корневую модель
	rootModel := tui.NewRootModel(store, cfg, &log)

	p := tea.NewProgram(rootModel)
	if _, err := p.Run(); err != nil {
		log.Debug().Err(err).Msg("Alas, there's been an error:")
		os.Exit(1)
	}

	return &App{
		Logger:  &log,
		Storage: store,
		Tui:     &rootModel,
	}
}

func (a *App) Stop() {
	a.Logger.Info().Msg("До свидания!")
	a.Storage.Close()
}
