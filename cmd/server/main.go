package main

import (
	"context"
	"os/signal"
	"syscall"

	"gophKeeper/internal/app"

)

func main() {

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	defer stop()

	app :=app.NewApp(ctx)

	app.Server.Start()
	
	<-ctx.Done()

	app.Server.Stop()

}
