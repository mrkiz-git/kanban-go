package server

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/mrkiz-git/kanba-go/internal/config"
	"github.com/mrkiz-git/kanba-go/internal/handler"
)

func New(cfg config.Config) *http.Server {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Route("/api", func(r chi.Router) {
		r.Get("/health", handler.Health)
	})

	return &http.Server{
		Addr:    cfg.Addr(),
		Handler: r,
	}
}
