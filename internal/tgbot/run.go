package tgbot

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func Run() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	log.Println("smetacheck telegram bot started")
	<-ctx.Done()
	log.Println("smetacheck telegram bot stopped")
}
