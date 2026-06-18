package auth

import (
	"context"

	"github.com/mrkiz-git/kanba-go/internal/domain"
)

const boardAccessContextKey contextKey = 2

type BoardAccess struct {
	BoardID    string
	Permission domain.BoardPermission
}

func WithBoardAccess(ctx context.Context, access BoardAccess) context.Context {
	return context.WithValue(ctx, boardAccessContextKey, access)
}

func BoardAccessFromContext(ctx context.Context) (BoardAccess, bool) {
	access, ok := ctx.Value(boardAccessContextKey).(BoardAccess)
	return access, ok
}
