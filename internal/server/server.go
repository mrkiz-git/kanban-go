package server

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/mrkiz-git/kanba-go/internal/auth"
	"github.com/mrkiz-git/kanba-go/internal/config"
	"github.com/mrkiz-git/kanba-go/internal/domain"
	"github.com/mrkiz-git/kanba-go/internal/handler"
	appmiddleware "github.com/mrkiz-git/kanba-go/internal/middleware"
	"github.com/mrkiz-git/kanba-go/internal/logging"
	"github.com/mrkiz-git/kanba-go/internal/store"
)

type Dependencies struct {
	Users  *store.UserStore
	Tokens *auth.TokenService
}

func New(cfg config.Config, deps Dependencies, logger *logging.Logger) *http.Server {
	if logger == nil {
		logger = logging.Default()
	}

	authHandler := handler.NewAuthHandler(deps.Users, deps.Tokens, cfg.SecureCookie)

	r := chi.NewRouter()
	// RealIP is intentionally omitted — chi's default trusts all proxies; add only behind a trusted reverse proxy.
	r.Use(middleware.RequestID)
	r.Use(requestLogger(logger))
	r.Use(middleware.Recoverer)

	r.Route("/api", func(r chi.Router) {
		r.Get("/health", handler.Health)

		r.Route("/auth", func(r chi.Router) {
			r.Post("/register", authHandler.Register)
			r.Post("/login", authHandler.Login)

			r.Group(func(r chi.Router) {
				r.Use(appmiddleware.Auth(appmiddleware.AuthConfig{
					Tokens: deps.Tokens,
					Users:  deps.Users,
				}))
				r.Get("/me", authHandler.Me)
				r.Post("/refresh", authHandler.Refresh)
				r.Post("/logout", authHandler.Logout)
			})
		})

		r.Group(func(r chi.Router) {
			r.Use(appmiddleware.Auth(appmiddleware.AuthConfig{
				Tokens: deps.Tokens,
				Users:  deps.Users,
			}))
			r.Get("/boards", func(w http.ResponseWriter, r *http.Request) {
				handler.WriteJSON(w, http.StatusOK, map[string][]any{"boards": []any{}})
			})
		})

		r.Route("/admin", func(r chi.Router) {
			r.Use(appmiddleware.Auth(appmiddleware.AuthConfig{
				Tokens: deps.Tokens,
				Users:  deps.Users,
			}))
			r.Use(appmiddleware.RequireRole(domain.RoleAdmin))
			r.Get("/users", func(w http.ResponseWriter, r *http.Request) {
				handler.WriteJSON(w, http.StatusOK, map[string][]any{"users": []any{}})
			})
		})
	})

	r.Handle("/*", handler.Static(cfg.StaticDir))

	return &http.Server{
		Addr:         cfg.Addr(),
		Handler:      r,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
}

func requestLogger(logger *logging.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			next.ServeHTTP(ww, r)

			logger.Debug(
				"request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", ww.Status(),
				"bytes", ww.BytesWritten(),
				"duration", time.Since(start).String(),
				"request_id", middleware.GetReqID(r.Context()),
			)
		})
	}
}
