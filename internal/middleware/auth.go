package middleware

import (
	"database/sql"
	"errors"
	"net/http"
	"strings"

	"github.com/mrkiz-git/kanba-go/internal/auth"
	"github.com/mrkiz-git/kanba-go/internal/domain"
	"github.com/mrkiz-git/kanba-go/internal/handler"
	"github.com/mrkiz-git/kanba-go/internal/store"
)

var (
	errNoToken       = errors.New("no token")
	errMalformedAuth = errors.New("malformed authorization")
)

type AuthConfig struct {
	Tokens *auth.TokenService
	Users  *store.UserStore
}

func Auth(cfg AuthConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenStr, err := extractToken(r)
			if errors.Is(err, errNoToken) || errors.Is(err, errMalformedAuth) {
				handler.WriteUnauthorized(w)
				return
			}
			if err != nil {
				handler.WriteAPIError(w, http.StatusInternalServerError, "internal_error", "internal error", nil)
				return
			}

			claims, err := cfg.Tokens.Parse(tokenStr)
			if err != nil {
				handler.WriteUnauthorized(w)
				return
			}

			user, err := cfg.Users.GetByID(r.Context(), claims.UserID)
			if errors.Is(err, sql.ErrNoRows) {
				handler.WriteUnauthorized(w)
				return
			}
			if err != nil {
				handler.WriteAPIError(w, http.StatusInternalServerError, "internal_error", "internal error", nil)
				return
			}
			if user.Status == domain.StatusSuspended {
				handler.WriteForbidden(w)
				return
			}

			authUser := auth.AuthUser{
				ID:    user.ID,
				Email: user.Email,
				Role:  user.Role,
			}
			ctx := auth.WithUser(r.Context(), authUser)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func RequireRole(roles ...domain.UserRole) func(http.Handler) http.Handler {
	allowed := make(map[domain.UserRole]bool, len(roles))
	for _, role := range roles {
		allowed[role] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authUser, ok := auth.UserFromContext(r.Context())
			if !ok {
				handler.WriteUnauthorized(w)
				return
			}
			if !allowed[authUser.Role] {
				handler.WriteForbidden(w)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func extractToken(r *http.Request) (string, error) {
	if header := r.Header.Get("Authorization"); header != "" {
		const prefix = "Bearer "
		if !strings.HasPrefix(header, prefix) {
			return "", errMalformedAuth
		}
		token := strings.TrimSpace(header[len(prefix):])
		if token == "" {
			return "", errMalformedAuth
		}
		return token, nil
	}

	if cookie, err := r.Cookie(auth.TokenCookieName); err == nil && cookie.Value != "" {
		return cookie.Value, nil
	}

	return "", errNoToken
}
