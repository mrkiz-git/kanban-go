package main

import (
	"context"
	"errors"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/mrkiz-git/kanba-go/internal/config"
	"github.com/mrkiz-git/kanba-go/internal/logging"
	"github.com/mrkiz-git/kanba-go/internal/server"
)

func main() {
	cfg := config.Load()

	logger, err := logging.NewFromConfig(cfg.LogLevel, cfg.LogFile)
	if err != nil {
		panic(err)
	}
	logging.SetDefault(logger)

	srv := server.New(cfg, logger)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	go func() {
		<-ctx.Done()
		logger.Info("shutdown signal received")
		shutCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutCtx); err != nil {
			logger.Error("shutdown failed", "error", err)
		}
	}()

	logger.Info("server starting", "version", config.Version, "addr", cfg.Addr(), "log_level", cfg.LogLevel.String())
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Error("server exited", "error", err)
	}
	logger.Info("server stopped")
}
