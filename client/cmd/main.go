package main

import (
	"context"
	"fmt"
	"gophKeeper/client/internal/app"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	defer stop()

	// Инициализируем приложение
	appInstance, err := app.NewApp(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize application: %v\n", err)
		os.Exit(1)
	}

	// Запускаем приложение (блокирует выполнение до завершения TUI)
	if err := appInstance.Run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Application error: %v\n", err)
		appInstance.Stop()
		os.Exit(1)
	}

	// Graceful Shutdown
	appInstance.Stop()
}
