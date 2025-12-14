package app

// План реализации:
// 1. Определить структуру приложения.
// 2. Реализовать конструктор для создания экземпляра приложения.
// 3. Реализовать методы для запуска и остановки приложения.
import (
	"context"
	"gophKeeper/pkg/logger"
	"gophKeeper/pkg/migrations/pg"
	"gophKeeper/server/internal/config"
	"gophKeeper/server/internal/helper"
	"gophKeeper/server/internal/services"
	"gophKeeper/server/internal/storage/postgres"
	"net/http"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog"
)

type App struct {
	DB     *sqlx.DB
	Server *services.Server
	Logger *zerolog.Logger
}

func (a *App) Stop() {
	a.Server.Stop()
	a.DB.Close()
}

func NewApp(ctx context.Context) *App {

	cfg := config.NewConfig()
	log := logger.NewСonsoleLogger()

	helper.BuildInfoPrint()

	//var storage store.Storage
	var err error

	db, err := sqlx.ConnectContext(ctx, "pgx", cfg.DatabaseURL)
	if err != nil && cfg.DatabaseURL != "" {
		log.Fatal().Err(err).Msg("Failed to connect to the database")
	}

	log.Info().Msg("Successfully create database storage.")

	err = migrations.NewMigration(cfg.DatabaseURL).Up(cfg.MigrationsPath)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to run migrations")
	}
	log.Info().Msg("Successfully run migrations.")

	userRepo := postgres.NewUserRepository(db)
	deviceRepo := postgres.NewDeviceRepository(db)
	noteRepo := postgres.NewTextDataRepository(db)
	cardRepo := postgres.NewCardRepository(db)
	passRepo := postgres.NewPasswordRepository(db)
	fileRepo := postgres.NewFileRepository(db)

	// Инициализируем сервис аутентификации с репозиторием пользователей.
	authService := services.NewService(log, userRepo, deviceRepo, cfg)
	noteService := services.NewNoteService(log, noteRepo)
	cardService := services.NewCardService(log, cardRepo)
	passService := services.NewPasswordService(log, passRepo)
	fileService := services.NewFileService(log, fileRepo)

	// 1. Инициализируем gRPC сервер
	// Передаем нашу реализацию сервиса.
	server, err := services.New(log, authService, noteService, cardService, passService, fileService, cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create gRPC server")
	}

	go func() {
		if err := server.Start(); err != nil {
			log.Fatal().Err(err).Msg("Failed to start gRPC server")
		}
	}()

	go func() {
		log.Info().Msg("Starting pprof server on :6060")
		if err := http.ListenAndServe(":6060", nil); err != nil {
			log.Error().Err(err).Msg("pprof server error")
		}
	}()

	return &App{
		DB:     db,
		Server: server,
		Logger: &log,
	}
}
