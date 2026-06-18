package auth

import (
	"context"

	"github.com/mrkiz-git/kanba-go/internal/domain"
)

type contextKey int

const userContextKey contextKey = 1

type AuthUser struct {
	ID    string
	Email string
	Role  domain.UserRole
}

func WithUser(ctx context.Context, user AuthUser) context.Context {
	return context.WithValue(ctx, userContextKey, user)
}

func UserFromContext(ctx context.Context) (AuthUser, bool) {
	user, ok := ctx.Value(userContextKey).(AuthUser)
	return user, ok
}
