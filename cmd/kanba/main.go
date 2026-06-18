package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/mrkiz-git/kanba-go/internal/config"
	"github.com/mrkiz-git/kanba-go/internal/server"
)

func main() {
	cfg := config.Load()
	srv := server.New(cfg)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	go func() {
		<-ctx.Done()
		shutCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutCtx); err != nil {
			log.Printf("shutdown error: %v", err)
		}
	}()

	log.Printf("kanba %s listening on %s", config.Version, cfg.Addr())
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}
