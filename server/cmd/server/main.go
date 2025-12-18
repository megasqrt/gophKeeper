package main

import (
	"context"
	"os/signal"
	"syscall"

	"gophKeeper/server/internal/app"
	"gophKeeper/server/internal/config"
)

func main() {

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	defer stop()

	cfg := config.NewConfig()
	app := app.NewApp(ctx, cfg)

	<-ctx.Done()

	// Graceful Shutdown.
	app.Stop()

}
