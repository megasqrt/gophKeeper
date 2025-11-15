package app

// План реализации:
// 1. Определить структуру приложения.
// 2. Реализовать конструктор для создания экземпляра приложения.
// 3. Реализовать методы для запуска и остановки приложения.
import (
	"context"
	"gophKeeper/internal/config"
	"gophKeeper/internal/helper"
	"gophKeeper/internal/services"
	"gophKeeper/internal/storage/postgres"
	"gophKeeper/pkg/logger"
	"gophKeeper/pkg/migrations"
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

func NewApp(ctx context.Context) *App {

	cfg := config.NewConfig()
	log := logger.NewZerologLogger()

	helper.BuildInfoPrint()

	//var storage store.Storage
	var err error

	db, err := sqlx.ConnectContext(ctx, "pgx", cfg.DatabaseDSN)
	if err != nil && cfg.DatabaseDSN != "" {
		log.Fatal().Err(err).Msg("Failed to connect to the database")
	}
	defer db.Close() //на всякий случай

	log.Info().Msg("Successfully create database storage.")

	err = migrations.NewMigration(cfg.DatabaseDSN).Up("file://internal/storage/postgres/migrations")
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to run migrations")
	}
	log.Info().Msg("Successfully run migrations.")

	userRepo := postgres.NewUserRepository(db)

	_ = userRepo // Используем переменную, чтобы избежать ошибки компиляции

	// 1. Инициализируем gRPC сервер
	// TODO: Заменить nil на реальную реализацию сервиса аутентификации
	server, err := services.New(log, nil, cfg.ServerAddress)
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
