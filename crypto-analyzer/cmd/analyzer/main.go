package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/NursultanKoshoev11/SmetaCheck/crypto-analyzer/internal/binance"
	"github.com/NursultanKoshoev11/SmetaCheck/crypto-analyzer/internal/config"
	"github.com/NursultanKoshoev11/SmetaCheck/crypto-analyzer/internal/httpapi"
	"github.com/NursultanKoshoev11/SmetaCheck/crypto-analyzer/internal/service"
	"github.com/NursultanKoshoev11/SmetaCheck/crypto-analyzer/internal/store"
	"github.com/NursultanKoshoev11/SmetaCheck/crypto-analyzer/internal/telegram"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("configuration error", "error", err)
		os.Exit(1)
	}
	level := slog.LevelInfo
	if cfg.LogLevel == "debug" {
		level = slog.LevelDebug
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
	slog.SetDefault(logger)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	database, err := store.New(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("database startup failed", "error", err)
		os.Exit(1)
	}
	defer database.Close()

	market := binance.New(cfg.BinanceSpotBaseURL, cfg.BinanceFuturesURL, cfg.HTTPTimeout, cfg.RequestDelay)
	bot := telegram.New(cfg.TelegramToken, cfg.TelegramChatID, cfg.HTTPTimeout)
	app := service.New(cfg, database, market, bot, logger)
	api := httpapi.New(cfg.HTTPAddr, cfg.APIKey, database, logger)

	app.Start(ctx)
	apiErrors := make(chan error, 1)
	go func() { apiErrors <- api.Start() }()

	select {
	case <-ctx.Done():
		logger.Info("shutdown requested")
	case apiErr := <-apiErrors:
		if apiErr != nil {
			logger.Error("HTTP server stopped", "error", apiErr)
		}
		stop()
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if err := api.Shutdown(shutdownCtx); err != nil {
		logger.Error("HTTP shutdown failed", "error", err)
	}
}
