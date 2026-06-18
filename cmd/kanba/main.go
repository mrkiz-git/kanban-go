package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/mrkiz-git/kanba-go/internal/auth"
	"github.com/mrkiz-git/kanba-go/internal/config"
	"github.com/mrkiz-git/kanba-go/internal/logging"
	"github.com/mrkiz-git/kanba-go/internal/server"
	"github.com/mrkiz-git/kanba-go/internal/store"
)

func main() {
	cfg := config.Load()

	logger, err := logging.NewFromConfig(cfg.LogLevel, cfg.LogFile)
	if err != nil {
		panic(err)
	}
	logging.SetDefault(logger)

	if cfg.JWTSecret == config.DefaultJWTSecret() {
		logger.Info("using default JWT secret; set JWT_SECRET in production")
	}
	if cfg.AdminPassword == config.DefaultAdminPassword() {
		logger.Info("using default admin password; set ADMIN_PASSWORD in production")
	}

	if err := ensureParentDir(cfg.DatabasePath); err != nil {
		logger.Error("database path", "error", err)
		panic(err)
	}

	db, err := store.Open(cfg.DatabasePath)
	if err != nil {
		logger.Error("open database", "error", err)
		panic(err)
	}
	defer db.Close()

	if err := store.Migrate(db); err != nil {
		logger.Error("migrate database", "error", err)
		panic(err)
	}

	users := store.NewUserStore(db)
	boards := store.NewBoardStore(db)
	adminHash, err := auth.HashPassword(cfg.AdminPassword)
	if err != nil {
		logger.Error("hash admin password", "error", err)
		panic(err)
	}
	if err := store.SeedAdmin(context.Background(), users, cfg.AdminEmail, adminHash, cfg.AdminName); err != nil {
		logger.Error("seed admin", "error", err)
		panic(err)
	}

	tokens := auth.NewTokenService([]byte(cfg.JWTSecret))
	deps := server.Dependencies{
		Users:  users,
		Boards: boards,
		Tokens: tokens,
	}

	srv := server.New(cfg, deps, logger)

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

func ensureParentDir(path string) error {
	dir := filepath.Dir(path)
	if dir == "" || dir == "." {
		return nil
	}
	return os.MkdirAll(dir, 0o755)
}
