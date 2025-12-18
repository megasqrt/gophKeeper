package logger

import (
	"fmt"
	"os"
	"time"

	"github.com/rs/zerolog"
)

func NewСonsoleLogger() zerolog.Logger {
	// Настройка формата времени
	zerolog.TimeFieldFormat = time.RFC3339Nano

	// Многоуровневый вывод
	multi := zerolog.MultiLevelWriter(
		zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339, NoColor: false},
		os.Stdout,
	)

	return zerolog.New(multi).
		With().
		Timestamp().
		Caller().
		Logger().
		Level(zerolog.InfoLevel)
}

func NewFileLogger(logPath string) *zerolog.Logger {
	// Настройка формата времени
	zerolog.TimeFieldFormat = time.RFC3339Nano

	// Открываем лог-файл для добавления. Создаем, если не существует.
	logFile, err := os.OpenFile(
		logPath,
		os.O_APPEND|os.O_CREATE|os.O_WRONLY,
		0664,
	)
	if err != nil {
		// Если не удалось открыть файл, логируем ошибку в консоль и продолжаем только с консольным выводом.
		fmt.Printf("Failed to open log file, falling back to console only: %v\n", err)
	}

	fileWriter := zerolog.ConsoleWriter{Out: logFile, TimeFormat: time.RFC3339, NoColor: true} // NoColor для файла

	// Многоуровневый вывод
	log := zerolog.MultiLevelWriter(fileWriter)

	logger := zerolog.New(log).
		With().
		Timestamp().
		Caller().
		Logger().
		Level(zerolog.DebugLevel)

	return &logger
}

// Get возвращает синглтон-экземпляр zerolog.Logger.
// Логгер инициализируется только при первом вызове.
// func Get() zerolog.Logger {
// 	once.Do(func() {
// 		log = zerolog.New(os.Stdout).With().Timestamp().Logger()
// 	})
// 	return log
// }
