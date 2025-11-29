package logger

import (
	"os"
	"time"
	"github.com/rs/zerolog"
)

func NewZerologLogger() zerolog.Logger {
	// Настройка формата времени
	zerolog.TimeFieldFormat = time.RFC3339Nano

	// Многоуровневый вывод
	multi := zerolog.MultiLevelWriter(
	//	zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339, NoColor: false},
		zerolog.ConsoleWriter{Out: os.NewFile(0,"log.txt"), TimeFormat: time.RFC3339, NoColor: false},
	
		os.Stdout,
	)

	return zerolog.New(multi).
		With().
		Timestamp().
		Caller().
		Logger().
		Level(zerolog.InfoLevel)
}

// Get возвращает синглтон-экземпляр zerolog.Logger.
// Логгер инициализируется только при первом вызове.
// func Get() zerolog.Logger {
// 	once.Do(func() {
// 		log = zerolog.New(os.Stdout).With().Timestamp().Logger()
// 	})
// 	return log
// }
